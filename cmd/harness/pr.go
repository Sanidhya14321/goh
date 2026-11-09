package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

var prRef = regexp.MustCompile(`^(#?[1-9][0-9]{0,9}|https://github\.com/[\w.-]+/[\w.-]+/pull/[1-9][0-9]{0,9})$`)

// runPR implements `goh pr NUMBER`, like `opencode pr`: it checks the pull
// request out with the GitHub CLI (the user's own gh sign-in; GoHarness never
// sees a token) and then opens the terminal UI on that branch. gh refuses a
// checkout that would overwrite uncommitted changes.
func runPR(args []string, open func() error) error {
	if len(args) != 1 || !prRef.MatchString(args[0]) {
		return errors.New("usage: goh pr NUMBER (or a github.com pull request URL)")
	}
	ref := args[0]
	if ref[0] == '#' {
		ref = ref[1:]
	}
	gh, err := exec.LookPath("gh")
	if err != nil {
		return errors.New("goh pr needs the GitHub CLI (gh) on PATH: https://cli.github.com")
	}
	cmd := exec.Command(gh, "pr", "checkout", ref)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh pr checkout %s failed: %w", ref, err)
	}
	return open()
}
