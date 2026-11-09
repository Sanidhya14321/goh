package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/core"
)

func TestCtrlZSuspends(t *testing.T) {
	m := &model{keys: keymapFor(nil)}
	if m.keys.Direct["ctrl+z"] != "suspend" {
		t.Fatalf("ctrl+z bound to %q", m.keys.Direct["ctrl+z"])
	}
	_, cmd := m.runKeyAction("suspend")
	if cmd == nil {
		t.Fatal("no command")
	}
	if _, ok := cmd().(tea.SuspendMsg); !ok {
		t.Fatal("suspend action must send tea.SuspendMsg")
	}
	c := core.Config{Keybinds: map[string]string{"suspend": "none"}}
	if (&model{keys: keymapFor(&c)}).keys.Direct["ctrl+z"] != "" {
		t.Fatal("suspend could not be disabled")
	}
}
