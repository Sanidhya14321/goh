package tui

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/sandbox"
)

// Shift+Tab cycles the session's access level, says what it means, and the
// footer shows it; the level is remembered for the next chat screen.
func TestShiftTabCyclesAccess(t *testing.T) {
	t.Cleanup(func() { SetSessionAccessLevel("", "") })
	m := fuzzModel(t)
	if !strings.Contains(m.View().Content, "Access: Ask") {
		t.Fatal("footer does not show the access level")
	}
	for _, want := range []core.AccessLevel{core.AccessEdits, core.AccessAuto, core.AccessFull, core.AccessAsk} {
		m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
		if m.a.ProjectAccess() != want || SessionAccessFor(m.a.Store.ID) != want {
			t.Fatalf("got %s / %s, want %s", m.a.ProjectAccess(), SessionAccessFor(m.a.Store.ID), want)
		}
		// Review P1: the session-only level belongs to this session only.
		if SessionAccessFor("another-session") != "" || SessionAccessFor("") != "" {
			t.Fatal("session access leaked to another session")
		}
		if !strings.Contains(m.status, want.Label()) || !strings.Contains(m.View().Content, "Access: "+want.Label()) {
			t.Fatalf("level %s not shown: %q", want, m.status)
		}
	}
}

// "a" in an edit approval allows it and switches the session to Auto-edit;
// it is not offered for requests no ordinary level covers.
func TestApprovalAllowAllRaisesAccess(t *testing.T) {
	t.Cleanup(func() { SetSessionAccessLevel("", "") })
	m := fuzzModel(t)
	response := make(chan core.Decision, 1)
	m.pending = &permissionMsg{request: core.Permission{Tool: "create_file", Preview: "Create a.go"}, response: response}
	if !strings.Contains(ansi.Strip(m.View().Content), "a all edits") {
		t.Fatal("shortcut not offered")
	}
	m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if d := <-response; !d.Allow || m.a.ProjectAccess() != core.AccessEdits {
		t.Fatalf("decision %+v access %s", d, m.a.ProjectAccess())
	}
	m.pending = &permissionMsg{request: core.Permission{Tool: "webfetch", Preview: "GET https://example.com"}, response: response}
	if strings.Contains(ansi.Strip(m.View().Content), "a all") {
		t.Fatal("shortcut offered for network access")
	}
	m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	select {
	case d := <-response:
		t.Fatalf("a answered a request it does not cover: %+v", d)
	default:
	}
	if sandbox.Supported() {
		m.pending = &permissionMsg{request: core.Permission{Tool: "shell", Preview: "Command: npm test"}, response: response}
		if !strings.Contains(ansi.Strip(m.View().Content), "a all, sandboxed") {
			t.Fatal("command shortcut not offered")
		}
	}
}
