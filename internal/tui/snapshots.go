package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Sanidhya14321/goh/internal/projectsnap"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
)

// showSnapshots lists what each run changed according to whole-project
// snapshots. Restoring rewrites project files, so it stays an explicit CLI
// step; the exact commands are shown.
func (m *model) showSnapshots() {
	if m.a == nil || m.a.Store == nil {
		return
	}
	if !m.a.Config.ProjectSnapshots {
		m.log += "\n[notice] Whole-project snapshots are off. Set \"project_snapshots\": true in the config to record every run.\n"
		return
	}
	type run struct{ start, end string }
	var runs []run
	index := map[string]int{}
	for _, r := range m.a.Store.Records() {
		if r.Type != "project_snapshot" {
			continue
		}
		var d struct {
			Phase string `json:"phase"`
			Tree  string `json:"tree"`
			RunID string `json:"run_id"`
		}
		if json.Unmarshal(r.Data, &d) != nil {
			continue
		}
		if d.Phase == "start" {
			index[d.RunID] = len(runs)
			runs = append(runs, run{start: d.Tree})
		} else if i, ok := index[d.RunID]; ok && d.Phase == "end" {
			runs[i].end = d.Tree
		}
	}
	if len(runs) == 0 {
		m.log += "\n[notice] No project snapshots in this session yet.\n"
		return
	}
	snap, err := projectsnap.Open(agent.ProjectSnapshotDir(m.a.Config.SessionDir), m.a.Store.Project)
	if err != nil {
		m.log += "\n[notice] " + Safe(err.Error()) + "\n"
		return
	}
	var b strings.Builder
	b.WriteString("\n[snapshots] Project changes by run:\n")
	for i, r := range runs {
		if r.end == "" {
			fmt.Fprintf(&b, "  run %d  (interrupted: no end snapshot)\n", i+1)
			continue
		}
		changes, err := snap.Diff(context.Background(), r.start, r.end)
		if err != nil {
			fmt.Fprintf(&b, "  run %d  %s\n", i+1, Safe(err.Error()))
			continue
		}
		fmt.Fprintf(&b, "  run %d  %d file(s)\n", i+1, len(changes))
		for k, c := range changes {
			if k == 8 {
				fmt.Fprintf(&b, "      … %d more\n", len(changes)-8)
				break
			}
			fmt.Fprintf(&b, "      %s %s\n", c.Status, Safe(c.Path))
		}
	}
	id := m.a.Store.ID
	fmt.Fprintf(&b, "Restore the project to before run N (lists first):  goh snapshots restore %s N\nApply it with --yes; undo a restore with:  goh snapshots undo %s --yes\n", id, id)
	m.log += b.String()
}
