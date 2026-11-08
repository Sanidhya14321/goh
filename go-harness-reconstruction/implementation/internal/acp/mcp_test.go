package acp

import (
	"encoding/json"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestMergeMCPHTTPAndStdio(t *testing.T) {
	raw := func(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
	merged, err := mergeMCP(map[string]core.MCPConfig{"own": {Command: "/bin/own"}}, []json.RawMessage{
		raw(map[string]any{"type": "http", "name": "docs", "url": "https://mcp.example.com/mcp", "headers": []map[string]string{{"name": "Authorization", "value": "Bearer t"}}}),
		raw(map[string]any{"name": "local", "command": "/usr/bin/tool", "args": []string{"--x"}}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if d := merged["docs"]; d.URL != "https://mcp.example.com/mcp" || d.Headers["Authorization"] != "Bearer t" || d.Command != "" {
		t.Fatalf("%+v", d)
	}
	if merged["local"].Command != "/usr/bin/tool" || merged["own"].Command != "/bin/own" {
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
