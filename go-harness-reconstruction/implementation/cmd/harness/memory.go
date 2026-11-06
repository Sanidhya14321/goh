package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"os/signal"
	"syscall"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/memory"
)

func runMemory(args []string) error {
	usage := errors.New("usage: goh memory add|list|recall|delete [flags]")
	if len(args) == 0 {
		return usage
	}
	command := args[0]
	flags := flag.NewFlagSet("memory "+command, flag.ContinueOnError)
	project := flags.String("project", ".", "project directory")
	configPath := flags.String("config", "", "user configuration file")
	sessionDir := flags.String("session-dir", "", "private session storage")
	var paths []string
	var query, id string
	switch command {
	case "add":
		flags.Func("path", "project-relative evidence file (repeat up to four times)", func(path string) error {
			paths = append(paths, path)
			return nil
		})
	case "list":
	case "recall":
		flags.StringVar(&query, "query", "", "search terms")
	case "delete":
		flags.StringVar(&id, "id", "", "note ID")
	default:
		return usage
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return usage
	}
	config, err := core.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	if *sessionDir != "" {
		config.SessionDir = *sessionDir
	}
	ledger, err := memory.New(config.SessionDir, *project)
	if err != nil {
		return err
	}
	var result any
	switch command {
	case "add":
		text, readErr := io.ReadAll(io.LimitReader(os.Stdin, memory.MaxNoteBytes+1))
		if readErr != nil {
			return readErr
		}
		if len(text) > memory.MaxNoteBytes {
			return errors.New("note input exceeds 1200 bytes")
		}
		result, err = ledger.Add(string(text), paths)
	case "list":
		result, err = ledger.List()
	case "recall":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		result, err = ledger.Recall(ctx, query)
	case "delete":
		if id == "" {
			return errors.New("delete requires --id")
		}
		return ledger.Delete(id)
	}
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
