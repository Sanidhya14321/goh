package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func allowAll(context.Context, core.Permission) (core.Decision, error) {
	return core.Decision{Allow: true}, nil
}

func TestSSEMCPServer(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "legacy", Version: "1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "echo", Description: "Echo text"}, func(_ context.Context, _ *mcp.CallToolRequest, in echoIn) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "sse: " + in.Text}}}, nil, nil
	})
	handler := mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return server }, nil)
	var auth atomic.Value
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth.Store(r.Header.Get("Authorization"))
		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()
	t.Setenv("FIXTURE_SSE_TOKEN", "Bearer s")
	m := New(map[string]core.MCPConfig{"old": {URL: ts.URL, Transport: "sse", HeadersEnv: map[string]string{"Authorization": "FIXTURE_SSE_TOKEN"}}}, t.TempDir(), allowAll)
	defer m.Close()
	if err := m.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]string{"text": "hi"})
	if r := m.Execute(context.Background(), "mcp__old__echo", args); r.Error || !strings.Contains(r.Content, "sse: hi") {
		t.Fatal(r.Content)
	}
	if auth.Load() != "Bearer s" {
		t.Fatal("header not sent", auth.Load())
	}
}

func TestSSEMessageEndpointStaysOnOrigin(t *testing.T) {
	var elsewhere atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { elsewhere.Add(1) }))
	defer other.Close()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: endpoint\ndata: %s/messages\n\n", other.URL)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer ts.Close()
	m := New(map[string]core.MCPConfig{"old": {URL: ts.URL, Transport: "sse"}}, t.TempDir(), allowAll)
	defer m.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5e9)
	defer cancel()
	if err := m.Connect(ctx); err != nil || len(m.Failures()) != 1 || m.Has("mcp__old__echo") {
		t.Fatal("connected through another origin", err, m.Failures())
	}
	if elsewhere.Load() != 0 {
		t.Fatal("a request reached the other origin")
	}
}

func TestMCPTransportSetting(t *testing.T) {
	for _, cfg := range []core.MCPConfig{{Command: "/bin/true", Transport: "sse"}, {URL: "https://example.com/mcp", Transport: "websocket"}} {
		m := New(map[string]core.MCPConfig{"x": cfg}, t.TempDir(), allowAll)
		m.Connect(context.Background())
		f := m.Failures()
		m.Close()
		if len(f) != 1 || !strings.Contains(f[0].Error, "transport") {
			t.Fatalf("%+v: %v", cfg, f)
		}
	}
	if _, err := Login(context.Background(), "x", core.MCPConfig{URL: "https://example.com/sse", Transport: "sse"}, nil); err == nil || !strings.Contains(err.Error(), "SSE") {
		t.Fatal(err)
	}
}
