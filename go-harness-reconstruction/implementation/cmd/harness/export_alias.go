package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/session"
)

// exportAlias implements `goh export [ID] [flags]` and `goh import FILE`, the
// top-level forms OpenCode uses. Without an ID, export picks this project's
// most recent session.
func exportAlias(command string, args []string) error {
	if command == "import" {
		return sessions(append([]string{"import"}, args...))
	}
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		id, err := latestHere(args)
		if err != nil {
			return err
		}
		args = append([]string{id}, args...)
	}
	return sessions(append([]string{"export"}, args...))
}

// latestHere finds the newest session for the current directory, honouring a
// --session-dir flag or HARNESS_SESSION_DIR.
func latestHere(args []string) (string, error) {
	dir := core.Defaults().SessionDir
	if v, ok := os.LookupEnv("HARNESS_SESSION_DIR"); ok {
		dir = v
	}
	for i, a := range args {
		if v, ok := strings.CutPrefix(a, "--session-dir="); ok {
			dir = v
		} else if a == "--session-dir" && i+1 < len(args) {
			dir = args[i+1]
		}
	}
	root, err := filepath.Abs(".")
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		return "", err
	}
	id, err := session.Latest(dir, root)
	if err != nil || id == "" {
		return "", errors.New("no saved session for this directory; pass a session ID (goh sessions list)")
	}
	return id, nil
}
