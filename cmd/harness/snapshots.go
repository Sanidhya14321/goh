package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/Sanidhya14321/goh/internal/projectsnap"
	"github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/tui"
)

const snapshotsUsage = "usage: goh snapshots list SESSION | diff SESSION RUN | restore SESSION RUN [--yes] | undo SESSION [--yes] [--session-dir DIR]"

// runSnap is one run's whole-project snapshots from the journal.
type runSnap struct {
	Run   int    `json:"run"`
	Start string `json:"start"`
	End   string `json:"end,omitempty"`
}

// snapshotRuns pairs start/end project_snapshot records by run_id, in order.
func snapshotRuns(records []session.Record) []runSnap {
	var runs []runSnap
	index := map[string]int{}
	for _, r := range records {
		if r.Type != "project_snapshot" {
			continue
		}
		var d struct{ Phase, Tree, RunID string }
		if json.Unmarshal(r.Data, &struct {
			Phase *string `json:"phase"`
			Tree  *string `json:"tree"`
			RunID *string `json:"run_id"`
		}{&d.Phase, &d.Tree, &d.RunID}) != nil {
			continue
		}
		switch d.Phase {
		case "start":
			index[d.RunID] = len(runs)
			runs = append(runs, runSnap{Run: len(runs) + 1, Start: d.Tree})
		case "end":
			if i, ok := index[d.RunID]; ok {
				runs[i].End = d.Tree
			}
		}
	}
	return runs
}

// runSnapshots implements `goh snapshots`: list each run's project changes,
// show one run's diff, or put the project back to how it was before a run.
// restore only lists what would change unless --yes is given, and records the
// current state first so the restore can itself be undone.
func runSnapshots(args []string, w io.Writer) error {
	if len(args) < 2 {
		return errors.New(snapshotsUsage)
	}
	op, id := args[0], args[1]
	need := map[string]int{"list": 0, "diff": 1, "restore": 1, "undo": 0}
	n, ok := need[op]
	if !ok || len(args) < 2+n {
		return errors.New(snapshotsUsage)
	}
	runNo := 0
	if n == 1 {
		var err error
		if runNo, err = strconv.Atoi(args[2]); err != nil || runNo < 1 {
			return errors.New(snapshotsUsage)
		}
	}
	f := flag.NewFlagSet("snapshots", flag.ContinueOnError)
	defaultDir := configuredSessionDir()
	dir := f.String("session-dir", defaultDir, "private session storage")
	yes := f.Bool("yes", false, "restore (default: only list what would change)")
	if err := f.Parse(args[2+n:]); err != nil || f.NArg() != 0 {
		return errors.New(snapshotsUsage)
	}
	store, err := session.Open(*dir, id)
	if err != nil {
		return err
	}
	defer store.Close()
	project := store.Project
	records := store.Records()
	runs := snapshotRuns(records)
	if len(runs) == 0 {
		return errors.New(`no project snapshots in this session; set "project_snapshots": true in the config`)
	}
	snap, err := projectsnap.Open(runtime.ProjectSnapshotDir(*dir), project)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if op == "list" {
		for _, r := range runs {
			if r.End == "" {
				fmt.Fprintf(w, "run %d  (no end snapshot: interrupted)\n", r.Run)
				continue
			}
			changes, err := snap.Diff(ctx, r.Start, r.End)
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "run %d  %d file(s) changed\n", r.Run, len(changes))
			for i, c := range changes {
				if i == 20 {
					fmt.Fprintf(w, "    ... %d more\n", len(changes)-20)
					break
				}
				fmt.Fprintf(w, "    %s %s\n", c.Status, tui.Safe(c.Path))
			}
		}
		return nil
	}
	target, label := "", ""
	switch op {
	case "undo":
		target = lastRestoreBefore(records)
		if target == "" {
			return errors.New("no restore to undo in this session")
		}
		label = "the state before the last restore"
	default:
		if runNo > len(runs) {
			return fmt.Errorf("session has %d run(s) with snapshots", len(runs))
		}
		r := runs[runNo-1]
		if op == "diff" {
			if r.End == "" {
				return errors.New("that run has no end snapshot (it was interrupted); restore still works")
			}
			patch, cut, err := snap.Patch(ctx, r.Start, r.End, 1<<20)
			if err != nil {
				return err
			}
			fmt.Fprint(w, tui.Safe(patch))
			if cut {
				fmt.Fprintln(w, "\n[diff truncated at 1 MiB]")
			}
			return nil
		}
		target, label = r.Start, fmt.Sprintf("the start of run %d", runNo)
	}
	current, err := snap.Track(ctx)
	if err != nil {
		return err
	}
	changes, err := snap.Diff(ctx, current, target)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		fmt.Fprintf(w, "The project already matches %s.\n", label)
		return nil
	}
	verb := map[string]string{"A": "recreate", "M": "revert", "D": "remove"}
	for _, c := range changes {
		fmt.Fprintf(w, "%-9s %s\n", verb[c.Status], tui.Safe(c.Path))
	}
	if !*yes {
		fmt.Fprintf(w, "%d file(s) would change to match %s; run again with --yes. Ignored files are never touched.\n", len(changes), label)
		return nil
	}
	before, _, err := snap.Restore(ctx, target)
	if before != "" {
		// Journal the restore even if it failed part-way, so undo can find
		// the state from just before it.
		_ = store.Append("project_restore", map[string]any{"before": before, "target": target, "ok": err == nil})
	}
	if err != nil {
		return fmt.Errorf("restore stopped part-way (goh snapshots undo %s --yes returns to the previous state): %w", id, err)
	}
	fmt.Fprintf(w, "Restored the project to %s. Undo with: goh snapshots undo %s --yes\n", label, id)
	return nil
}

// lastRestoreBefore returns the state recorded just before the most recent
// restore in the journal.
func lastRestoreBefore(records []session.Record) string {
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].Type != "project_restore" {
			continue
		}
		var d struct {
			Before string `json:"before"`
		}
		if json.Unmarshal(records[i].Data, &d) == nil && d.Before != "" {
			return d.Before
		}
	}
	return ""
}
