package mcpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"example.com/independent-harness/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type echoIn struct {
	Text string `json:"text"`
}

func remoteServer(t *testing.T, sawAuth *atomic.Value) *httptest.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "fixture", Version: "1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "echo", Description: "Echo text"}, func(_ context.Context, _ *mcp.CallToolRequest, in echoIn) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "echo: " + in.Text}}}, nil, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth.Store(r.Header.Get("Authorization"))
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func TestRemoteMCPServer(t *testing.T) {
	var sawAuth atomic.Value
	ts := remoteServer(t, &sawAuth)
	t.Setenv("FIXTURE_MCP_TOKEN", "Bearer secret-token")
	var preview string
	approve := func(_ context.Context, p core.Permission) (core.Decision, error) {
		preview = p.Preview
		return core.Decision{Allow: true}, nil
	}
	m := New(map[string]core.MCPConfig{"remote": {URL: ts.URL, HeadersEnv: map[string]string{"Authorization": "FIXTURE_MCP_TOKEN"}}}, t.TempDir(), approve)
	defer m.Close()
	if err := m.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(preview, "secret-token") || !strings.Contains(preview, ts.URL) || !strings.Contains(preview, "Authorization") {
		t.Fatal("approval preview must show URL and header names, never values:", preview)
	}
	if !m.Has("mcp__remote__echo") {
		t.Fatal("remote tool not discovered", m.Definitions())
	}
	args, _ := json.Marshal(map[string]string{"text": "hi"})
	r := m.Execute(context.Background(), "mcp__remote__echo", args)
	if r.Error || !strings.Contains(r.Content, "echo: hi") {
		t.Fatal(r.Content)
	}
	if sawAuth.Load() != "Bearer secret-token" {
		t.Fatal("header from environment not sent", sawAuth.Load())
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteMCPRefusals(t *testing.T) {
	for _, bad := range []string{"http://example.com/mcp", "ftp://x/mcp", "https://user:pw@example.com/mcp", "https://example.com/mcp#f"} {
		if checkRemoteURL(bad) == nil {
			t.Fatal("accepted", bad)
		}
	}
	for _, ok := range []string{"https://example.com/mcp", "http://127.0.0.1:9/mcp", "http://localhost:9/mcp"} {
		if checkRemoteURL(ok) != nil {
			t.Fatal("refused", ok)
		}
	}
	if _, err := remoteClient(core.MCPConfig{HeadersEnv: map[string]string{"Authorization": "UNSET_FIXTURE_VAR"}}); err == nil {
		t.Fatal("missing header variable accepted")
	}
	allow := func(context.Context, core.Permission) (core.Decision, error) { return core.Decision{Allow: true}, nil }
	both := New(map[string]core.MCPConfig{"x": {Command: "x", URL: "https://example.com"}}, t.TempDir(), allow)
	if err := both.Connect(context.Background()); err == nil && len(both.Failures()) == 0 {
		t.Fatal("command and url together accepted")
	}
	redirect := httptest.NewServer(http.RedirectHandler("https://elsewhere.example/mcp", http.StatusTemporaryRedirect))
	defer redirect.Close()
	r := New(map[string]core.MCPConfig{"r": {URL: redirect.URL}}, t.TempDir(), allow)
	defer r.Close()
	if err := r.Connect(context.Background()); err == nil && len(r.Failures()) == 0 {
		t.Fatal("redirecting server connected")
	}
}
