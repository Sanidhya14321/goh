// Package acp adapts the durable runtime to ACP v1 over JSON-RPC stdio.
package acp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/mcpclient"
	agent "example.com/independent-harness/internal/runtime"
	"example.com/independent-harness/internal/session"
)

const maxFrame = 512 << 10

type message struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

type server struct {
	ctx     context.Context
	stop    context.CancelFunc
	config  core.Config
	project string
	version string

	output   io.Writer
	writeMu  sync.Mutex
	writeErr error

	initialized bool
	sessions    map[string]map[string]core.MCPConfig
	mu          sync.Mutex
	active      *turn
	workers     sync.WaitGroup
}

func Serve(ctx context.Context, input io.ReadCloser, output io.Writer, config core.Config, project, version string) error {
	if err := config.Validate(); err != nil {
		return err
	}
	root, err := filepath.Abs(project)
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return errors.New("ACP project must be an existing directory")
	}
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	s := &server{
		ctx: ctx, stop: stop, config: config, project: root, version: version,
		output: output, sessions: make(map[string]map[string]core.MCPConfig),
	}
	closeInput := context.AfterFunc(ctx, func() { _ = input.Close() })
	defer closeInput()
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), maxFrame)
	for ctx.Err() == nil && scanner.Scan() {
		if len(bytes.TrimSpace(scanner.Bytes())) != 0 {
			s.receive(scanner.Bytes())
		}
	}
	readErr := scanner.Err()
	if ctx.Err() != nil {
		readErr = nil
	}
	stop()
	s.workers.Wait()
	return errors.Join(readErr, s.writeErr)
}

func (s *server) send(frame map[string]any) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.ctx.Err() != nil || s.writeErr != nil {
		return
	}
	frame["jsonrpc"] = "2.0"
	if writer, ok := s.output.(interface{ SetWriteDeadline(time.Time) error }); ok {
		if writer.SetWriteDeadline(time.Now().Add(5*time.Second)) == nil {
			defer writer.SetWriteDeadline(time.Time{})
		}
	}
	if err := json.NewEncoder(s.output).Encode(frame); err != nil {
		s.writeErr = err
		s.stop()
	}
}

func (s *server) result(id json.RawMessage, result any) {
	s.send(map[string]any{"id": id, "result": result})
}

func (s *server) fail(id json.RawMessage, code int, text string) {
	s.send(map[string]any{"id": id, "error": map[string]any{"code": code, "message": text}})
}

func validID(id json.RawMessage) bool {
	return len(id) > 0 && (id[0] == '"' || id[0] == '-' || (id[0] >= '0' && id[0] <= '9'))
}

func (s *server) receive(line []byte) {
	var m message
	if !utf8.Valid(line) || !json.Valid(line) {
		s.fail(nil, -32700, "Invalid JSON")
		return
	}
	if json.Unmarshal(line, &m) != nil || m.Version != "2.0" || (len(m.ID) > 0 && !validID(m.ID)) {
		s.fail(nil, -32600, "Invalid request")
		return
	}
	if m.Method == "" {
		if len(m.Result) == 0 && len(m.Error) == 0 {
			s.fail(nil, -32600, "Invalid request")
			return
		}
		s.receivePermission(m)
		return
	}
	if len(m.ID) == 0 {
		if m.Method == "session/cancel" {
			var params struct {
				SessionID string `json:"sessionId"`
			}
			if json.Unmarshal(m.Params, &params) == nil {
				s.mu.Lock()
				if s.active != nil && s.active.sessionID == params.SessionID {
					s.active.cancel()
				}
				s.mu.Unlock()
			}
		}
		return
	}
	if m.Method == "initialize" {
		var params struct {
			Version int `json:"protocolVersion"`
		}
		if s.initialized || json.Unmarshal(m.Params, &params) != nil || params.Version < 1 {
			s.fail(m.ID, -32602, "Invalid initialization")
			return
		}
		s.initialized = true
		s.result(m.ID, map[string]any{
			"protocolVersion": 1,
			"agentInfo":       map[string]string{"name": "independent-harness", "version": s.version},
			"agentCapabilities": map[string]any{
				"sessionCapabilities": map[string]any{"resume": struct{}{}, "close": struct{}{}},
				"mcpCapabilities":     map[string]any{"http": true},
			},
		})
		return
	}
	if !s.initialized {
		s.fail(m.ID, -32000, "Initialize the connection first")
		return
	}
	switch m.Method {
	case "session/new", "session/resume":
		s.openSession(m)
	case "session/close":
		var params struct {
			SessionID string `json:"sessionId"`
		}
		if json.Unmarshal(m.Params, &params) != nil || params.SessionID == "" {
			s.fail(m.ID, -32602, "Invalid session ID")
			return
		}
		s.mu.Lock()
		busy := s.active != nil && s.active.sessionID == params.SessionID
		s.mu.Unlock()
		if busy {
			s.fail(m.ID, -32000, "Cancel the active prompt before closing the session")
			return
		}
		delete(s.sessions, params.SessionID)
		s.result(m.ID, struct{}{})
	case "session/prompt":
		s.startPrompt(m)
	default:
		s.fail(m.ID, -32601, "Method not supported")
	}
}

