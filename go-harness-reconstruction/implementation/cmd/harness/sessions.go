package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/session"
)

func sessions(args []string) error {
	usage := "sessions list|preview ID|show ID|export ID [--output FILE]|import FILE [--project DIR]|fork ID [--before-turn N] [--session-dir DIR]"
	if len(args) == 0 {
		return errors.New(usage)
	}
	command := args[0]
	args = args[1:]
	subject := ""
	switch command {
	case "preview", "show", "export", "import", "fork":
		if len(args) == 0 || args[0] == "" {
			return errors.New(usage)
		}
		if args[0] != "--help" && args[0] != "-h" {
			subject = args[0]
			args = args[1:]
		}
	case "list":
	default:
		return errors.New(usage)
	}
	f := flag.NewFlagSet("sessions "+command, flag.ContinueOnError)
	defaultDir := core.Defaults().SessionDir
	if value, ok := os.LookupEnv("HARNESS_SESSION_DIR"); ok {
		defaultDir = value
	}
	dir := f.String("session-dir", defaultDir, "private session storage")
	output, project := "", ""
	beforeTurn := 0
	if command == "fork" {
		f.IntVar(&beforeTurn, "before-turn", 0, "branch before this one-based user turn; zero copies the full session")
	}
	if command == "export" {
		f.StringVar(&output, "output", "", "new private portable JSON file; default stdout")
	}
	if command == "import" {
		f.StringVar(&project, "project", "", "explicit existing project directory for relocated import")
	}
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() > 0 {
		return errors.New("unexpected positional arguments")
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	switch command {
	case "preview":
		info, err := session.Preview(*dir, subject)
		if err != nil {
			return err
		}
		return encoder.Encode(info)
	case "list":
		infos, err := session.List(*dir)
		if err != nil {
			return err
		}
		return encoder.Encode(infos)
	case "show":
		store, err := session.Open(*dir, subject)
		if err != nil {
			return err
		}
		defer store.Close()
		return encoder.Encode(store.Records())
	case "export":
		if output == "" {
			return session.Export(*dir, subject, os.Stdout)
		}
		return exportSessionFile(*dir, subject, output)
	case "import":
		if project != "" {
			var err error
			project, err = filepath.Abs(project)
			if err != nil {
				return err
			}
		}
		file, err := os.Open(subject)
		if err != nil {
			return err
		}
		defer file.Close()
		store, err := session.Import(*dir, file, project)
		if err != nil {
			return err
		}
		defer store.Close()
		return encoder.Encode(session.Info{ID: store.ID, Project: store.Project})
	case "fork":
		store, _, err := session.ForkBefore(*dir, subject, beforeTurn)
		if err != nil {
			return err
		}
		defer store.Close()
		return encoder.Encode(session.Info{ID: store.ID, Project: store.Project})
	}
	return errors.New(usage)
}

func exportSessionFile(dir, id, output string) (err error) {
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(output), ".harness-export-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err = session.Export(dir, id, file); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	// Publish complete bytes without replacing existing files or symlinks.
	if err = os.Link(file.Name(), output); err != nil {
		return fmt.Errorf("export destination must not exist: %w", err)
	}
	directory, err := os.Open(filepath.Dir(output))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
