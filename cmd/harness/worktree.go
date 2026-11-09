package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var worktreeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// runWorktree implements `goh worktree NAME`: it opens the UI in a separate Git
// worktree (a sibling folder <repo>-NAME on branch goh/NAME) so an experiment
// never touches the main checkout. An existing worktree of that name is reused;
// nothing is ever removed (git worktree remove is left to the user).
// `goh worktree list` prints git worktree list.
func runWorktree(args []string, open func() error) error {
	if len(args) != 1 || (args[0] != "list" && !worktreeName.MatchString(args[0])) || strings.Contains(args[0], "..") {
		return errors.New("usage: goh worktree NAME|list (NAME: letters, digits, . _ -)")
	}
	git, err := exec.LookPath("git")
	if err != nil {
		return errors.New("goh worktree needs git on PATH")
	}
	if args[0] == "list" {
		cmd := exec.Command(git, "worktree", "list")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		return cmd.Run()
	}
	out, err := exec.Command(git, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return errors.New("goh worktree must run inside a Git repository")
	}
	root := filepath.Clean(strings.TrimSpace(string(out)))
	name := args[0]
	path := root + "-" + name
	branch := "goh/" + name
	if _, err := os.Stat(path); err == nil {
		if top, err := gitTop(git, path); err != nil || !sameDir(top, path) {
			return fmt.Errorf("%s exists and is not a worktree; choose another name", path)
		}
	} else {
		add := []string{"worktree", "add", "-b", branch, path}
		if exec.Command(git, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil {
			add = []string{"worktree", "add", path, branch}
		}
		cmd := exec.Command(git, add...)
		cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git worktree add failed: %w", err)
		}
	}
	fmt.Fprintf(os.Stderr, "Worktree %s (branch %s)\n", path, branch)
	if err := os.Chdir(path); err != nil {
		return err
	}
	return open()
}

func gitTop(git, dir string) (string, error) {
	cmd := exec.Command(git, "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func sameDir(a, b string) bool {
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	return err1 == nil && err2 == nil && ra == rb
}
