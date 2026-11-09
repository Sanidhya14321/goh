// Package mcpclient admits explicitly configured local stdio MCP servers.
// Discovery and calls use the official SDK; no remote transports, sampling,
// executable installation, or automatic call replay are supported.
package mcpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const maxFrame = 1 << 20
const maxResult = 64 << 10
const callTimeout = 60 * time.Second

var component = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
var ErrPermission = errors.New("MCP server requires permission")

type binding struct {
	server     string
	session    *mcp.ClientSession
	process    *process
	original   string
	schema     *jsonschema.Schema
	definition core.Tool
}
type Manager struct {
	config    map[string]core.MCPConfig
	project   string
	approve   func(context.Context, core.Permission) (core.Decision, error)
	mu        sync.Mutex
	started   bool
	connected bool
	closed    bool
	bindings  map[string]binding
	processes []*process
	failures  []Failure
	// resourceServers maps servers that offer resources to their sessions.
	resourceServers map[string]*mcp.ClientSession
	// reconnects counts restarts per server; reconnectMu serialises them.
	reconnects  map[string]int
	reconnectMu sync.Mutex
}

// maxReconnects bounds automatic restarts of one server per run.
const maxReconnects = 3

func New(config map[string]core.MCPConfig, project string, approve func(context.Context, core.Permission) (core.Decision, error)) *Manager {
	cfg := make(map[string]core.MCPConfig, len(config))
	for k, v := range config {
		v.Args = append([]string(nil), v.Args...)
		env := map[string]string{}
		for k, v := range v.Env {
			env[k] = v
		}
		v.Env = env
		cfg[k] = v
	}
	return &Manager{config: cfg, project: project, approve: approve, bindings: map[string]binding{}}
}
func validComponent(name string) bool {
	return component.MatchString(name) && !strings.Contains(name, "__")
}

// Connect initializes once and is idempotent after success. A failed connection
// is cleaned up and cannot be retried on the same Manager.
func (m *Manager) Connect(ctx context.Context) (err error) {
	m.mu.Lock()
	if m.connected && !m.closed {
		m.mu.Unlock()
		return nil
	}
	if m.started || m.closed {
		m.mu.Unlock()
		return errors.New("MCP manager already connected or closed")
	}
	m.started = true
	m.mu.Unlock()
	defer func() {
		if err != nil {
			m.Close()
		} else {
			m.mu.Lock()
			m.connected = true
			m.mu.Unlock()
		}
	}()
	names := make([]string, 0, len(m.config))
	for n := range m.config {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		if e := m.connectServer(ctx, name); e != nil {
			// Approval outcomes and cancellation keep their existing meaning.
			if errors.Is(e, ErrPermission) || ctx.Err() != nil || errors.Is(e, errApproval) {
				return e
			}
			m.mu.Lock()
			m.failures = append(m.failures, Failure{Name: name, Error: e.Error()})
			m.mu.Unlock()
		}
	}
	return nil
}

var errApproval = errors.New("MCP approval failed")

// Failure records one optional server that could not start. Its process is
// stopped and its tools are absent; other servers and built-in tools remain.
type Failure = core.IntegrationFailure

// Failures lists servers skipped during Connect.
func (m *Manager) Failures() []Failure {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Failure(nil), m.failures...)
}

func (m *Manager) dropProcess(p *process) {
	if p == nil {
		return
	}
	p.stop()
	m.mu.Lock()
	for i, q := range m.processes {
		if q == p {
			m.processes = append(m.processes[:i], m.processes[i+1:]...)
			break
		}
	}
	m.mu.Unlock()
}

