package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"example.com/independent-harness/internal/core"
)

// topRow renders and returns the first non-blank visible transcript row.
func topRow(m *model) string {
	m.View()
	for _, r := range m.visibleRows(m.wrappedText, m.wrappedWidth, m.transcriptHeight(), false) {
		if text := strings.TrimSpace(r.text); text != "" {
			return text
		}
	}
	return ""
}

func leader(m *model, key rune) {
	m.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	m.Update(tea.KeyPressMsg{Code: key, Text: string(key)})
}

func TestTranscriptJumpsBetweenUserTurns(t *testing.T) {
	m := &model{width: 80, height: 20, status: "ready"}
	for i := 1; i <= 4; i++ {
		m.log += fmt.Sprintf("\nYou: prompt %d\n", i) + strings.Repeat(fmt.Sprintf("answer %d line\n", i), 12)
	}
	m.View()
	leader(m, 'k') // previous turn from the bottom
	if got := topRow(m); got != "You: prompt 4" {
		t.Fatalf("previous: top row %q", got)
	}
	leader(m, 'k')
	if got := topRow(m); got != "You: prompt 3" {
		t.Fatalf("previous again: top row %q", got)
	}
	leader(m, 'j')
	if got := topRow(m); got != "You: prompt 4" {
		t.Fatalf("next: top row %q", got)
	}
	leader(m, 'g') // first message
	if got := topRow(m); got != "You: prompt 1" {
		t.Fatalf("first: top row %q", got)
	}
	m.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl | tea.ModAlt}) // last
	if m.scrollOffset != 0 {
		t.Fatal("last should return to the bottom", m.scrollOffset)
	}
}

func TestCopyWithoutReplyExplains(t *testing.T) {
	m := &model{width: 40, height: 10}
	if cmd := m.copyLastReply(); cmd != nil {
		t.Fatal("nothing to copy without an agent")
	}
}

func TestShortcutHintsFollowKeymap(t *testing.T) {
	m := &model{}
	if m.keyLabel("command_palette") != "Ctrl+P" || m.keyLabel("sidebar_toggle") != "Ctrl+B" || m.keyLabel("editor_open") != "Ctrl+X E" {
		t.Fatal(m.keyLabel("command_palette"), m.keyLabel("sidebar_toggle"), m.keyLabel("editor_open"))
	}
	c := core.Defaults()
	c.Keybinds = map[string]string{"sidebar_toggle": "<leader>b", "command_palette": "f1", "leader": "ctrl+o"}
	m = &model{keys: keymapFor(&c)}
	if m.keyLabel("sidebar_toggle") != "Ctrl+O B" || m.keyLabel("command_palette") != "F1" {
		t.Fatal(m.keyLabel("sidebar_toggle"), m.keyLabel("command_palette"))
	}
	c.Keybinds = map[string]string{"sidebar_toggle": "none"}
	m = &model{keys: keymapFor(&c)}
	if m.keyLabel("sidebar_toggle") != "" {
		t.Fatal("unbound action should have no hint")
	}
}
