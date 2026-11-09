package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
)

func damagedSession(t *testing.T) (dir, id string) {
	dir = t.TempDir()
	s, err := session.New(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{"first", "answer", "second"} {
		role := "user"
		if c == "answer" {
			role = "assistant"
		}
		s.Append("message", core.Message{Role: role, Content: c})
	}
	s.Close()
	path := filepath.Join(dir, s.ID+".jsonl")
	b, _ := os.ReadFile(path)
	lines := strings.Split(string(b), "\n")
	lines[2] = `{"version":1,"type":"message","data":{"role"` // corrupt the middle
	os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600)
	if _, err := session.Open(dir, s.ID); err == nil {
		t.Fatal("journal not damaged")
	}
	return dir, s.ID
}

func TestOfferSalvage(t *testing.T) {
	dir, id := damagedSession(t)
	before, _ := os.ReadFile(filepath.Join(dir, id+".jsonl"))
	var out bytes.Buffer
	if _, err := offerSalvage(strings.NewReader("n\n"), &out, dir, id, session.ErrDamaged); err == nil || !strings.Contains(err.Error(), "goh sessions salvage") {
		t.Fatal("decline must keep the error and the hint", err)
	}
	store, err := offerSalvage(strings.NewReader("y\n"), &out, dir, id, session.ErrDamaged)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if store.ID == id || !strings.Contains(out.String(), "Recovered") {
		t.Fatalf("salvage did not produce a new session: %s", out.String())
	}
	after, _ := os.ReadFile(filepath.Join(dir, id+".jsonl"))
	if !bytes.Equal(before, after) {
		t.Fatal("damaged journal was modified")
	}
}
