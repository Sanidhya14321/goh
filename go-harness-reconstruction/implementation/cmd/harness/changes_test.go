package main

import (
	"context"
	"example.com/independent-harness/internal/session"
	"example.com/independent-harness/internal/snapshots"
	"os"
	"path/filepath"
	"testing"
)

func TestChangesCLIUndoRedoConflictAndRecovery(t *testing.T) {
	dir, project := t.TempDir(), t.TempDir()
	s, err := session.New(dir, project)
	if err != nil {
		t.Fatal(err)
	}
	subject := s.ID
	s.Close()
	path := filepath.Join(project, "note")
	os.WriteFile(path, []byte("before"), 0600)
	m, err := snapshots.Open(project, filepath.Join(dir, subject+"-snapshots"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := m.Capture(context.Background(), []string{"note"})
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(path, []byte("after"), 0600)
	if _, err = m.Seal(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	m.Close()
	for _, op := range []string{"list", "diff", "undo", "redo"} {
		args := []string{op, subject, "--session-dir", dir}
		if op != "list" {
			args = append(args, "--id", id)
		}
		if err = changes(args); err != nil {
			t.Fatal(op, err)
		}
		want := "after"
		if op == "undo" {
			want = "before"
		}
		b, _ := os.ReadFile(path)
		if string(b) != want {
			t.Fatal(op, string(b))
		}
	}
	os.WriteFile(path, []byte("external"), 0600)
	if err = changes([]string{"undo", subject, "--session-dir", dir, "--id", id}); err == nil {
		t.Fatal("CLI overwrote external change")
	}
	m, err = snapshots.Open(project, filepath.Join(dir, subject+"-snapshots"))
	if err != nil {
		t.Fatal(err)
	}
	pending, err := m.Capture(context.Background(), []string{"note"})
	if err != nil {
		t.Fatal(err)
	}
	m.Close()
	if err = changes([]string{"acknowledge", subject, "--session-dir", dir, "--id", pending}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "external" {
		t.Fatal("acknowledgement changed file")
	}
	for _, args := range [][]string{{"changes", "--help"}, {"changes", "diff", "--help"}} {
		if err = run(args); err != nil {
			t.Fatal(err)
		}
	}
}