func (m *Manager) connectServer(ctx context.Context, name string) (err error) {
	var p *process
	defer func() {
		if err != nil && !errors.Is(err, ErrPermission) {
			m.dropProcess(p)
		}
	}()
	if !validComponent(name) {
		return fmt.Errorf("MCP server name %q must use letters, digits, underscore or hyphen without double underscores", name)
	}
	cfg := m.config[name]
	if (cfg.Command == "") == (cfg.URL == "") {
		return fmt.Errorf("MCP server %s: set either command or url", name)
	}
	var description []byte
	if cfg.URL != "" {
		if err := checkRemoteURL(cfg.URL); err != nil {
			return fmt.Errorf("MCP server %s: %w", name, err)
		}
		headers := make([]string, 0, len(cfg.HeadersEnv)+len(cfg.Headers))
		for h := range cfg.HeadersEnv {
			headers = append(headers, h)
		}
		for h := range cfg.Headers {
			headers = append(headers, h)
		}
		sort.Strings(headers)
		// Header values are never shown; only their names.
		description, _ = json.Marshal(map[string]any{"url": cfg.URL, "headers": headers, "initialization_timeout_seconds": 15})
	} else {
		description, _ = json.Marshal(map[string]any{"command": cfg.Command, "args": cfg.Args, "cwd": m.project, "initialization_timeout_seconds": 15})
	}
	permission := core.Permission{Tool: "mcp_start:" + name, Preview: string(description), Arguments: description}
	if m.approve == nil {
		return ErrPermission
	}
	decision, e := m.approve(ctx, permission)
	if e != nil {
		return fmt.Errorf("%w: %w", errApproval, e)
	}
	if !decision.Allow {
		return fmt.Errorf("%w: %s", ErrPermission, name)
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	if cfg.URL != "" {
		p, e = &process{done: make(chan struct{})}, nil
	} else {
		p, e = startProcess(cfg, m.project)
	}
	if e != nil {
		p = nil
		return fmt.Errorf("MCP server %s: %w", name, e)
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		p.stop()
		p = nil
		return fmt.Errorf("%w: MCP manager closed during initialization", errApproval)
	}
	m.processes = append(m.processes, p)
	m.mu.Unlock()
	initCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	client := mcp.NewClient(&mcp.Implementation{Name: "independent-harness", Version: "0.1.0"}, &mcp.ClientOptions{Capabilities: &mcp.ClientCapabilities{}, MultiRoundTrip: &mcp.MultiRoundTripOptions{Disabled: true}})
	var transport mcp.Transport = &mcp.IOTransport{Reader: p.stdout, Writer: p.stdin, MaxLineLength: maxFrame}
	if cfg.URL != "" {
		httpClient, err := remoteClient(cfg)
		if err != nil {
			cancel()
			return fmt.Errorf("MCP server %s: %w", name, err)
		}
		transport = &mcp.StreamableClientTransport{Endpoint: cfg.URL, HTTPClient: httpClient, MaxRetries: -1, DisableStandaloneSSE: true}
	}
	session, e := client.Connect(initCtx, transport, nil)
	if e != nil {
		cancel()
		return fmt.Errorf("MCP server %s initialization failed: %w", name, e)
	}
	p.setSession(session)
	if init := session.InitializeResult(); init != nil && init.Capabilities != nil && init.Capabilities.Resources != nil {
		m.mu.Lock()
		if m.resourceServers == nil {
			m.resourceServers = map[string]*mcp.ClientSession{}
		}
		m.resourceServers[name] = session
		m.mu.Unlock()
	}
	count := 0
	discovered := map[string]binding{}
	for tool, e := range session.Tools(initCtx, nil) {
		if e != nil {
			cancel()
			return fmt.Errorf("MCP server %s discovery failed: %w", name, e)
		}
		count++
		if count > 128 {
			cancel()
			return fmt.Errorf("MCP server %s exceeds 128-tool discovery limit", name)
		}
		if !validComponent(tool.Name) {
			cancel()
			return fmt.Errorf("MCP tool %q uses an unsupported name; alpha supports letters, digits, underscore and hyphen without double underscores", tool.Name)
		}
		full := "mcp__" + name + "__" + tool.Name
		if len(full) > 64 {
			cancel()
			return fmt.Errorf("MCP namespaced tool %q exceeds 64 bytes", full)
		}
		if _, exists := discovered[full]; exists {
			cancel()
			return fmt.Errorf("MCP duplicate tool name %q", full)
		}
		schema, ok := tool.InputSchema.(map[string]any)
		if !ok {
			cancel()
			return fmt.Errorf("MCP tool %s requires an object input schema", full)
		}
		if e := rejectReferences(schema); e != nil {
			cancel()
			return fmt.Errorf("MCP tool %s: %w", full, e)
		}
		compiler := jsonschema.NewCompiler()
		compiler.UseLoader(noLoader{})
		const schemaURL = "https://harness.invalid/schema"
		if e := compiler.AddResource(schemaURL, schema); e != nil {
			cancel()
			return fmt.Errorf("MCP tool %s schema: %w", full, e)
		}
		compiled, e := compiler.Compile(schemaURL)
		if e != nil {
			cancel()
			return fmt.Errorf("MCP tool %s unsupported schema: %w", full, e)
		}
		description := tool.Description
		if len(description) > 8192 {
			description = description[:8192]
		}
		discovered[full] = binding{server: name, session: session, process: p, original: tool.Name, schema: compiled, definition: core.NewTool(full, description, schema)}
	}
	cancel()
	m.mu.Lock()
	for n, b := range discovered {
		if _, exists := m.bindings[n]; exists {
			m.mu.Unlock()
			return fmt.Errorf("MCP tool namespace collision %q", n)
		}
		m.bindings[n] = b
	}
	m.mu.Unlock()
	return nil
}

type noLoader struct{}

func (noLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("external JSON Schema loading is disabled: %s", url)
}
func rejectReferences(value any) error {
	switch v := value.(type) {
	case map[string]any:
		for k, x := range v {
			if k == "$ref" || k == "$dynamicRef" || k == "$recursiveRef" {
				return errors.New("JSON Schema references are unsupported in this alpha; supply an inline schema")
			}
			if err := rejectReferences(x); err != nil {
				return err
			}
		}
	case []any:
		for _, x := range v {
			if err := rejectReferences(x); err != nil {
				return err
			}
		}
	}
	return nil
}
func (m *Manager) Definitions() []core.Tool {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.bindings))
	for name := range m.bindings {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]core.Tool, 0, len(names))
	for _, name := range names {
		out = append(out, m.bindings[name].definition)
	}
	if len(m.resourceServers) > 0 {
		out = append(out, resourceDefinitions()...)
	}
	return out
}
func (m *Manager) Has(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if IsResourceTool(name) {
		return len(m.resourceServers) > 0
	}
	_, ok := m.bindings[name]
	return ok
}
func (m *Manager) get(name string) (binding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return binding{}, errors.New("MCP manager closed")
	}
	b, ok := m.bindings[name]
	if !ok {
		return b, fmt.Errorf("unknown MCP tool %q", name)
	}
	return b, nil
}
func (m *Manager) Validate(name string, args json.RawMessage) error {
	if IsResourceTool(name) {
		_, err := decodeResourceArgs(name, args)
		return err
	}
	b, e := m.get(name)
	if e != nil {
		return e
	}
	if len(args) > 64<<10 {
		return errors.New("MCP arguments exceed 64 KiB")
	}
	if e = rejectDuplicateKeys(args); e != nil {
		return fmt.Errorf("invalid MCP arguments: %w", e)
	}
	var value map[string]any
	if e = json.Unmarshal(args, &value); e != nil {
		return fmt.Errorf("invalid MCP arguments: %w", e)
	}
	if value == nil {
		return errors.New("MCP arguments must be an object")
	}
	return b.schema.Validate(value)
}

