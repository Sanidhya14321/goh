package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Sanidhya14321/goh/internal/session"
)

// exportAlias implements `goh export [ID] [flags]` and `goh import FILE`, the
// top-level forms OpenCode uses. Without an ID, export picks this project's
// most recent session.
func exportAlias(command string, args []string) error {
	if command == "import" {
		return sessions(append([]string{"import"}, args...))
	}
	for _, a := range args {
		if a == "-h" || a == "--help" || a == "-help" {
			fmt.Println("usage: goh export [SESSION_ID] [--format json|markdown|html] [--include-tools] [--output FILE] [--project DIR] [--session-dir DIR]\nWithout an ID, exports the most recent session of the project (default: this directory).")
			return nil
		}
	}
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		id, err := latestHere(args)
		if err != nil {
			return err
		}
		args = append([]string{id}, args...)
	}
	// --project only chooses the session; the export itself does not take it.
	kept := args[:0:0]
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--project=") {
			continue
		}
		if args[i] == "--project" {
			i++
			continue
		}
		kept = append(kept, args[i])
	}
	return sessions(append([]string{"export"}, kept...))
}

// latestHere finds the newest session for the current directory, honouring a
// --session-dir flag or HARNESS_SESSION_DIR.
func latestHere(args []string) (string, error) {
	dir, project := configuredSessionDir(), "."
	for i, a := range args {
		if v, ok := strings.CutPrefix(a, "--session-dir="); ok {
			dir = v
		} else if a == "--session-dir" && i+1 < len(args) {
			dir = args[i+1]
		} else if v, ok := strings.CutPrefix(a, "--project="); ok {
			project = v
		} else if a == "--project" && i+1 < len(args) {
			project = args[i+1]
		}
	}
	root, err := filepath.Abs(project)
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
