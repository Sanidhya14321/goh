package tui

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

// C10: /snapshots shows each run's whole-project changes and the restore
// commands; with snapshots off it explains how to turn them on.
func TestSnapshotsCommandListsRunChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	cfg := core.Defaults()
	cfg.SessionDir, cfg.Model, cfg.ContextWindow = filepath.Join(t.TempDir(), "sessions"), "fixture", 1<<20
	off := false
	cfg.CompletionCheck = &off
	store, err := session.New(cfg.SessionDir, testutil.TempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	a, err := agent.New(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	m := &model{a: a, width: 100, height: 30}
	m.showSnapshots()
	if !strings.Contains(m.log, "project_snapshots") {
		t.Fatalf("off notice missing: %s", m.log)
	}
	a.Config.ProjectSnapshots = true
	a.Grants["create_file"] = true
	a.Provider = &scriptedCreate{}
	if status, err := a.Run(context.Background(), "make a file"); status != "completed" {
		t.Fatal(status, err)
	}
	m.log = ""
	m.showSnapshots()
	for _, want := range []string{"run 1  1 file(s)", "A made.txt", "goh snapshots restore " + store.ID} {
		if !strings.Contains(m.log, want) {
			t.Fatalf("missing %q in:\n%s", want, m.log)
		}
	}
}
