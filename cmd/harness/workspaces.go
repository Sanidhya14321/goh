package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Sanidhya14321/goh/internal/core"
)

// runWorkspaces is goh workspaces: the trusted project folders, newest
// first, and forgetting one so goh asks again before opening it.
func runWorkspaces(args []string, out io.Writer) error {
	switch {
	case len(args) == 0 || (len(args) == 1 && args[0] == "list"):
		list := core.Workspaces()
		if len(list) == 0 {
			fmt.Fprintln(out, "No trusted folders yet. Run goh and open a project to trust it.")
			return nil
		}
		for _, w := range list {
			fmt.Fprintf(out, "%s  (opened %s)\n", displayPath(w.Path), w.Opened.Local().Format("2006-01-02 15:04"))
		}
		return nil
	case len(args) == 2 && args[0] == "forget":
		path, err := filepath.Abs(args[1])
		if err != nil {
			return err
		}
		if err := core.ForgetWorkspace(path); err != nil {
			return err
		}
		fmt.Fprintf(out, "Forgot %s; goh will ask before opening it again.\n", displayPath(path))
		return nil
	}
	return errors.New("usage: goh workspaces [list] | goh workspaces forget PATH")
}
