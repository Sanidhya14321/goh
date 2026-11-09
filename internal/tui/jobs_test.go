package tui

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/tools"
)

func TestJobsPanelShowsOutputAndStops(t *testing.T) {
	// Windows releases a killed process's handle on its working folder a
	// moment after it exits, so remove the project folder with retries.
	project, err := os.MkdirTemp("", "goh-jobs-")
	if err != nil {
		t.Fatal(err)
	}
	if project, err = filepath.EvalSymlinks(project); err != nil {
		t.Fatal(err)
	}
	engine, err := tools.New(project, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		engine.StopJobs()
		for i := 0; i < 50 && os.RemoveAll(project) != nil; i++ {
			time.Sleep(100 * time.Millisecond)
		}
	})
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
	if runningJobs(m.a) != 0 {
		t.Fatal("exited job counted as running")
	}
	longCmd := `{"command":"sleep 30"}`
	if runtime.GOOS == "windows" {
		longCmd = `{"command":"ping -n 30 127.0.0.1"}`
	}
	long := engine.Execute(context.Background(), "job_start", json.RawMessage(longCmd))
	if long.Error {
		t.Fatal(long.Content)
	}
	if runningJobs(m.a) != 1 {
		t.Fatal("running job not counted")
	}
	sidebar := strings.Join(m.sidebar("fixture", "model"), "\n")
	if !strings.Contains(sidebar, "1 background job(s)") {
		t.Fatal("sidebar missing job count", sidebar)
	}
	m.openStatus()
	status := strings.Join(m.infoPanel.lines, "\n")
	if !strings.Contains(status, "BACKGROUND JOBS") || !strings.Contains(status, "job-2 · running · ") {
		t.Fatal("status missing jobs", status)
	}
	m.statusKey("esc")
	m.openJobs()
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	if cmd == nil || m.jobsPicker != nil {
		t.Fatal("ctrl+k should stop the job asynchronously")
	}
	msg := cmd()
	m.Update(msg)
	if !strings.Contains(m.status, "job-2") {
		t.Fatal(m.status)
	}
}
