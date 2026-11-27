package mcpclient

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestToolListChanges(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	m := New(map[string]core.MCPConfig{"g": {Command: binary, Args: []string{"-test.run=^TestMCPHelper$"},
		Env: map[string]string{"HARNESS_MCP_HELPER": "1", "HARNESS_MCP_GROW": "1"}}}, t.TempDir(), allowAll)
	defer m.Close()
	if err := m.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if m.Has("mcp__g__extra") || !m.Has("mcp__g__unlock") || m.Generation() != 0 {
		t.Fatal("unexpected start", m.Definitions())
	}
	if r := m.Execute(context.Background(), "mcp__g__unlock", json.RawMessage(`{}`)); r.Error {
		t.Fatal(r.Content)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !(m.Has("mcp__g__extra") && !m.Has("mcp__g__unlock")) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if m.Generation() == 0 || !m.Has("mcp__g__extra") || m.Has("mcp__g__unlock") || !m.Has("mcp__g__echo") {
		t.Fatalf("tool list not refreshed: generation %d, defs %v", m.Generation(), m.Definitions())
	}
	if r := m.Execute(context.Background(), "mcp__g__extra", json.RawMessage(`{}`)); r.Error || !strings.Contains(r.Content, "EXTRA-41") {
		t.Fatal(r.Content)
	}
}
