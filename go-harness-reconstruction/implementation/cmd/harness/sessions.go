package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/session"
)

func sessions(args []string) error {
	usage := "sessions list|preview ID|show ID|export ID [--format json|markdown] [--include-tools] [--output FILE]|import FILE [--project DIR]|fork ID [--before-turn N]|salvage ID|rename ID TITLE|delete ID|restore ID [--session-dir DIR]"
	if len(args) == 0 {
		return errors.New(usage)
	}
	command := args[0]
	args = args[1:]
	subject, title := "", ""
	switch command {
	case "preview", "show", "export", "import", "fork", "salvage", "rename", "delete", "restore":
		if len(args) == 0 || args[0] == "" {
			return errors.New(usage)
		}
		if args[0] != "--help" && args[0] != "-h" {
			subject = args[0]
			args = args[1:]
		}
		if command == "rename" {
			if len(args) == 0 || strings.HasPrefix(args[0], "--") {
				return errors.New("usage: sessions rename ID TITLE (an empty TITLE restores the default)")
			}
			title, args = args[0], args[1:]
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
	exportFormat, includeTools := "json", false
	if command == "export" {
		f.StringVar(&exportFormat, "format", "json", "json portable archive or markdown transcript")
		f.BoolVar(&includeTools, "include-tools", false, "include tool calls and results in Markdown")
		f.StringVar(&output, "output", "", "new private export file; default stdout")
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
		infos, damaged, err := session.ListAll(*dir)
		if err != nil {
			return err
		}
		// stdout keeps its JSON array; damaged journals are reported on stderr.
		for _, d := range damaged {
			fmt.Fprintf(os.Stderr, "damaged session %s: %s (recover with: goh sessions salvage %s)\n", d.ID, d.Problem, d.ID)
		}
		return encoder.Encode(infos)
	case "rename":
		if err := session.SetTitle(*dir, subject, title); err != nil {
			return err
		}
		return encoder.Encode(map[string]string{"id": subject, "title": session.CleanTitle(title)})
	case "delete":
		if err := session.Trash(*dir, subject); err != nil {
			return err
		}
		return encoder.Encode(map[string]string{"id": subject, "status": "moved to trash", "restore": "goh sessions restore " + subject})
	case "restore":
		if err := session.Restore(*dir, subject); err != nil {
			return err
		}
		return encoder.Encode(map[string]string{"id": subject, "status": "restored"})
	case "salvage":
		store, report, err := session.Salvage(*dir, subject)
		if err != nil {
			return err
		}
		defer store.Close()
		return encoder.Encode(report)
	case "show":
		store, err := session.Open(*dir, subject)
		if err != nil {
			return err
		}
		defer store.Close()
		return encoder.Encode(store.Records())
	case "export":
		var write func(io.Writer) error
		switch exportFormat {
		case "json":
			if includeTools {
				return errors.New("--include-tools requires --format markdown")
			}
			write = func(w io.Writer) error { return session.Export(*dir, subject, w) }
		case "markdown":
			write = func(w io.Writer) error {
				store, err := session.Open(*dir, subject)
				if err != nil {
					return err
				}
				defer store.Close()
				return store.WriteTranscript(w, includeTools)
			}
		default:
			return errors.New("export format must be json or markdown")
		}
		if output == "" {
			return write(os.Stdout)
		}
		return session.WriteExportFile(output, write)
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
