package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"example.com/independent-harness/internal/session"
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
	if info.Mode().Perm() != 0600 {
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
