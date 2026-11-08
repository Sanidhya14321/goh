package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/tui"
)

// listAgents prints the built-in and configured agents, like
// `opencode agent list`, plus any project agent files that were skipped.
func listAgents(args []string) error {
	f := flag.NewFlagSet("agents", flag.ContinueOnError)
	configPath := f.String("config", "", "explicit user JSON configuration")
	project := f.String("project", ".", "project directory")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 {
		return errors.New("usage: goh agents [--project DIR] [--config FILE]")
	}
	c, err := core.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	dir, err := filepath.Abs(*project)
	if err != nil {
		return err
	}
	c, skipped := c.WithProjectAgents(dir)
	writeAgents(os.Stdout, c, skipped)
	return nil
}

func writeAgents(w io.Writer, c core.Config, skipped []core.SkippedAgentFile) {
	fmt.Fprintln(w, "build      built-in  full tool access, with approvals")
	fmt.Fprintln(w, "plan       built-in  read-only investigation and planning")
	names := make([]string, 0, len(c.Agents))
	for name := range c.Agents {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		a := c.Agents[name]
		mode := a.Mode
		if mode == "" {
			mode = "primary"
		}
		line := fmt.Sprintf("%-10s %-9s %s", tui.Safe(name), tui.Safe(mode), tui.Safe(a.Description))
		if a.Profile != "" {
			line += " (model profile " + tui.Safe(a.Profile) + ")"
		}
		fmt.Fprintln(w, line)
	}
	for _, s := range skipped {
		fmt.Fprintf(w, "skipped %s: %s\n", tui.Safe(s.Path), tui.Safe(s.Reason))
	}
}
