package tui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

type scriptedCreate struct{ done bool }

func (s *scriptedCreate) Complete(_ context.Context, _ []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
	if s.done {
		return core.Reply{Message: core.Message{Role: "assistant", Content: "done"}, Finish: "stop"}, nil
	}
	s.done = true
	c := core.Call{ID: "c1", Type: "function"}
	c.Function.Name, c.Function.Arguments = "create_file", `{"path":"made.txt","content":"hello changes\n"}`
	return core.Reply{Message: core.Message{Role: "assistant", ToolCalls: []core.Call{c}}, Finish: "tool_calls"}, nil
}

func TestChangesPanelShowsCheckpointDiff(t *testing.T) {
	cfg := core.Defaults()
	cfg.SessionDir, cfg.Model, cfg.ContextWindow = t.TempDir(), "fixture", 1<<20
	store, err := session.New(cfg.SessionDir, testutil.TempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	a, err := agent.New(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	m := &model{a: a, width: 100, height: 30, status: "ready"}
	m.openChanges()
	if m.changesPicker != nil || !strings.Contains(m.status, "snapshots") {
		t.Fatal("without checkpoints /changes should explain how to enable them", m.status)
	}
	a.Config.Snapshots = true
	a.Grants["create_file"] = true
	a.Provider = &scriptedCreate{}
	if status, err := a.Run(context.Background(), "make a file"); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	m.openChanges()
	if m.changesPicker == nil || len(m.changesPicker.options) != 1 || !strings.Contains(m.changesPicker.options[0].Label, "made.txt") {
		t.Fatal("checkpoint not listed", m.status)
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.infoPanel == nil || m.infoPanel.title != "Changes" {
		t.Fatal("diff not shown", m.status)
	}
	all := strings.Join(m.infoPanel.lines, "\n")
	if !strings.Contains(all, "made.txt · added") || !strings.Contains(all, "hello changes") {
		t.Fatal(all)
	}
	m.infoPanel = nil
	m.active = true
	m.openChanges()
	if m.changesPicker != nil {
		t.Fatal("opened during a run")
	}
}