// Execute assumes the caller has approved the exact call. An interruption is
// reported as uncertain; neither this package nor the SDK repeats the call.
func (m *Manager) Execute(ctx context.Context, name string, args json.RawMessage) tools.Result {
	fail := func(err error) tools.Result { return tools.Result{Content: err.Error(), Error: true} }
	if IsResourceTool(name) {
		return m.executeResource(ctx, name, args)
	}
	if e := m.Validate(name, args); e != nil {
		return fail(e)
	}
	b, e := m.get(name)
	if e != nil {
		return fail(e)
	}
	if b.process.dead() {
		// A new call to a server that dropped: restart it first. The call
		// that failed earlier is never repeated.
		if e := m.reconnect(ctx, b); e != nil {
			return fail(fmt.Errorf("MCP server %s disconnected and could not be restarted: %w", b.server, e))
		}
		if b, e = m.get(name); e != nil {
			return fail(e)
		}
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	// Send exactly what was validated and approved: the decoded object
	// re-encoded, never the model's original bytes.
	var validated map[string]any
	if e := json.Unmarshal(args, &validated); e != nil {
		return fail(e)
	}
	result, e := b.session.CallTool(ctx, &mcp.CallToolParams{Name: b.original, Arguments: validated})
	if e != nil {
		status := "interrupted"
		message := fmt.Errorf("MCP call failed; outcome may be uncertain and was not replayed: %w", e).Error()
		b.process.markBroken() // the next call reconnects; this one is not replayed
		if ctx.Err() != nil {
			if cleanupErr := b.process.stop(); cleanupErr != nil {
				status = "cleanup_incomplete"
				message += "; " + cleanupErr.Error()
			}
		}
		encoded, _ := json.Marshal(map[string]any{"status": status, "error": message})
		return tools.Result{Content: string(encoded), Error: true}
	}
	if result.NeedsInput() {
		return fail(errors.New("MCP interactive input/sampling is unsupported; operation was not replayed"))
	}
	encoded, e := json.Marshal(result)
	if e != nil {
		return fail(e)
	}
	if len(encoded) > maxResult {
		// The server answered, so the action ran: report it as done (not as
		// an error the model might retry) and keep a bounded excerpt.
		head := string(encoded[:4096])
		for len(head) > 0 && !utf8.ValidString(head) {
			head = head[:len(head)-1]
		}
		out, _ := json.Marshal(map[string]any{"status": "completed_output_truncated", "server_reported_error": result.IsError, "bytes": len(encoded), "head": head,
			"note": "The MCP server completed this call; its output was too large and is truncated. Do not repeat the call to see more output."})
		return tools.Result{Content: string(out), Error: result.IsError}
	}
	return tools.Result{Content: string(encoded), Error: result.IsError}
}

// reconnect stops a dead server, forgets its tools and connects it again
// through the normal approval path. It is bounded per server.
func (m *Manager) reconnect(ctx context.Context, b binding) error {
	m.reconnectMu.Lock()
	defer m.reconnectMu.Unlock()
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return errors.New("MCP manager closed")
	}
	if cur, ok := m.bindings["mcp__"+b.server+"__"+b.original]; ok && cur.process != b.process && !cur.process.dead() {
		m.mu.Unlock()
		return nil // another call already reconnected it
	}
	if m.reconnects == nil {
		m.reconnects = map[string]int{}
	}
	if m.reconnects[b.server] >= maxReconnects {
		m.mu.Unlock()
		return fmt.Errorf("restarted %d times already in this run", maxReconnects)
	}
	m.reconnects[b.server]++
	prefix := "mcp__" + b.server + "__"
	for n := range m.bindings {
		if strings.HasPrefix(n, prefix) {
			delete(m.bindings, n)
		}
	}
	m.mu.Unlock()
	m.dropProcess(b.process)
	return m.connectServer(ctx, b.server)
}

