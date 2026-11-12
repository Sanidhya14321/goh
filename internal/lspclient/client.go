// Package lspclient provides explicitly admitted local language servers. It
// never installs servers or accepts server-requested edits/commands.
package lspclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
)

const maxFile = 512 << 10
const maxResult = 64 << 10
const operationTimeout = 15 * time.Second

var serverName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
var operations = map[string]string{"goToDefinition": "textDocument/definition", "findReferences": "textDocument/references", "hover": "textDocument/hover", "documentSymbol": "textDocument/documentSymbol", "workspaceSymbol": "workspace/symbol", "goToImplementation": "textDocument/implementation", "diagnostics": "textDocument/diagnostic",
	"goToTypeDefinition": "textDocument/typeDefinition", "prepareCallHierarchy": "textDocument/prepareCallHierarchy",
	"incomingCalls": "callHierarchy/incomingCalls", "outgoingCalls": "callHierarchy/outgoingCalls"}
var capabilities = map[string]string{"goToDefinition": "definitionProvider", "findReferences": "referencesProvider", "hover": "hoverProvider", "documentSymbol": "documentSymbolProvider", "workspaceSymbol": "workspaceSymbolProvider", "goToImplementation": "implementationProvider", "diagnostics": "diagnosticProvider",
	"goToTypeDefinition": "typeDefinitionProvider", "prepareCallHierarchy": "callHierarchyProvider", "incomingCalls": "callHierarchyProvider", "outgoingCalls": "callHierarchyProvider"}

type arguments struct {
	Operation string `json:"operation"`
	FilePath  string `json:"filePath"`
	Line      int    `json:"line"`
	Character int    `json:"character"`
	Query     string `json:"query,omitempty"`
}
type server struct {
	process      *process
	capabilities map[string]json.RawMessage
	documents    map[string]document
	broken       bool
}
type document struct {
	version int
	text    string
}
type Manager struct {
	config    map[string]core.LSPConfig
	project   string
	approve   func(context.Context, core.Permission) (core.Decision, error)
	OnStarted func(name string) error
	// OnStartFailed settles an approved start that left no running process.
	OnStartFailed func(name string, err error) error
	mu            sync.Mutex
	operationMu   sync.Mutex
	servers       map[string]*server
	restarts      map[string]int
	closed        bool
	configErr     error
}

// diagnosticsWait bounds the wait for pushed diagnostics; a slow server is
// reported, not stopped.
const diagnosticsWait = 5 * time.Second

// diagnosticsSettle is how long unversioned pushed diagnostics must go
// without a newer publish before they are taken as describing the current
// text. Servers that support versionSupport (declared at initialize) attach
// the document version and skip the wait.
const diagnosticsSettle = 400 * time.Millisecond

var errDiagnosticsPending = errors.New("LSP server has not published diagnostics for this file yet; try again shortly")

// maxRestarts bounds approved restarts of one disconnected server per run.
const maxRestarts = 2

