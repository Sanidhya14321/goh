package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/tui"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
)

func sessions(args []string) error {
	usage := "sessions list [--format json|table|ids] [--project DIR]|preview ID|show ID|export ID [--format json|markdown] [--include-tools] [--output FILE]|import FILE [--project DIR]|fork ID [--before-turn N]|salvage ID|rename ID TITLE|delete ID|restore ID|prune --days N [--yes] [--session-dir DIR]"
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
	case "list", "prune":
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
	pruneDays, pruneYes := 0, false
	if command == "prune" {
		f.IntVar(&pruneDays, "days", 0, "sessions not changed for this many days (required, at least 1)")
		f.BoolVar(&pruneYes, "yes", false, "move them to the trash (default: only list them)")
	}
	exportFormat, includeTools := "json", false
	if command == "export" {
		f.StringVar(&exportFormat, "format", "json", "json portable archive, markdown transcript, or html page")
		f.BoolVar(&includeTools, "include-tools", false, "include tool calls and results in Markdown")
		f.StringVar(&output, "output", "", "new private export file; default stdout")
	}
	listFormat, listProject := "json", ""
	if command == "list" {
		f.StringVar(&listFormat, "format", "json", "json (default), table, or ids (one ID per line)")
		f.StringVar(&listProject, "project", "", "only sessions for this project directory (. for the current one)")
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
		root, err := projectFilter(listProject)
		if err != nil {
			return err
		}
		if root != "" {
			kept := infos[:0]
			for _, info := range infos {
				if info.Project == root {
					kept = append(kept, info)
				}
			}
			infos = kept
		}
		// stdout keeps its JSON array; damaged journals are reported on stderr.
		for _, d := range damaged {
			fmt.Fprintf(os.Stderr, "damaged session %s: %s (recover with: goh sessions salvage %s)\n", d.ID, d.Problem, d.ID)
		}
		switch listFormat {
		case "json":
			return encoder.Encode(infos)
		case "table":
			return writeSessionTable(os.Stdout, *dir, infos)
		case "ids":
			for _, info := range infos {
				fmt.Println(info.ID) // used by shell completion
			}
			return nil
		}
		return errors.New("format must be json, table or ids")
	case "rename":
		if err := session.SetTitle(*dir, subject, title); err != nil {
			return err
		}
		return encoder.Encode(map[string]string{"id": subject, "title": session.CleanTitle(title)})
	case "prune":
		if pruneDays < 1 {
			return errors.New("usage: goh sessions prune --days N [--yes]")
		}
		return pruneSessions(os.Stdout, *dir, time.Now().AddDate(0, 0, -pruneDays), pruneYes)
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
				return errors.New("--include-tools requires --format markdown or html")
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
		case "html":
			write = func(w io.Writer) error {
				store, err := session.Open(*dir, subject)
				if err != nil {
					return err
				}
				defer store.Close()
				return store.WriteHTMLTranscript(w, includeTools)
			}
		default:
			return errors.New("export format must be json, markdown or html")
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

// pruneSessions lists, or with yes moves to the trash, sessions whose journal
// was last changed before cutoff. Trashed sessions stay restorable; a session
// open elsewhere is skipped by session.Trash and reported.
func pruneSessions(w io.Writer, dir string, cutoff time.Time, yes bool) error {
	infos, _, err := session.ListAll(dir)
	if err != nil {
		return err
	}
	var old []session.Info
	for _, info := range infos {
		fi, err := os.Stat(filepath.Join(dir, info.ID+".jsonl"))
		if err == nil && fi.ModTime().Before(cutoff) {
			old = append(old, info)
		}
	}
	if len(old) == 0 {
		fmt.Fprintln(w, "No sessions to prune.")
		return nil
	}
	moved := 0
	for _, info := range old {
		if !yes {
			fmt.Fprintf(w, "%s  %s\n", info.ID, tui.Safe(info.Project))
			continue
		}
		if err := session.Trash(dir, info.ID); err != nil {
			fmt.Fprintf(w, "skipped %s: %v\n", info.ID, err)
			continue
		}
		moved++
	}
	if !yes {
		fmt.Fprintf(w, "%d session(s) would move to the trash; run again with --yes (goh sessions restore ID undoes it).\n", len(old))
	} else {
		fmt.Fprintf(w, "Moved %d session(s) to the trash; goh sessions restore ID brings one back.\n", moved)
	}
	return nil
}
