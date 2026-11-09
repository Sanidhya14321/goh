package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttachFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o600)
	os.WriteFile(filepath.Join(dir, "bin"), []byte{0, 1}, 0o600)
	got, err := attachFiles(dir, []string{"a.go", "shot.png", filepath.Join(dir, "a.go")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "<file path=\"a.go\">\npackage a\n</file>") || !strings.Contains(got, " @shot.png") {
		t.Fatalf("got %q", got)
	}
	for _, bad := range []string{"../x", "bin", "missing.txt"} {
		if _, err := attachFiles(dir, []string{bad}); err == nil {
			t.Fatalf("%s accepted", bad)
		}
	}
}

func TestCommandPrompt(t *testing.T) {
	if got, _ := commandPrompt("/review", "src/a.go"); got != "/review src/a.go" {
		t.Fatal(got)
	}
	if got, _ := commandPrompt("init", ""); got != "/init" {
		t.Fatal(got)
	}
	for _, bad := range []string{"", "/", "a b", "x/y"} {
		if _, err := commandPrompt(bad, ""); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
}