func New(config map[string]core.LSPConfig, project string, approve func(context.Context, core.Permission) (core.Decision, error)) *Manager {
	m := &Manager{config: map[string]core.LSPConfig{}, project: project, approve: approve, servers: map[string]*server{}}
	root, err := filepath.Abs(project)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		m.configErr = err
	} else {
		m.project = root
	}
	assigned := map[string]string{}
	for name, c := range config {
		if !serverName.MatchString(name) || c.Command == "" || len(c.Extensions) == 0 {
			m.configErr = fmt.Errorf("LSP server %q requires a valid name, command and explicit extensions", name)
		}
		if len(c.LanguageID) > 128 || !utf8.ValidString(c.LanguageID) || strings.ContainsRune(c.LanguageID, '\x00') {
			m.configErr = fmt.Errorf("LSP server %s language_id invalid or exceeds 128 bytes", name)
		}
		c.Args = append([]string(nil), c.Args...)
		c.Extensions = append([]string(nil), c.Extensions...)
		env := map[string]string{}
		for k, v := range c.Env {
			env[k] = v
		}
		c.Env = env
		// Own the JSON settings snapshot; concurrent caller mutation cannot alter it.
		data, e := json.Marshal(c.Initialization)
		if e != nil || len(data) > 64<<10 {
			m.configErr = fmt.Errorf("LSP server %s initialization settings invalid or exceed 64 KiB", name)
		} else {
			json.Unmarshal(data, &c.Initialization)
		}
		for _, extension := range c.Extensions {
			if !strings.HasPrefix(extension, ".") || strings.ContainsAny(extension, "/\\\x00") || len(extension) > 32 {
				m.configErr = fmt.Errorf("LSP server %s extension %q must be a file suffix beginning with a dot", name, extension)
			}
			if prior, ok := assigned[extension]; ok {
				m.configErr = fmt.Errorf("LSP extension %q assigned to both %s and %s", extension, prior, name)
			}
			assigned[extension] = name
		}
		m.config[name] = c
	}
	if len(config) > 32 {
		m.configErr = errors.New("at most 32 configured LSP servers supported")
	}
	return m
}
func (m *Manager) Definitions() []core.Tool {
	if len(m.config) == 0 {
		return nil
	}
	names := make([]string, 0, len(operations))
	for name := range operations {
		names = append(names, name)
	}
	sort.Strings(names)
	schema := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"operation", "filePath", "line", "character"}, "properties": map[string]any{"operation": map[string]any{"type": "string", "enum": names}, "filePath": map[string]any{"type": "string"}, "line": map[string]any{"type": "integer", "minimum": 1}, "character": map[string]any{"type": "integer", "minimum": 1}, "query": map[string]any{"type": "string"}}}
	return []core.Tool{core.NewTool("lsp", "Query explicitly configured local language servers. Requires approval before invoking the tool and before each server starts. File positions are 1-based Unicode character columns, converted to UTF-16 for the server. Operations: definition, references, hover, symbols, implementation and diagnostics.", schema)}
}
func (m *Manager) Has(name string) bool { return name == "lsp" && len(m.config) > 0 }
func (m *Manager) Validate(name string, input json.RawMessage) error {
	if name != "lsp" {
		return fmt.Errorf("unknown LSP tool %q", name)
	}
	if m.configErr != nil {
		return m.configErr
	}
	if len(input) > 64<<10 {
		return errors.New("LSP arguments exceed 64 KiB")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(input, &fields) != nil || fields == nil {
		return errors.New("LSP arguments must be an object")
	}
	for key, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("LSP argument %s cannot be null", key)
		}
	}
	var args arguments
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&args); err != nil {
		return fmt.Errorf("LSP arguments: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("trailing LSP argument content")
	}
	if _, ok := operations[args.Operation]; !ok {
		return fmt.Errorf("unsupported LSP operation %q", args.Operation)
	}
	if args.Line < 1 || args.Character < 1 {
		return errors.New("LSP line and character must be positive 1-based positions")
	}
	if len(args.Query) > 8192 {
		return errors.New("LSP query exceeds 8 KiB")
	}
	_, _, err := m.readFile(args.FilePath)
	if err != nil {
		return err
	}
	_, _, err = m.selectServer(args.FilePath)
	return err
}
func (m *Manager) selectServer(path string) (string, core.LSPConfig, error) {
	extension := filepath.Ext(path)
	chosen := ""
	var selected core.LSPConfig
	longest := 0
	for name, cfg := range m.config {
		for _, suffix := range cfg.Extensions {
			if strings.HasSuffix(filepath.Base(path), suffix) && len(suffix) > longest {
				chosen = name
				selected = cfg
				longest = len(suffix)
			}
		}
	}
	if chosen != "" {
		return chosen, selected, nil
	}
	return "", core.LSPConfig{}, fmt.Errorf("no explicitly configured LSP server for extension %q", extension)
}
func (m *Manager) readFile(path string) (string, string, error) {
	if path == "" || strings.ContainsRune(path, '\x00') {
		return "", "", errors.New("LSP filePath required")
	}
	absolute := path
	if !filepath.IsAbs(path) {
		absolute = filepath.Join(m.project, path)
	}
	relative, err := filepath.Rel(m.project, absolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", errors.New("LSP path escapes project")
	}
	root, err := os.OpenRoot(m.project)
	if err != nil {
		return "", "", err
	}
	defer root.Close()
	current := ""
	for _, part := range strings.Split(filepath.Clean(relative), string(os.PathSeparator)) {
		current = filepath.Join(current, part)
		st, statErr := root.Lstat(current)
		if statErr != nil {
			return "", "", statErr
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return "", "", errors.New("LSP symlink paths are unsupported")
		}
	}
	file, err := root.OpenFile(relative, os.O_RDONLY|platform.NonBlock, 0)
	if err != nil {
		return "", "", fmt.Errorf("LSP project file: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", "", errors.New("LSP requires a regular project file")
	}
	if info.Size() > maxFile {
		return "", "", errors.New("LSP file exceeds 512 KiB")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxFile+1))
	if err != nil {
		return "", "", err
	}
	if len(data) > maxFile || !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return "", "", errors.New("LSP requires UTF-8 text without NUL, up to 512 KiB")
	}
	return filepath.Clean(absolute), string(data), nil
}
func fileURI(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}
func utf16Position(text string, line, column int) (map[string]int, error) {
	// CRLF and bare CR form one editor line break; columns count Unicode scalars.
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n"), "\n")
	if line < 1 || line > len(lines) {
		return nil, errors.New("LSP line is outside file")
	}
	runes := []rune(lines[line-1])
	if column < 1 || column > len(runes)+1 {
		return nil, errors.New("LSP character is outside line")
	}
	units := 0
	for _, r := range runes[:column-1] {
		units += utf16.RuneLen(r)
	}
	return map[string]int{"line": line - 1, "character": units}, nil
}
func (m *Manager) start(ctx context.Context, name string, cfg core.LSPConfig) (*server, string, error) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, "failed", errors.New("LSP manager closed")
	}
	restart := false
	if s := m.servers[name]; s != nil {
		if !s.broken {
			m.mu.Unlock()
			return s, "", nil
		}
		// A disconnected server may be started again, but only through a
		// fresh approval and at most maxRestarts times per run. The failed
		// operation itself is never replayed.
		if m.restarts == nil {
			m.restarts = map[string]int{}
		}
		if m.restarts[name] >= maxRestarts {
			m.mu.Unlock()
			return nil, "failed", errors.New("LSP server disconnected again; restart limit reached for this run")
		}
		m.restarts[name]++
		delete(m.servers, name)
		restart = true
	}
	m.mu.Unlock()
	if m.approve == nil {
		return nil, "needs_permission", errors.New("explicit LSP server start approval required")
	}
	preview, _ := json.Marshal(map[string]any{"command": cfg.Command, "args": cfg.Args, "cwd": m.project, "initialization_timeout_seconds": 15, "restart": restart})
	decision, err := m.approve(ctx, core.Permission{Tool: "lsp_start:" + name, Preview: string(preview), Arguments: preview})
	if err != nil {
		return nil, "needs_permission", err
	}
	if !decision.Allow {
		return nil, "needs_permission", errors.New("LSP server start denied; no process started")
	}
	if err := ctx.Err(); err != nil {
		return nil, "interrupted", err
	}
	p, err := startProcess(cfg, m.project)
	if err != nil {
		// Nothing was spawned, so the effect is certain: no process exists.
		status, failure := m.settleFailedStart(name, nil, fmt.Errorf("approved LSP startup failed: %w", err))
		return nil, status, failure
	}
	s := &server{process: p, documents: map[string]document{}}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		p.stop()
		return nil, "interrupted", errors.New("LSP manager closed during startup")
	}
	m.servers[name] = s
	m.mu.Unlock()
	params := map[string]any{"processId": os.Getpid(), "rootUri": fileURI(m.project), "rootPath": m.project, "workspaceFolders": []any{map[string]any{"uri": fileURI(m.project), "name": filepath.Base(m.project)}}, "capabilities": map[string]any{"general": map[string]any{"positionEncodings": []string{"utf-16"}}, "textDocument": map[string]any{"synchronization": map[string]any{"didSave": false}, "diagnostic": map[string]any{"dynamicRegistration": false}, "publishDiagnostics": map[string]any{"versionSupport": true}}, "workspace": map[string]any{"configuration": true}}, "initializationOptions": cfg.Initialization, "clientInfo": map[string]string{"name": "independent-harness", "version": "0.1.0"}}
	result, err := p.rpc.call(ctx, "initialize", params)
	if err != nil {
		s.broken = true
		status, failure := m.settleFailedStart(name, p, fmt.Errorf("LSP initialize: %w", err))
		return nil, status, failure
	}
	var initialized struct {
		Capabilities map[string]json.RawMessage `json:"capabilities"`
	}
	if err := json.Unmarshal(result, &initialized); err != nil || initialized.Capabilities == nil {
		s.broken = true
		status, failure := m.settleFailedStart(name, p, errors.New("LSP initialize missing capabilities"))
		return nil, status, failure
	}
	if encoding := initialized.Capabilities["positionEncoding"]; len(encoding) > 0 && string(encoding) != `"utf-16"` {
		s.broken = true
		status, failure := m.settleFailedStart(name, p, fmt.Errorf("LSP server requires unsupported position encoding %s", encoding))
		return nil, status, failure
	}
	s.capabilities = initialized.Capabilities
	if err := p.rpc.notifyContext(ctx, "initialized", map[string]any{}); err != nil {
		s.broken = true
		status, failure := m.settleFailedStart(name, p, err)
		return nil, status, failure
	}
	if cfg.Initialization != nil {
		if err := p.rpc.notifyContext(ctx, "workspace/didChangeConfiguration", map[string]any{"settings": cfg.Initialization}); err != nil {
			s.broken = true
			status, failure := m.settleFailedStart(name, p, err)
			return nil, status, failure
		}
	}
	if m.OnStarted != nil {
		if err := m.OnStarted(name); err != nil {
			s.broken = true
			return nil, "interrupted", err
		}
	}
	return s, "", nil
}

