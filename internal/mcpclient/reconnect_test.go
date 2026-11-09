package mcpclient

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestReconnectAfterServerDropsWithoutReplay(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	approvals := 0
	m := New(map[string]core.MCPConfig{"fixture": {Command: binary, Args: []string{"-test.run=^TestMCPHelper$"}, Env: map[string]string{"HARNESS_MCP_HELPER": "1"}}}, t.TempDir(),
		func(context.Context, core.Permission) (core.Decision, error) {
			approvals++
			return core.Decision{Allow: true}, nil
		})
	t.Cleanup(func() { m.Close() })
	ctx := context.Background()
	if err := m.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	echo := func() (bool, string) {
		r := m.Execute(ctx, "mcp__fixture__echo", json.RawMessage(`{"text":"hello"}`))
		return !r.Error && strings.Contains(r.Content, "hello"), r.Content
	}
	for round := 1; round <= maxReconnects; round++ {
		r := m.Execute(ctx, "mcp__fixture__disconnect", json.RawMessage(`{}`))
		if !r.Error || !strings.Contains(r.Content, "not replayed") {
			t.Fatalf("round %d: dropped call: %+v", round, r)
		}
		if ok, out := echo(); !ok {
			t.Fatalf("round %d: call after drop did not reconnect: %s", round, out)
		}
		if approvals != round+1 {
			t.Fatalf("round %d: %d approvals; each restart must go through approval", round, approvals)
		}
	}
	m.Execute(ctx, "mcp__fixture__disconnect", json.RawMessage(`{}`))
	if ok, out := echo(); ok || !strings.Contains(out, "could not be restarted") {
		t.Fatalf("reconnects were not bounded: %s", out)
	}
}
