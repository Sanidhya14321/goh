package session

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestSessionTitleCleanAndClear(t *testing.T) {
	dir, project := t.TempDir(), t.TempDir()
	id := journalWith(t, dir, project, core.Message{Role: "user", Content: "hi"})
	if err := SetTitle(dir, id, "  Fix\tthe \x1b[31mparser‮  "+strings.Repeat("x", 200)); err != nil {
		t.Fatal(err)
	}
	m, err := ReadMeta(dir, id)
	if err != nil || strings.ContainsAny(m.Title, "\x1b\t‮") || !strings.HasPrefix(m.Title, "Fix the [31mparser") || len([]rune(m.Title)) != maxTitle {
		t.Fatalf("%q %v", m.Title, err)
	}
	if err := SetTitle(dir, id, ""); err != nil {
		t.Fatal(err)
	}
	if m, _ := ReadMeta(dir, id); m.Title != "" {
		t.Fatal("empty title should clear", m.Title)
	}
	if err := SetTitle(dir, "20260101T000000-000000000000000000000000", "x"); err == nil {
		t.Fatal("title set on a missing session")
	}
}

func TestTrashRestoreMovesEverything(t *testing.T) {
	dir, project := t.TempDir(), t.TempDir()
	id := journalWith(t, dir, project, core.Message{Role: "user", Content: "hi"})
	SetTitle(dir, id, "keep me")
	os.MkdirAll(filepath.Join(dir, id+"-artifacts"), 0o700)
	os.WriteFile(filepath.Join(dir, id+"-artifacts", "out.log"), []byte("x"), 0o600)
	if err := Trash(dir, id); err != nil {
		t.Fatal(err)
	}
	if infos, _ := List(dir); len(infos) != 0 {
		t.Fatal("trashed session still listed")
	}
	for _, name := range []string{id + ".jsonl", id + ".meta.json", id + "-artifacts"} {
		if _, err := os.Stat(filepath.Join(dir, "trash", name)); err != nil {
			t.Fatal("not moved to trash:", name)
		}
	}
	if err := Restore(dir, id); err != nil {
		t.Fatal(err)
	}
	if m, _ := ReadMeta(dir, id); m.Title != "keep me" {
		t.Fatal("title lost in restore")
	}
	s, err := Open(dir, id)
	if err != nil {
		t.Fatal("restored session does not open:", err)
	}
	s.Close()
	if _, err := os.Stat(filepath.Join(dir, id+"-artifacts", "out.log")); err != nil {
		t.Fatal("artifacts not restored")
	}
}

func TestTrashRefusesOpenSession(t *testing.T) {
	dir, project := t.TempDir(), t.TempDir()
	s, err := New(dir, project)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := Trash(dir, s.ID); !errors.Is(err, ErrSessionOpen) {
		t.Fatal("an open session must not be trashed:", err)
	}
	if _, err := os.Stat(filepath.Join(dir, s.ID+".jsonl")); err != nil {
		t.Fatal("refused delete moved the journal")
	}
}

func TestRestoreRefusesOverwrite(t *testing.T) {
	dir, project := t.TempDir(), t.TempDir()
	id := journalWith(t, dir, project, core.Message{Role: "user", Content: "hi"})
	if err := Trash(dir, id); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, id+".jsonl"), []byte("other"), 0o600)
	if err := Restore(dir, id); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, id+".jsonl")); string(b) != "other" {
		t.Fatal("restore overwrote an existing file")
	}
}
