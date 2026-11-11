package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	run := func(dir string, a ...string) {
		cmd := exec.Command("git", a...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatal(a, err, string(out))
		}
	}
	os.Mkdir(repo, 0o755)
	run(repo, "init", "-q")
	os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a"), 0o644)
	run(repo, "add", ".")
	run(repo, "commit", "-qm", "init")
	wd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(wd) })
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	opened := ""
	open := func() error { opened, _ = os.Getwd(); return nil }
	for _, bad := range [][]string{nil, {"-x"}, {"a/b"}, {"../x"}, {"a", "b"}, {"a..b"}} {
		if runWorktree(bad, open) == nil {
			t.Fatal("accepted", bad)
		}
	}
	if err := runWorktree([]string{"exp"}, open); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(base, "repo-exp")
	if !sameDir(opened, want) {
		t.Fatalf("opened in %q", opened)
	}
	if _, err := os.Stat(filepath.Join(want, "a.txt")); err != nil {
		t.Fatal(err)
	}
	os.Chdir(repo)
	opened = ""
	if err := runWorktree([]string{"exp"}, open); err != nil || !sameDir(opened, want) {
		t.Fatal("reuse failed", err, opened)
	}
	os.Mkdir(filepath.Join(base, "repo-plain"), 0o755)
	os.Chdir(repo)
	if err := runWorktree([]string{"plain"}, open); err == nil || !strings.Contains(err.Error(), "not a worktree") {
		t.Fatal(err)
	}
	// A separate repository at the expected path is not this repo's worktree.
	other := filepath.Join(base, "repo-other")
	os.Mkdir(other, 0o755)
	run(other, "init", "-q")
	os.Chdir(repo)
	if err := runWorktree([]string{"other"}, open); err == nil || !strings.Contains(err.Error(), "not a worktree") {
		t.Fatal("unrelated repository accepted:", err)
	}
}
