package mcpclient

import (
	"context"
	"strings"
	"testing"
)

// C9: prompts from servers that offer them are listed and rendered.
func TestPrompts(t *testing.T) {
	m := manager(t, map[string]string{"HARNESS_MCP_PROMPTS": "1"}, true)
	if err := m.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	list, err := m.Prompts(context.Background())
	if err != nil || len(list) != 1 || list[0].Name != "review" || list[0].Arguments[0] != "file" {
		t.Fatalf("%+v %v", list, err)
	}
	text, err := m.GetPrompt(context.Background(), list[0].Server, "review", map[string]string{"file": "main.go"})
	if err != nil || !strings.Contains(text, "PROMPT-77 review main.go") {
		t.Fatalf("%q %v", text, err)
	}
	if _, err := m.GetPrompt(context.Background(), "nope", "review", nil); err == nil {
		t.Fatal("unknown server accepted")
	}
	plain := manager(t, nil, true)
	plain.Connect(context.Background())
	if l, _ := plain.Prompts(context.Background()); len(l) != 0 {
		t.Fatal("prompts from a server without prompts")
	}
}
