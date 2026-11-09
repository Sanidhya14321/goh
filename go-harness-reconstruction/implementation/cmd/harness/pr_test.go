package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunPR(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake gh is a shell script")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	script := "#!/bin/sh\necho \"$@\" > " + log + "\n[ \"$3\" = 13 ] && exit 1\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	opened := 0
	open := func() error { opened++; return nil }
	for _, bad := range [][]string{nil, {"abc"}, {"0"}, {"1", "2"}, {"--repo=x"}, {"https://evil.example/pull/1"}} {
		if runPR(bad, open) == nil {
			t.Fatal("accepted", bad)
		}
	}
	if err := runPR([]string{"#42"}, open); err != nil || opened != 1 {
		t.Fatal(err, opened)
	}
	if b, _ := os.ReadFile(log); strings.TrimSpace(string(b)) != "pr checkout 42" {
		t.Fatalf("gh args %q", b)
	}
	if err := runPR([]string{"13"}, open); err == nil || opened != 1 {
		t.Fatal("UI opened after a failed checkout")
	}
	t.Setenv("PATH", t.TempDir())
	if err := runPR([]string{"5"}, open); err == nil || !strings.Contains(err.Error(), "GitHub CLI") {
		t.Fatal(err)
	}
}
