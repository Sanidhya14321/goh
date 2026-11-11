package mcpclient

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// Review R9: arguments with duplicate keys are refused before reaching the
// server, and what is sent is the validated object.
func TestDuplicateKeysRefusedAndValidatedObjectSent(t *testing.T) {
	m := manager(t, nil, true)
	if err := m.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	r := m.Execute(context.Background(), "mcp__fixture__echo", json.RawMessage(`{"text":"safe","text":"rm -rf /"}`))
	if !r.Error || !strings.Contains(r.Content, "duplicate key") {
		t.Fatalf("duplicate keys accepted: %+v", r)
	}
	r = m.Execute(context.Background(), "mcp__fixture__echo", json.RawMessage(`{"text":"ok"}`))
	if r.Error || !strings.Contains(r.Content, "ok") {
		t.Fatalf("normal call failed: %+v", r)
	}
	for _, bad := range []string{`{"a":{"b":1,"b":2}}`, `{"a":[{"x":1,"x":1}]}`} {
		if rejectDuplicateKeys([]byte(bad)) == nil {
			t.Fatal("nested duplicate accepted:", bad)
		}
	}
}

func TestLargeIntegersReachTheServerExactly(t *testing.T) {
	m := manager(t, map[string]string{"HARNESS_MCP_RAW": "1"}, true)
	if err := m.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	r := m.Execute(context.Background(), "mcp__fixture__raw", json.RawMessage(`{"id":9007199254740993}`))
	if r.Error || !strings.Contains(r.Content, "9007199254740993") {
		t.Fatalf("server received %s", r.Content)
	}
	if r := m.Execute(context.Background(), "mcp__fixture__raw", json.RawMessage(`{"id":1.5}`)); !r.Error {
		t.Fatal("non-integer accepted for an integer field")
	}
}
