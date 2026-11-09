package acp

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestMergeMCPHTTPAndStdio(t *testing.T) {
	raw := func(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
	tool := filepath.Join(t.TempDir(), "tool") // absolute on every OS
	merged, err := mergeMCP(map[string]core.MCPConfig{"own": {Command: "/bin/own"}}, []json.RawMessage{
		raw(map[string]any{"type": "http", "name": "docs", "url": "https://mcp.example.com/mcp", "headers": []map[string]string{{"name": "Authorization", "value": "Bearer t"}}}),
		raw(map[string]any{"name": "local", "command": tool, "args": []string{"--x"}}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if d := merged["docs"]; d.URL != "https://mcp.example.com/mcp" || d.Headers["Authorization"] != "Bearer t" || d.Command != "" {
		t.Fatalf("%+v", d)
	}
	if merged["local"].Command != tool || merged["own"].Command != "/bin/own" {
		t.Fatal(merged)
	}
	for _, bad := range []any{
		map[string]any{"type": "http", "name": "x"},
		map[string]any{"type": "http", "name": "x", "url": "https://a", "headers": []map[string]string{{"name": "Bad Name", "value": "v"}}},
		map[string]any{"type": "http", "name": "x", "url": "https://a", "headers": []map[string]string{{"name": "X", "value": "a\r\nInjected: 1"}}},
		map[string]any{"type": "sse", "name": "x", "url": "https://a"},
		map[string]any{"name": "own", "command": "/bin/dup"},
		map[string]any{"name": "rel", "command": "relative"},
	} {
		if _, err := mergeMCP(map[string]core.MCPConfig{"own": {Command: "/bin/own"}}, []json.RawMessage{raw(bad)}); err == nil {
			t.Fatal("accepted", bad)
		}
	}
}
