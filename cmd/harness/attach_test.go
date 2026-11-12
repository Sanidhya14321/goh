package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

func TestAttachFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o600)
	os.WriteFile(filepath.Join(dir, "bin"), []byte{0, 1}, 0o600)
	got, err := attachFiles(dir, []string{"a.go", "shot.png", filepath.Join(dir, "a.go")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "<file path=\"a.go\" bytes=9>\npackage a\n</file>") || !strings.Contains(got, " @shot.png") {
		t.Fatalf("got %q", got)
	}
	for _, bad := range []string{"../x", "bin", "missing.txt"} {
		if _, err := attachFiles(dir, []string{bad}, nil); err == nil {
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

func TestAttachFilesRefusesDeniedAndSpecialFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "deny.txt"), []byte("secret\n"), 0o600)
	os.Symlink(".", filepath.Join(dir, "loop"))
	denied := func(rel string, _ bool) bool { return rel == "deny.txt" }
	bad := []string{"deny.txt", "loop/deny.txt"}
	if _, err := os.Stat(filepath.Join(dir, "DENY.TXT")); err == nil {
		bad = append(bad, "DENY.TXT")
	}
	for _, p := range bad {
		if _, err := attachFiles(dir, []string{p}, denied); err == nil || !strings.Contains(err.Error(), "denied") {
			t.Fatalf("%s: %v", p, err)
		}
	}
	if err := testutil.Mkfifo(t, filepath.Join(dir, "pipe")); err == nil {
		if _, err := attachFiles(dir, []string{"pipe"}, nil); err == nil || !strings.Contains(err.Error(), "not a regular file") {
			t.Fatalf("pipe: %v", err)
		}
	}
}

func TestWithoutStartupFlags(t *testing.T) {
	in := []string{"chat", "--continue", "--fork", "--title", "T", "--session=abc", "-session", "x", "--acknowledge-uncertain", "--profile=p", "--model", "m", "-acknowledge-uncertain=true"}
	got := strings.Join(withoutStartupFlags(in), " ")
	if got != "chat --profile=p --model m" {
		t.Fatalf("%q", got)
	}
}

func TestSessionCommandsUseConfiguredSessionDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	os.Unsetenv("HARNESS_SESSION_DIR")
	path, err := core.UserConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(t.TempDir(), "my-sessions")
	os.MkdirAll(filepath.Dir(path), 0o700)
	if err := os.WriteFile(path, []byte(`{"session_dir": "`+filepath.ToSlash(want)+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := configuredSessionDir(); filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("got %q want %q", got, want)
	}
}
