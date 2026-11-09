package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
)

func TestSessionTable(t *testing.T) {
	dir := t.TempDir()
	s, err := session.New(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.Append("message", core.Message{Role: "user", Content: "fix the parser"})
	s.Close()
	session.SetTitle(dir, s.ID, "Parser\x1b[31m fix")
	var b bytes.Buffer
	if err := writeSessionTable(&b, dir, []session.Info{{ID: s.ID, Project: s.Project}}); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.HasPrefix(out, s.ID) || !strings.Contains(out, "Parser") || strings.Contains(out, "\x1b") {
		t.Fatalf("%q", out)
	}
	b.Reset()
	writeSessionTable(&b, dir, nil)
	if !strings.Contains(b.String(), "No saved sessions") {
		t.Fatal(b.String())
	}
}
