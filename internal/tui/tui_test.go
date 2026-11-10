package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestSafeTerminalContent(t *testing.T) {
	s := Safe("hello\x1b]52;c;secret\a\x1b[2J\r\u202eevil\n✓")
	for _, r := range s {
		if r == '\x1b' || r == '\r' || r == '\a' || r == '\u202e' {
			t.Fatalf("control %U survived", r)
		}
	}
	if !strings.Contains(s, "✓") {
		t.Fatal("unicode lost")
	}
}
func TestResizeAndCancelDoNotExit(t *testing.T) {
	c := core.Defaults()
	s, e := session.New(t.TempDir(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	a, e := agent.New(c, s)
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	m := &model{a: a, active: true, cancel: cancel, width: 80, height: 24}
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	if m.width != 40 || !m.active {
		t.Fatal("resize changed runtime state")
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd != nil || ctx.Err() == nil || !m.active {
		t.Fatal("first cancellation exited app")
	}
	if v := m.View(); !strings.Contains(v.Content, "Ctrl+D") {
		t.Fatal("missing exit control")
	}
}
func TestPermissionDialogResolvesExplicitDecision(t *testing.T) {
	m := &model{}
	ch := make(chan core.Decision, 1)
	m.pending = &permissionMsg{request: core.Permission{Tool: "shell", Preview: "printf hi"}, response: ch}
	m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	d := <-ch
	if !d.Allow || !d.Session || m.pending != nil {
		t.Fatal("permission decision missing")
	}
}

func TestReportedUsageAndTimingRemainVisible(t *testing.T) {
	m := &model{events: make(chan tea.Msg), closed: make(chan struct{})}
	m.Update(core.Event{Type: "assistant_completed", Data: map[string]any{"usage": core.Usage{Prompt: 20, Completion: 5, Total: 25}}})
	m.Update(core.Event{Type: "run_finished", Data: map[string]any{"status": "completed", "duration_ms": 123}})
	m.Update(doneMsg{status: "completed"})
	if m.usage != 25 || m.durationMS != 123 {
		t.Fatalf("usage=%d duration=%d", m.usage, m.durationMS)
	}
}

func TestTaskUpdatesAndLocalTaskCommand(t *testing.T) {
	m := &model{a: &agent.Agent{Config: core.Config{Model: "fixture"}, Store: &session.Store{ID: "fixture"}}, events: make(chan tea.Msg), closed: make(chan struct{}), width: 80, height: 24}
	m.Update(core.Event{Type: "todos_updated", Data: map[string]any{"todos": []core.Todo{{Content: "Investigate parser", Status: "in_progress", Priority: "high"}}}})
	if len(m.todos) != 1 {
		t.Fatal("task update not rendered")
	}
	if !strings.Contains(m.View().Content, "Tasks: 1 remaining / 1 total") {
		t.Fatal("missing task overview")
	}
	m.input = "/todos"
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil || m.active || m.input != "" || !strings.Contains(m.log, "Investigate parser") {
		t.Fatal("task reader started a provider run")
	}
}

func TestNarrowApprovalViewFitsTerminalCells(t *testing.T) {
	m := &model{
		a:     &agent.Agent{Config: core.Config{Model: "fixture"}, Store: &session.Store{ID: "fixture"}},
		width: 24, height: 10, input: strings.Repeat("界", 100),
		pending: &permissionMsg{request: core.Permission{
			Tool: "edit_file", Preview: strings.Repeat("界", 19) + "\n\tfile.go",
		}},
	}
	view := m.View().Content
	if !strings.Contains(view, "Allow edit_file?") {
		t.Fatal("approval title disappeared in a narrow view")
	}
	rows := strings.Split(view, "\n")
	if len(rows) > m.height {
		t.Fatalf("view has %d rows for a %d-row terminal", len(rows), m.height)
	}
	for _, row := range rows {
		if ansi.StringWidth(row) > m.width {
			t.Fatalf("row exceeds terminal width: %q", row)
		}
	}
}
