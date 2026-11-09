package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRecallScopesNotesAndRejectsStaleEvidence(t *testing.T) {
	storage, project, other := t.TempDir(), t.TempDir(), t.TempDir()
	path := filepath.Join(project, "parser.go")
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("package parser\n")
	ledger, err := New(storage, project)
	if err != nil {
		t.Fatal(err)
	}
	note, err := ledger.Add("Parser tests use the fixture grammar.", []string{"parser.go"})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := New(storage, project)
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := New(storage, other)
	if err != nil {
		t.Fatal(err)
	}
	check := func(ledger *Ledger, expected int) {
		t.Helper()
		notes, err := ledger.Recall(context.Background(), "parser grammar")
		if err != nil || len(notes) != expected {
			t.Fatalf("recall returned %d notes, want %d: %v", len(notes), expected, err)
		}
	}
	check(reopened, 1)
	check(foreign, 0)
	write("package changed\n")
	check(reopened, 0)
	write("package parser\n")
	check(reopened, 1)
	if err := reopened.Delete(note.ID); err != nil {
		t.Fatal(err)
	}
	check(ledger, 0)
}

func TestMemoryRejectsSymlinkEvidence(t *testing.T) {
	project := t.TempDir()
	target := filepath.Join(t.TempDir(), "external")
	if err := os.WriteFile(target, []byte("external"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(project, "link")); err != nil {
		t.Fatal(err)
	}
	ledger, err := New(t.TempDir(), project)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Add("note", []string{"link"}); err == nil {
		t.Fatal("accepted symlink as project evidence")
	}
}