// settleFailedStart stops a broken server and reports the start as settled
// ("failed") only when no process can remain. If cleanup cannot be confirmed,
// the effect stays uncertain ("interrupted") for explicit acknowledgement.
func (m *Manager) settleFailedStart(name string, p *process, err error) (string, error) {
	if p != nil {
		if stopErr := p.stop(); stopErr != nil {
			return "interrupted", err
		}
	}
	if m.OnStartFailed != nil {
		if e := m.OnStartFailed(name, err); e != nil {
			return "interrupted", err
		}
	}
	return "failed", err
}
func resultError(status string, err error) tools.Result {
	message := err.Error()
	if len(message) > 4096 {
		message = strings.ToValidUTF8(message[:4096], "�") + " [truncated]"
	}
	data, _ := json.Marshal(map[string]any{"status": status, "error": message})
	return tools.Result{Content: string(data), Error: true}
}
func (m *Manager) Execute(ctx context.Context, name string, input json.RawMessage) tools.Result {
	m.operationMu.Lock()
	defer m.operationMu.Unlock()
	if err := m.Validate(name, input); err != nil {
		return resultError("failed", err)
	}
	var args arguments
	json.Unmarshal(input, &args)
	path, text, err := m.readFile(args.FilePath)
	if err != nil {
		return resultError("failed", err)
	}
	position, err := utf16Position(text, args.Line, args.Character)
	if err != nil {
		return resultError("failed", err)
	}
	serverName, cfg, _ := m.selectServer(path)
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	s, status, err := m.start(ctx, serverName, cfg)
	if err != nil {
		return resultError(status, err)
	}
	uri := fileURI(path)
	if args.Operation != "diagnostics" {
		capability := s.capabilities[capabilities[args.Operation]]
		if len(capability) == 0 || string(capability) == "false" || string(capability) == "null" {
			return resultError("failed", fmt.Errorf("LSP server does not support %s", args.Operation))
		}
	}
	operation := func() (json.RawMessage, error) {
		old, opened := s.documents[uri]
		if !opened {
			if len(s.documents) >= 64 {
				return nil, errors.New("LSP opened-document limit of 64 reached; start a new run")
			}
			version := 1
			s.process.rpc.forgetDiagnostics(uri)
			if err := s.process.rpc.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": languageID(path, cfg.LanguageID), "version": version, "text": text}}); err != nil {
				return nil, err
			}
			s.documents[uri] = document{version, text}
		} else if old.text != text {
			version := old.version + 1
			s.process.rpc.forgetDiagnostics(uri)
			if err := s.process.rpc.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": version}, "contentChanges": []any{map[string]string{"text": text}}}); err != nil {
				return nil, err
			}
			s.documents[uri] = document{version, text}
		}
		params := map[string]any{"textDocument": map[string]string{"uri": uri}}
		switch args.Operation {
		case "workspaceSymbol":
			params = map[string]any{"query": args.Query}
		case "documentSymbol":
		case "diagnostics":
			capability := s.capabilities["diagnosticProvider"]
			if len(capability) == 0 || string(capability) == "false" || string(capability) == "null" {
				timer := time.NewTimer(diagnosticsWait)
				defer timer.Stop()
				for {
					var settle <-chan time.Time
					data, ok, unversioned, quiet := s.process.rpc.cachedDiagnostics(uri, s.documents[uri].version)
					switch {
					case ok && (!unversioned || quiet >= diagnosticsSettle):
						return data, nil
					case ok:
						// Unversioned diagnostics can describe the text before
						// the change: take the latest once publishing settles.
						settle = time.After(diagnosticsSettle - quiet)
					}
					select {
					case <-settle:
					case <-s.process.rpc.changed:
					case <-s.process.rpc.done:
						return nil, s.process.rpc.failure()
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-timer.C:
						return nil, errDiagnosticsPending
					}
				}
			}
		default:
			params["position"] = position
			if args.Operation == "findReferences" {
				params["context"] = map[string]bool{"includeDeclaration": true}
			}
		}
		if args.Operation == "incomingCalls" || args.Operation == "outgoingCalls" {
			// Call hierarchy is two requests: prepare at the position, then ask
			// about the first item returned.
			prepared, err := s.process.rpc.call(ctx, "textDocument/prepareCallHierarchy", params)
			if err != nil {
				return nil, err
			}
			var items []json.RawMessage
			if json.Unmarshal(prepared, &items) != nil || len(items) == 0 {
				return json.RawMessage(`[]`), nil
			}
			return s.process.rpc.call(ctx, operations[args.Operation], map[string]any{"item": items[0]})
		}
		return s.process.rpc.call(ctx, operations[args.Operation], params)
	}
	// Bound notifications as well as requests. Closing the RPC pipes on timeout
	// releases a server that stopped reading without blocking the runtime forever.
	type answer struct {
		data json.RawMessage
		err  error
	}
	done := make(chan answer, 1)
	go func() { data, err := operation(); done <- answer{data, err} }()
	var answerResult answer
	select {
	case answerResult = <-done:
	case <-ctx.Done():
		answerResult.err = ctx.Err()
		if args.Operation == "diagnostics" {
			// The diagnostics wait also watches ctx and returns at once. Let it
			// finish so the server can be reused without a concurrent writer;
			// if it does not, fall through and stop the server as before.
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				args.Operation = "diagnostics-stuck"
			}
		}
	}
	if answerResult.err != nil {
		var protocolError *rpcError
		alive := true
		select {
		case <-s.process.rpc.done:
			alive = false
		default:
		}
		// Waiting for diagnostics only reads published state, so a timeout
		// there says nothing about the server's health.
		if errors.As(answerResult.err, &protocolError) || errors.Is(answerResult.err, errDiagnosticsPending) ||
			(alive && args.Operation == "diagnostics") {
			// The server is healthy: it answered with an error, or has simply
			// not published diagnostics yet (slow cold start). Keep it.
			return resultError("failed", answerResult.err)
		}
		s.broken = true
		status := "interrupted"
		cleanupErr := s.process.stop()
		if cleanupErr != nil {
			status = "cleanup_incomplete"
			answerResult.err = errors.Join(answerResult.err, cleanupErr)
		}
		return resultError(status, fmt.Errorf("LSP operation did not complete; no automatic replay: %w", answerResult.err))
	}
	if len(answerResult.data) > maxResult {
		return resultError("failed", errors.New("LSP result exceeds 64 KiB; output omitted"))
	}
	data, _ := json.Marshal(map[string]any{"server": serverName, "operation": args.Operation, "result": answerResult.data})
	if len(data) > maxResult {
		return resultError("failed", errors.New("LSP result envelope exceeds 64KiB; output omitted"))
	}
	return tools.Result{Content: string(data)}
}
func (m *Manager) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	servers := make([]*server, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}
	m.mu.Unlock()
	var wg sync.WaitGroup
	errs := make(chan error, len(servers))
	for _, server := range servers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := server.process.stop(); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	var err error
	for e := range errs {
		err = errors.Join(err, e)
	}
	return err
}

func languageID(path, override string) string {
	if override != "" {
		return override
	}
	extension := strings.ToLower(filepath.Ext(path))
	ids := map[string]string{".go": "go", ".js": "javascript", ".jsx": "javascriptreact", ".ts": "typescript", ".tsx": "typescriptreact", ".py": "python", ".rs": "rust", ".c": "c", ".h": "c", ".cpp": "cpp", ".cc": "cpp", ".cxx": "cpp", ".hpp": "cpp", ".java": "java", ".json": "json", ".md": "markdown", ".yaml": "yaml", ".yml": "yaml", ".sh": "shellscript", ".html": "html", ".css": "css"}
	if id := ids[extension]; id != "" {
		return id
	}
	return strings.TrimPrefix(extension, ".")
}
