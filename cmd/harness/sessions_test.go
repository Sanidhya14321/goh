package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/Sanidhya14321/goh/internal/testutil"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/session"
)

func TestSessionsPortableFileAndNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	store, err := session.New(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id := store.ID
	if err := store.Append("message", map[string]string{"role": "user", "content": "Original task"}); err != nil {
		t.Fatal(err)
	}
	store.Close()
	output := filepath.Join(t.TempDir(), "session.json")
	if err := sessions([]string{"export", id, "--session-dir", dir, "--output", output}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if testutil.UnixModes() && info.Mode().Perm() != 0600 {
		t.Fatal("export file not private")
	}
	var portable session.Portable
	if err := json.Unmarshal(before, &portable); err != nil {
		t.Fatal(err)
	}
	if portable.SourceSessionID != id {
		t.Fatal("wrong archive identity")
	}
	if err := sessions([]string{"export", id, "--session-dir", dir, "--output", output}); err == nil {
		t.Fatal("overwrote existing archive")
	}
	after, _ := os.ReadFile(output)
	if string(after) != string(before) {
		t.Fatal("existing archive modified")
	}
	destination := t.TempDir()
	if err := sessions([]string{"import", output, "--session-dir", destination}); err != nil {
		t.Fatal(err)
	}
	infos, err := session.List(destination)
	if err != nil || len(infos) != 1 || infos[0].ID == id {
		t.Fatalf("imported sessions %+v %v", infos, err)
	}
	if err := sessions([]string{"fork", id, "--session-dir", dir}); err != nil {
		t.Fatal(err)
	}
	infos, err = session.List(dir)
	if err != nil || len(infos) != 2 {
		t.Fatalf("forked sessions %+v %v", infos, err)
	}
	if err := sessions([]string{"list", "--session-dir", dir, "unexpected"}); err == nil {
		t.Fatal("extra arguments ignored")
	}
}

func TestHelpBeforeSessionIdentity(t *testing.T) {
	for _, args := range [][]string{{"compact", "--help"}, {"resume", "--help"}, {"sessions", "export", "--help"}, {"sessions", "import", "--help"}} {
		if e := run(args); e != nil {
			t.Fatalf("%v: %v", args, e)
		}
	}
}

func TestSessionsSalvageCLI(t *testing.T) {
	dir := t.TempDir()
	store, err := session.New(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id := store.ID
	for _, c := range []string{"keep me", "lost"} {
		if err := store.Append("message", map[string]string{"role": "user", "content": c}); err != nil {
			t.Fatal(err)
		}
	}
	store.Close()
	path := filepath.Join(dir, id+".jsonl")
	b, _ := os.ReadFile(path)
	lines := strings.SplitAfter(string(b), "\n")
	lines[2] = "NOT JSON\n" // damage the second prompt
	os.WriteFile(path, []byte(strings.Join(lines, "")), 0o600)

	if _, err := session.Open(dir, id); !errors.Is(err, session.ErrDamaged) {
		t.Fatal("damage not classified", err)
	}
	if err := sessions([]string{"list", "--session-dir", dir}); err != nil {
		t.Fatal("list must not fail because one journal is damaged:", err)
	}
	if err := sessions([]string{"salvage", id, "--session-dir", dir}); err != nil {
		t.Fatal(err)
	}
	// List reads headers only; the original's header is intact, so both are
	// listed and the damage is reported when the original is opened.
	infos, err := session.List(dir)
	newID := ""
	for _, info := range infos {
		if info.ID != id {
			newID = info.ID
		}
	}
	if err != nil || len(infos) != 2 || newID == "" {
		t.Fatal("salvaged session should be listed", infos, err)
	}
	salvaged, err := session.Open(dir, newID)
	if err != nil {
		t.Fatal(err)
	}
	defer salvaged.Close()
	if turns := salvaged.RecentUserTurns(5); len(turns) != 1 || turns[0].Prompt != "keep me" {
		t.Fatal(turns)
	}
}

func TestSessionsRenameDeleteRestoreCLI(t *testing.T) {
	dir := t.TempDir()
	store, err := session.New(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id := store.ID
	store.Close()
	if err := sessions([]string{"rename", id, "Parser work", "--session-dir", dir}); err != nil {
		t.Fatal(err)
	}
	if m, _ := session.ReadMeta(dir, id); m.Title != "Parser work" {
		t.Fatal(m)
	}
	if err := sessions([]string{"rename", id, "--session-dir", dir}); err == nil {
		t.Fatal("rename without a title accepted")
	}
	if err := sessions([]string{"delete", id, "--session-dir", dir}); err != nil {
		t.Fatal(err)
	}
	if infos, _ := session.List(dir); len(infos) != 0 {
		t.Fatal("deleted session still listed")
	}
	if err := sessions([]string{"restore", id, "--session-dir", dir}); err != nil {
		t.Fatal(err)
	}
	if infos, _ := session.List(dir); len(infos) != 1 {
		t.Fatal("restored session not listed")
	}
}

func TestSessionsPruneListsThenTrashes(t *testing.T) {
	dir := t.TempDir()
	mk := func(age time.Duration) string {
		s, err := session.New(dir, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		s.Close()
		when := time.Now().Add(-age)
		os.Chtimes(filepath.Join(dir, s.ID+".jsonl"), when, when)
		return s.ID
	}
	old, fresh := mk(40*24*time.Hour), mk(time.Hour)
	var out bytes.Buffer
	if err := pruneSessions(&out, dir, time.Now().AddDate(0, 0, -30), false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), old) || strings.Contains(out.String(), fresh) {
		t.Fatal(out.String())
	}
	if infos, _ := session.List(dir); len(infos) != 2 {
		t.Fatal("listing removed sessions")
	}
	if err := pruneSessions(&out, dir, time.Now().AddDate(0, 0, -30), true); err != nil {
		t.Fatal(err)
	}
	if infos, _ := session.List(dir); len(infos) != 1 || infos[0].ID != fresh {
		t.Fatal("wrong sessions remain", infos)
	}
	if err := session.Restore(dir, old); err != nil {
		t.Fatal("pruned session not restorable", err)
	}
	if sessions([]string{"prune", "--session-dir", dir}) == nil {
		t.Fatal("prune without --days accepted")
	}
}

func TestSessionsListProjectFilter(t *testing.T) {
	dir := t.TempDir()
	projA, projB := testutil.TempDir(t), testutil.TempDir(t)
	ids := map[string]string{}
	for name, p := range map[string]string{"a": projA, "b": projB} {
		s, err := session.New(dir, p)
		if err != nil {
			t.Fatal(err)
		}
		ids[name] = s.ID
		s.Close()
	}
	r, w, _ := os.Pipe()
	saved := os.Stdout
	os.Stdout = w
	err := sessions([]string{"list", "--format", "ids", "--project", projA, "--session-dir", dir})
	os.Stdout = saved
	w.Close()
	out, _ := io.ReadAll(r)
	if err != nil || strings.TrimSpace(string(out)) != ids["a"] {
		t.Fatalf("%v %q want %s", err, out, ids["a"])
	}
}
