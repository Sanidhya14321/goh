package mcpclient

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The fixture is an independent Go MCP server running in a real subprocess.
func TestMCPHelper(t *testing.T) {
	if os.Getenv("HARNESS_MCP_HELPER") != "1" {
		return
	}
	if marker := os.Getenv("HARNESS_MCP_MARKER"); marker != "" {
		if err := os.WriteFile(marker, []byte("started"), 0600); err != nil {
			os.Exit(3)
		}
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "fixture", Version: "1"}, nil)
	schema := map[string]any{"type": "object", "properties": map[string]any{"text": map[string]any{"type": "string"}}, "required": []string{"text"}, "additionalProperties": false}
	if os.Getenv("HARNESS_MCP_BAD_SCHEMA") == "1" {
		schema = map[string]any{"type": "object", "properties": map[string]any{"text": map[string]any{"$ref": "https://invalid.example/schema"}}}
	}
	server.AddTool(&mcp.Tool{Name: "echo", Description: "Echo fixture", InputSchema: schema}, func(_ context.Context, r *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input struct {
			Text string `json:"text"`
		}
		json.Unmarshal(r.Params.Arguments, &input)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: input.Text}}}, nil
	})
	empty := map[string]any{"type": "object", "additionalProperties": false}
	if os.Getenv("HARNESS_MCP_RAW") == "1" {
		server.AddTool(&mcp.Tool{Name: "raw", InputSchema: map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "integer"}}}}, func(_ context.Context, r *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(r.Params.Arguments)}}}, nil
		})
	}
	server.AddTool(&mcp.Tool{Name: "wait", InputSchema: empty}, func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	server.AddTool(&mcp.Tool{Name: "disconnect", InputSchema: empty}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) { os.Exit(7); return nil, nil })
	server.AddTool(&mcp.Tool{Name: "large", InputSchema: empty}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: strings.Repeat("x", maxResult+1)}}}, nil
	})
	if os.Getenv("HARNESS_MCP_RESOURCES") == "1" {
		server.AddResource(&mcp.Resource{URI: "fixture://docs/api", Name: "api", MIMEType: "text/plain"}, func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: "fixture://docs/api", MIMEType: "text/plain", Text: "RESOURCE-TEXT-55"}}}, nil
		})
	}
	if os.Getenv("HARNESS_MCP_PROMPTS") == "1" {
		server.AddPrompt(&mcp.Prompt{Name: "review", Description: "Review a file", Arguments: []*mcp.PromptArgument{{Name: "file", Required: true}}}, func(_ context.Context, r *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: "PROMPT-77 review " + r.Params.Arguments["file"]}}}}, nil
		})
	}
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}
func manager(t *testing.T, env map[string]string, allow bool) *Manager {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if env == nil {
		env = map[string]string{}
	}
	env["HARNESS_MCP_HELPER"] = "1"
	m := New(map[string]core.MCPConfig{"fixture": {Command: binary, Args: []string{"-test.run=^TestMCPHelper$"}, Env: env}}, t.TempDir(), func(context.Context, core.Permission) (core.Decision, error) { return core.Decision{Allow: allow}, nil })
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	return m
}
func TestDiscoveryValidationAndInvocation(t *testing.T) {
	m := manager(t, nil, true)
	if err := m.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := m.Connect(context.Background()); err != nil {
		t.Fatalf("idempotent connect: %v", err)
	}
	if len(m.Definitions()) != 4 || !m.Has("mcp__fixture__echo") {
		t.Fatalf("definitions: %+v", m.Definitions())
	}
	for _, input := range []string{`{}`, `{"text":3}`, `{"text":"hi","extra":true}`, `null`, `{`} {
		if err := m.Validate("mcp__fixture__echo", json.RawMessage(input)); err == nil {
			t.Fatalf("admitted %s", input)
		}
	}
	result := m.Execute(context.Background(), "mcp__fixture__echo", json.RawMessage(`{"text":"hello"}`))
	if result.Error || !strings.Contains(result.Content, "hello") {
		t.Fatalf("result %+v", result)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if result := m.Execute(context.Background(), "mcp__fixture__echo", json.RawMessage(`{"text":"hello"}`)); !result.Error {
		t.Fatal("call after close succeeded")
	}
}
func TestPermissionDenialPreventsStart(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	m := manager(t, map[string]string{"HARNESS_MCP_MARKER": marker}, false)
	if err := m.Connect(context.Background()); err == nil {
		t.Fatal("unapproved server started")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("marker exists: %v", err)
	}
}
func TestRejectSchemaReferences(t *testing.T) {
	m := manager(t, map[string]string{"HARNESS_MCP_BAD_SCHEMA": "1"}, true)
	// An invalid server is skipped and stopped; it no longer fails Connect.
	if err := m.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	f := m.Failures()
	if len(f) != 1 || f[0].Name != "fixture" || !strings.Contains(f[0].Error, "inline schema") || len(m.Definitions()) != 0 {
		t.Fatalf("failures %+v definitions %d", f, len(m.Definitions()))
	}
	m.mu.Lock()
	running := len(m.processes)
	m.mu.Unlock()
	if running != 0 {
		t.Fatal("failed server process kept running")
	}
}
func TestCancellationAndDisconnect(t *testing.T) {
	for _, tool := range []string{"wait", "disconnect"} {
		t.Run(tool, func(t *testing.T) {
			m := manager(t, nil, true)
			if err := m.Connect(context.Background()); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			start := time.Now()
			result := m.Execute(ctx, "mcp__fixture__"+tool, json.RawMessage(`{}`))
			if !result.Error || !strings.Contains(result.Content, "not replayed") {
				t.Fatalf("result: %+v", result)
			}
			if time.Since(start) > 5*time.Second {
				t.Fatal("cancellation cleanup exceeded bound")
			}
		})
	}
}
func TestBoundedResults(t *testing.T) {
	m := manager(t, nil, true)
	if err := m.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	result := m.Execute(context.Background(), "mcp__fixture__large", json.RawMessage(`{}`))
	// Review R16: a completed action with oversized output is reported as
	// completed (not an error to retry), bounded, with a do-not-repeat note.
	if result.Error || len(result.Content) > maxResult || !strings.Contains(result.Content, "completed_output_truncated") || !strings.Contains(result.Content, "Do not repeat") {
		t.Fatalf("oversized result: %d %v %.200s", len(result.Content), result.Error, result.Content)
	}
}

func TestFailedServerDoesNotRemoveHealthyServer(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"-test.run=^TestMCPHelper$"}
	m := New(map[string]core.MCPConfig{
		"good":    {Command: binary, Args: args, Env: map[string]string{"HARNESS_MCP_HELPER": "1"}},
		"broken":  {Command: binary, Args: args, Env: map[string]string{"HARNESS_MCP_HELPER": "1", "HARNESS_MCP_BAD_SCHEMA": "1"}},
		"missing": {Command: filepath.Join(t.TempDir(), "no-such-server")},
	}, t.TempDir(), func(context.Context, core.Permission) (core.Decision, error) { return core.Decision{Allow: true}, nil })
	defer m.Close()
	if err := m.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	var failed []string
	for _, f := range m.Failures() {
		failed = append(failed, f.Name)
	}
	if strings.Join(failed, ",") != "broken,missing" || !m.Has("mcp__good__echo") || m.Has("mcp__broken__echo") {
		t.Fatalf("failed=%v definitions=%d", failed, len(m.Definitions()))
	}
	if r := m.Execute(context.Background(), "mcp__good__echo", json.RawMessage(`{"text":"still works"}`)); r.Error || !strings.Contains(r.Content, "still works") {
		t.Fatal(r)
	}
}
