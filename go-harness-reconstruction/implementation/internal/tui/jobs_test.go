package tui

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/session"
	"example.com/independent-harness/internal/testutil"
	"example.com/independent-harness/internal/tools"
	agent "example.com/independent-harness/internal/runtime"
)

func TestJobsPanelShowsOutputAndStops(t *testing.T) {
	engine, err := tools.New(testutil.TempDir(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(engine.StopJobs)
	m := &model{a: &agent.Agent{Tools: engine, Config: core.Defaults(), Store: &session.Store{ID: "fixture", Project: "/tmp/p"}}, width: 100, height: 30, input: "draft", inputCursor: 5}
	m.openJobs()
	if m.jobsPicker != nil || !strings.Contains(m.status, "No background jobs") {
		t.Fatal("empty job list should explain", m.status)
	}
	r := engine.Execute(context.Background(), "job_start", json.RawMessage(`{"command":"echo panel-output"}`))
	if r.Error {
		t.Fatal(r.Content)
	}
	for i := 0; i < 200 && !strings.Contains(func() string { b, _ := json.Marshal(engine.Jobs()); return string(b) }(), "exited"); i++ {
		time.Sleep(20 * time.Millisecond)
	}
	m.openJobs()
	if m.jobsPicker == nil || len(m.jobsPicker.options) != 1 {
		t.Fatal("job not listed")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.infoPanel == nil || m.infoPanel.title != "Job output" || !strings.Contains(strings.Join(m.infoPanel.lines, "\n"), "panel-output") {
		t.Fatal("job output not shown", m.infoPanel)
	}
	if !strings.Contains(m.View().Content, "Job output") {
		t.Fatal("panel title not rendered")
	}
	m.statusKey("r")
	if m.infoPanel.title != "Job output" {
		t.Fatal("refresh replaced job output with status")
	}
	m.statusKey("esc")
	if m.input != "draft" {
		t.Fatal("draft changed")
	}
	m.openJobs()
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	if cmd == nil || m.jobsPicker != nil {
		t.Fatal("ctrl+k should stop the job asynchronously")
	}
	msg := cmd()
	m.Update(msg)
	if !strings.Contains(m.status, "job-1") {
		t.Fatal(m.status)
	}
}