func (m *Manager) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	processes := append([]*process(nil), m.processes...)
	m.mu.Unlock()
	var wg sync.WaitGroup
	errs := make(chan error, len(processes))
	for _, p := range processes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := p.stop(); e != nil {
				errs <- e
			}
		}()
	}
	wg.Wait()
	close(errs)
	var out error
	for e := range errs {
		out = errors.Join(out, e)
	}
	return out
}

type discardBound struct {
	mu   sync.Mutex
	data []byte
}

func (w *discardBound) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := len(p)
	remaining := (16 << 10) - len(w.data)
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		w.data = append(w.data, p...)
	}
	return n, nil
}

type process struct {
	mu       sync.Mutex
	stopping bool
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	done     chan struct{}
	session  *mcp.ClientSession
	once     sync.Once
	stopErr  error
	broken   bool
}

// dead reports that the server can no longer serve calls: it exited, was
// stopped, or a call failed at the transport level.
func (p *process) dead() bool {
	p.mu.Lock()
	gone := p.stopping || p.broken
	p.mu.Unlock()
	if gone {
		return true
	}
	if p.cmd == nil {
		return false
	}
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

func (p *process) markBroken() {
	p.mu.Lock()
	p.broken = true
	p.mu.Unlock()
}

func startProcess(cfg core.MCPConfig, project string) (*process, error) {
	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Dir = project
	platform.NewGroup(cmd)
	cmd.WaitDelay = time.Second
	env := platform.ChildEnvMap() // secrets withheld; cfg.Env adds explicit values
	for k, v := range cfg.Env {
		if strings.ContainsAny(k, "=\x00") || strings.ContainsRune(v, '\x00') {
			return nil, errors.New("invalid MCP environment")
		}
		env[k] = v
	}
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	childIn, parentIn, e := os.Pipe()
	if e != nil {
		return nil, e
	}
	parentOut, childOut, e := os.Pipe()
	if e != nil {
		childIn.Close()
		parentIn.Close()
		return nil, e
	}
	cmd.Stdin = childIn
	cmd.Stdout = childOut
	cmd.Stderr = &discardBound{}
	if e = cmd.Start(); e != nil {
		childIn.Close()
		parentIn.Close()
		parentOut.Close()
		childOut.Close()
		return nil, e
	}
	childIn.Close()
	childOut.Close()
	p := &process{cmd: cmd, stdin: parentIn, stdout: parentOut, done: make(chan struct{})}
	go func() { cmd.Wait(); close(p.done) }()
	return p, nil
}
func (p *process) setSession(session *mcp.ClientSession) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopping {
		session.Close()
		return
	}
	p.session = session
}

