package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/rewind"
	"example.com/independent-harness/internal/session"
	"example.com/independent-harness/internal/snapshots"
)

func changes(args []string) error {
	usage := "changes list|diff|undo|redo|acknowledge SESSION --id CHECKPOINT [--session-dir DIR]\n       changes undo-turn|redo-turn SESSION [--session-dir DIR]\n       changes revert-turn SESSION --turn N [--session-dir DIR]"
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Println(usage)
		return nil
	}
	if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
		fmt.Println(usage)
		return nil
	}
	if len(args) < 2 {
		return errors.New(usage)
	}
	operation, subject := args[0], args[1]
	switch operation {
	case "list", "diff", "undo", "redo", "acknowledge", "undo-turn", "redo-turn", "revert-turn":
	default:
		return errors.New(usage)
	}
	f := flag.NewFlagSet("changes "+operation, flag.ContinueOnError)
	defaultDir := core.Defaults().SessionDir
	if d, ok := os.LookupEnv("HARNESS_SESSION_DIR"); ok {
		defaultDir = d
	}
	dir := f.String("session-dir", defaultDir, "private session storage")
	id := f.String("id", "", "checkpoint ID")
	turn := f.Int("turn", 0, "user turn to revert, with every later turn (revert-turn)")
	if err := f.Parse(args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	turnLevel := operation == "undo-turn" || operation == "redo-turn" || operation == "revert-turn"
	if operation == "revert-turn" && *turn < 1 {
		return errors.New(usage)
	}
	if f.NArg() != 0 || operation != "list" && !turnLevel && *id == "" {
		return errors.New(usage)
	}
	if turnLevel {
		// Turn-level rewind opens the session and checkpoints itself.
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		var result rewind.Result
		var err error
		if operation == "undo-turn" {
			result, err = rewind.Undo(ctx, *dir, subject)
		} else if operation == "revert-turn" {
			result, err = rewind.RevertTurn(ctx, *dir, subject, *turn)
		} else {
			result, err = rewind.Redo(ctx, *dir, subject)
		}
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	}
	store, err := session.Open(*dir, subject)
	if err != nil {
		return err
	}
	defer store.Close()
	manager, err := snapshots.Open(store.Project, filepath.Join(*dir, subject+"-snapshots"))
	if err != nil {
		return err
	}
	defer manager.Close()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	var value any
	switch operation {
	case "list":
		value, err = manager.List()
	case "diff":
		value, err = manager.Diff(*id)
	case "undo":
		value, err = manager.Undo(ctx, *id)
	case "redo":
		value, err = manager.Redo(ctx, *id)
	case "acknowledge":
		err = manager.Acknowledge(ctx, *id)
		value = map[string]any{"id": *id, "status": "acknowledged", "effects": "Inspected by caller; no restoration or automatic replay performed."}
	}
	if err != nil {
		return err
	}
	if operation == "undo" || operation == "redo" || operation == "acknowledge" {
		if err = store.Append("snapshot_"+operation, map[string]any{"id": *id}); err != nil {
			return err
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
