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
