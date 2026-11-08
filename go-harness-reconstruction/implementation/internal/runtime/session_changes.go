package runtime

import (
	"errors"
	"os"
	"sort"

	"example.com/independent-harness/internal/snapshots"
)

// Checkpoint summarizes one saved file checkpoint for the UI.
type Checkpoint struct {
	ID     string
	Status string
	Files  []string
}

// ErrNoCheckpoints means file checkpoints are off or none were saved.
var ErrNoCheckpoints = errors.New("no file checkpoints in this session; set \"snapshots\": true (or --snapshots) to record before/after copies of edited files")

// Checkpoints lists this session's file checkpoints, newest first. Call it
// only while no run is active.
func (a *Agent) Checkpoints() ([]Checkpoint, error) {
	if _, err := os.Stat(a.snapshotDirectory()); err != nil {
		return nil, ErrNoCheckpoints
	}
	m, err := snapshots.Open(a.Store.Project, a.snapshotDirectory())
	if err != nil {
		return nil, err
	}
	defer m.Close()
	records, err := m.List()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, ErrNoCheckpoints
	}
	sort.SliceStable(records, func(i, j int) bool { return records[i].Created.After(records[j].Created) })
	out := make([]Checkpoint, 0, len(records))
	for _, r := range records {
		c := Checkpoint{ID: r.ID, Status: r.Status}
		for _, e := range r.Before {
			c.Files = append(c.Files, e.Path)
		}
		out = append(out, c)
	}
	return out, nil
}

// CheckpointDiff returns the bounded diff of one checkpoint.
func (a *Agent) CheckpointDiff(id string) ([]snapshots.Change, error) {
	m, err := snapshots.Open(a.Store.Project, a.snapshotDirectory())
	if err != nil {
		return nil, err
	}
	defer m.Close()
	return m.Diff(id)
}
