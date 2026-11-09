package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestConfigShowRemovesSecrets(t *testing.T) {
	c := core.Defaults()
	c.Model = "m"
	c.Endpoint = "https://user:pw-secret@gw.example/v1/chat/completions?key=q-secret"
	c.MCP = map[string]core.MCPConfig{"db": {Command: "db-mcp", Env: map[string]string{"DB_PASSWORD": "env-secret"}}}
	c.Formatters = map[string]core.FormatterConfig{"x": {Command: "fmt", Args: []string{"--token=sk-proj-abcdefghijklmnopqrstuvwxyz123456"}}}
	var b bytes.Buffer
	if err := writeConfig(&b, c); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, secret := range []string{"pw-secret", "q-secret", "env-secret", "abcdefghijklmnopqrstuvwxyz123456"} {
		if strings.Contains(out, secret) {
			t.Fatalf("leaked %s:\n%s", secret, out)
		}
	}
	for _, kept := range []string{"gw.example/v1/chat/completions", "DB_PASSWORD", `"model": "m"`} {
		if !strings.Contains(out, kept) {
			t.Fatalf("missing %s:\n%s", kept, out)
		}
	}
	var v map[string]any
	if json.Unmarshal(b.Bytes(), &v) != nil {
		t.Fatal("not JSON")
	}
}
