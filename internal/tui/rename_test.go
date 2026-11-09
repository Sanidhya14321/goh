package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
)

func TestRenameCommandSavesAndResetsTitle(t *testing.T) {
	dir := t.TempDir()
	s, err := session.New(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	cfg := core.Defaults()
	cfg.SessionDir = dir
	m := &model{a: &agent.Agent{Config: cfg, Store: s}, width: 80, height: 24, status: "ready"}
	m.input = "/rename  Fix\tlogin\x07 bug "
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	meta, err := session.ReadMeta(dir, s.ID)
	if err != nil || meta.Title != "Fix login bug" {
		t.Fatalf("title %q, %v", meta.Title, err)
	}
	if m.input != "" || !strings.Contains(m.log, "renamed") {
		t.Fatalf("input %q log %q", m.input, m.log)
	}
	m.input = "/rename"
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if meta, _ := session.ReadMeta(dir, s.ID); meta.Title != "" {
		t.Fatalf("title not reset: %q", meta.Title)
	}
}

func TestExitCommandQuitsWhenIdle(t *testing.T) {
	m := &model{a: &agent.Agent{Config: core.Defaults(), Store: &session.Store{ID: "fixture"}}, width: 80, height: 24, status: "ready"}
	m.input = "/quit"
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("no quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("not a quit")
	}
}

func TestSidebarShowsSessionTitle(t *testing.T) {
	m := &model{a: &agent.Agent{Config: core.Defaults(), Store: &session.Store{ID: "fixture"}}, width: 80, height: 24}
	m.Update(core.Event{Type: "session_titled", Data: map[string]any{"title": "Fix\x1b[2J parser"}})
	side := strings.Join(m.sidebar("fixture", "model"), "\n")
	if !strings.Contains(side, "Fix") || !strings.Contains(side, "parser") || strings.Contains(side, "\x1b") {
		t.Fatalf("%q", side)
	}
}
