package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
)

func TestExportAliasDefaultsToLatestSession(t *testing.T) {
	dir, project := t.TempDir(), t.TempDir()
	t.Chdir(project)
	root, _ := filepath.EvalSymlinks(project)
	s, err := session.New(dir, root)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Append("message", core.Message{Role: "user", Content: "hello export"}); err != nil {
		t.Fatal(err)
	}
	s.Close()
	out := filepath.Join(t.TempDir(), "x.md")
	if err := exportAlias("export", []string{"--session-dir", dir, "--format", "markdown", "--output", out}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil || !strings.Contains(string(b), "hello export") {
		t.Fatalf("%v %s", err, b)
	}
	empty := t.TempDir()
	if err := exportAlias("export", []string{"--session-dir", empty}); err == nil || !strings.Contains(err.Error(), "no saved session") {
		t.Fatal(err)
	}
}
