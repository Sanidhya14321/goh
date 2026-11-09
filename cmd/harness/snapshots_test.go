package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

type scripted func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error)

func (f scripted) Complete(c context.Context, h []core.Message, d []core.Tool, e func(string)) (core.Reply, error) {
	return f(c, h, d, e)
}

func TestProjectSnapshotsRevertShellChanges(t *testing.T) {
	testutil.RequireUnixShell(t)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	project := testutil.TempDir(t)
	os.WriteFile(filepath.Join(project, "keep.txt"), []byte("original\n"), 0o644)
	c := core.Defaults()
	c.SessionDir = filepath.Join(t.TempDir(), "sessions")
	c.ContextWindow, c.Model, c.ProjectSnapshots = 1<<20, "fixture", true
	s, err := session.New(c.SessionDir, project)
	if err != nil {
		t.Fatal(err)
	}
	a, err := runtime.New(c, s)
	if err != nil {
		t.Fatal(err)
	}
	a.Grants["shell"] = true
	turn := 0
	a.Provider = scripted(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		if turn++; turn == 1 {
			b, _ := json.Marshal(map[string]any{"command": "echo clobbered > keep.txt; mkdir gen; echo x > gen/new.txt"})
			call := core.Call{ID: "sh", Type: "function"}
			call.Function.Name, call.Function.Arguments = "shell", string(b)
			return core.Reply{Message: core.Message{Role: "assistant", ToolCalls: []core.Call{call}}, Finish: "tool_calls"}, nil
		}
		return core.Reply{Message: core.Message{Role: "assistant", Content: "done"}, Finish: "stop"}, nil
	})
	if status, err := a.Run(context.Background(), "change things"); status != "completed" {
		t.Fatal(status, err)
	}
	id := s.ID
	s.Close()
	run := func(args ...string) string {
		var out bytes.Buffer
		if err := runSnapshots(append(args, "--session-dir", c.SessionDir), &out); err != nil {
			t.Fatal(args, err)
		}
		return out.String()
	}
	if got := run("list", id); !strings.Contains(got, "run 1  2 file(s) changed") || !strings.Contains(got, "A gen/new.txt") || !strings.Contains(got, "M keep.txt") {
		t.Fatalf("list: %s", got)
	}
	if got := run("diff", id, "1"); !strings.Contains(got, "+clobbered") {
		t.Fatalf("diff: %s", got)
	}
	if got := run("restore", id, "1"); !strings.Contains(got, "would change") {
		t.Fatalf("restore without --yes: %s", got)
	}
	if b, _ := os.ReadFile(filepath.Join(project, "keep.txt")); string(b) != "clobbered\n" {
		t.Fatal("dry run changed files")
	}
	run("restore", id, "1", "--yes")
	if b, _ := os.ReadFile(filepath.Join(project, "keep.txt")); string(b) != "original\n" {
		t.Fatalf("not restored: %q", b)
	}
	if _, err := os.Stat(filepath.Join(project, "gen")); err == nil {
		t.Fatal("created directory survived the restore")
	}
	run("undo", id, "--yes")
	if b, _ := os.ReadFile(filepath.Join(project, "gen/new.txt")); string(b) != "x\n" {
		t.Fatal("undo did not bring the run's changes back")
	}
	var out bytes.Buffer
	if err := runSnapshots([]string{"restore", id, "9", "--session-dir", c.SessionDir}, &out); err == nil {
		t.Fatal("accepted a run that does not exist")
	}
}
