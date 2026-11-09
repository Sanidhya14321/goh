package runtime

import (
	"context"
	"path/filepath"

	"github.com/Sanidhya14321/goh/internal/projectsnap"
)

// ProjectSnapshotDir is where shadow repositories for whole-project
// snapshots live: beside the session directory, never inside a project.
func ProjectSnapshotDir(sessionDir string) string {
	return filepath.Join(filepath.Dir(sessionDir), "project-snapshots")
}

// recordProjectSnapshot journals the project tree at the start or end of a
// top-level run. It is best effort: a failure is journaled and reported but
// never stops the run.
func (a *Agent) recordProjectSnapshot(ctx context.Context, phase string) {
	if !a.Config.ProjectSnapshots || a.depth > 0 {
		return
	}
	store, err := projectsnap.Open(ProjectSnapshotDir(a.Config.SessionDir), a.Store.Project)
	var tree string
	if err == nil {
		tree, err = store.Track(ctx)
	}
	if err != nil {
		d := map[string]any{"phase": phase, "error": err.Error()}
		_ = a.Store.Append("project_snapshot_failed", d)
		a.event("project_snapshot_failed", d)
		return
	}
	d := map[string]any{"phase": phase, "tree": tree, "run_id": a.runID}
	_ = a.Store.Append("project_snapshot", d)
	a.event("project_snapshot", d)
}