func (s *server) openSession(m message) {
	var params struct {
		CWD                   string            `json:"cwd"`
		SessionID             string            `json:"sessionId"`
		MCPServers            []json.RawMessage `json:"mcpServers"`
		AdditionalDirectories []string          `json:"additionalDirectories"`
	}
	if json.Unmarshal(m.Params, &params) != nil || !filepath.IsAbs(params.CWD) ||
		len(params.AdditionalDirectories) != 0 {
		s.fail(m.ID, -32602, "An absolute cwd and a single workspace root are required.")
		return
	}
	cwd, err := filepath.EvalSymlinks(params.CWD)
	if err != nil || cwd != s.project {
		s.fail(m.ID, -32602, "cwd must match the configured project")
		return
	}
	_, known := s.sessions[params.SessionID]
	if len(s.sessions) >= 128 && !known {
		s.fail(m.ID, -32000, "Close an inactive session before opening another")
		return
	}
	mcp, err := mergeMCP(s.config.MCP, params.MCPServers)
	if err != nil {
		s.fail(m.ID, -32602, err.Error())
		return
	}
	var store *session.Store
	if m.Method == "session/new" {
		if params.SessionID != "" {
			s.fail(m.ID, -32602, "A new session cannot specify an ID")
			return
		}
		store, err = session.New(s.config.SessionDir, s.project)
	} else {
		store, err = session.OpenProject(s.config.SessionDir, params.SessionID, s.project)
	}
	if err != nil {
		s.sessionError(m.ID, err)
		return
	}
	if err := store.Close(); err != nil {
		s.fail(m.ID, -32603, "Cannot close the session journal")
		return
	}
	s.sessions[store.ID] = mcp
	if m.Method == "session/new" {
		s.result(m.ID, map[string]string{"sessionId": store.ID})
	} else {
		s.result(m.ID, struct{}{})
	}
}

func (s *server) sessionError(id json.RawMessage, err error) {
	switch {
	case errors.Is(err, session.ErrBusy):
		s.fail(id, -32000, "Session already has an active writer")
	case errors.Is(err, session.ErrProjectMismatch), errors.Is(err, os.ErrNotExist):
		s.fail(id, -32002, "Unknown session")
	default:
		s.fail(id, -32603, "Cannot open the session journal")
	}
}

func (s *server) startPrompt(m message) {
	var params struct {
		SessionID string `json:"sessionId"`
		Prompt    []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"prompt"`
	}
	if json.Unmarshal(m.Params, &params) != nil || len(params.Prompt) == 0 || len(params.Prompt) > 64 {
		s.fail(m.ID, -32602, "A prompt requires 1 to 64 text blocks")
		return
	}
	var text strings.Builder
	for i, block := range params.Prompt {
		if block.Type != "text" || text.Len()+len(block.Text)+1 > 64<<10 {
			s.fail(m.ID, -32602, "Only text prompts up to 64 KiB are supported")
			return
		}
		if i > 0 {
			text.WriteByte('\n')
		}
		text.WriteString(block.Text)
	}
	if strings.TrimSpace(text.String()) == "" {
		s.fail(m.ID, -32602, "Prompt cannot be blank")
		return
	}
	mcp, known := s.sessions[params.SessionID]
	if !known {
		s.fail(m.ID, -32002, "Unknown session")
		return
	}
	s.mu.Lock()
	if s.active != nil {
		s.mu.Unlock()
		s.fail(m.ID, -32000, "Agent already has an active prompt")
		return
	}
	ctx, cancel := context.WithTimeout(s.ctx, 15*time.Minute)
	t := &turn{server: s, sessionID: params.SessionID, cancel: cancel, calls: make(map[string]*toolCall)}
	s.active = t
	s.mu.Unlock()
	s.workers.Add(1)
	go func() {
		defer s.workers.Done()
		defer cancel()
		code, result := t.run(ctx, text.String(), mcp)
		s.mu.Lock()
		s.active = nil
		if code == 0 {
			s.result(m.ID, map[string]string{"stopReason": result})
		} else {
			s.fail(m.ID, code, result)
		}
		s.mu.Unlock()
	}()
}

func (t *turn) run(ctx context.Context, text string, mcp map[string]core.MCPConfig) (int, string) {
	store, err := session.OpenProject(t.server.config.SessionDir, t.sessionID, t.server.project)
	if err != nil {
		return -32000, "Session unavailable"
	}
	defer store.Close()
	if len(store.Pending()) > 0 {
		return -32000, "Session has uncertain tool effects. Inspect and acknowledge them using the CLI."
	}
	a, err := agent.New(t.server.config, store)
	if err != nil {
		return -32603, "Cannot initialize the runtime"
	}
	defer a.Tools.StopJobs() // background jobs end with the prompt
	a.ExternalFactory = func() agent.External {
		return mcpclient.New(mcp, store.Project, a.Permission)
	}
	a.Emit = t.event
	a.Approve = t.approve
	status, err := a.Run(ctx, text)
	t.finishOpenCalls()
	if status == "needs_input" && err != nil {
		t.update(map[string]any{"sessionUpdate": "agent_message_chunk", "content": textBlock(err.Error())})
	}
	return stopReason(status, err)
}