func (p *process) stop() error {
	p.once.Do(func() {
		p.mu.Lock()
		p.stopping = true
		session := p.session
		p.mu.Unlock()
		if p.cmd == nil { // remote server: closing the session is the whole cleanup
			if session != nil {
				done := make(chan struct{})
				go func() { session.Close(); close(done) }()
				select {
				case <-done:
				case <-time.After(2 * time.Second):
					p.stopErr = errors.New("MCP cleanup_incomplete: remote session did not close")
				}
			}
			return
		}
		// Closing both pipes also interrupts SDK readers/writers before waiting.
		p.stdin.Close()
		p.stdout.Close()
		pgid := p.cmd.Process.Pid
		if err := platform.StopGroup(pgid, 2*time.Second); err != nil {
			p.stopErr = fmt.Errorf("MCP cleanup_incomplete: %w", err)
		}
		select {
		case <-p.done:
		case <-time.After(time.Second):
			p.stopErr = errors.Join(p.stopErr, errors.New("MCP cleanup_incomplete: process did not settle"))
		}
		if session != nil {
			done := make(chan struct{})
			go func() { session.Close(); close(done) }()
			select {
			case <-done:
			case <-time.After(time.Second):
				p.stopErr = errors.Join(p.stopErr, errors.New("MCP cleanup_incomplete: SDK session did not close"))
			}
		}
	})
	return p.stopErr
}

// rejectDuplicateKeys refuses JSON with a repeated object key at any depth.
// Parsers disagree on which duplicate wins, so the value GoHarness validated
// could differ from the one a server acts on.
func rejectDuplicateKeys(raw []byte) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func() error
	walk = func() error {
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch t {
		case json.Delim('{'):
			seen := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return err
				}
				key, _ := k.(string)
				if seen[key] {
					return fmt.Errorf("duplicate key %q", key)
				}
				seen[key] = true
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		case json.Delim('['):
			for d.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		}
		return nil
	}
	return walk()
}
