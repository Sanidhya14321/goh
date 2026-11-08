package main

import (
	"bytes"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestWriteMCPHidesSecrets(t *testing.T) {
	servers := map[string]core.MCPConfig{
		"files": {Command: "mcp-files", Args: []string{"--root", "."}, Env: map[string]string{"API_KEY": "s3cret-env"}},
		"docs":  {URL: "https://user:pw@mcp.example.com/v1?token=s3cret-q#frag", HeadersEnv: map[string]string{"Authorization": "DOCS_TOKEN"}},
	}
	var b bytes.Buffer
	writeMCP(&b, servers, func(string) (string, bool) { return "", false })
	out := b.String()
	for _, want := range []string{"files", "local  mcp-files --root .", "env API_KEY", "remote https://mcp.example.com/v1", "header Authorization from $DOCS_TOKEN (not set)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in\n%s", want, out)
		}
	}
	for _, secret := range []string{"s3cret", "pw@", "frag"} {
		if strings.Contains(out, secret) {
			t.Fatalf("leaked %q:\n%s", secret, out)
		}
	}
	b.Reset()
	writeMCP(&b, nil, nil)
	if !strings.Contains(b.String(), "No MCP servers") {
		t.Fatal(b.String())
	}
}
