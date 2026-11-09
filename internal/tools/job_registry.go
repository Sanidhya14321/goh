package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/platform"
)

// The job registry survives a hard kill of GoHarness so that background jobs
// it leaves behind can be found and stopped later. Each engine owns one file
// in <session storage>/jobs, rewritten whenever its running jobs change.

type registryEntry struct {
	ID      string `json:"id"`
	PID     int    `json:"pid"`
	Command string `json:"command"`
	Started string `json:"started"`
}

type registryFile struct {
	Owner int             `json:"owner_pid"`
	Jobs  []registryEntry `json:"jobs"`
}

// OrphanJob is a job whose GoHarness process is gone but whose process group
// still runs.
type OrphanJob struct {
	File    string `json:"-"`
	ID      string `json:"id"`
	PID     int    `json:"pid"`
	Command string `json:"command"`
	Started string `json:"started"`
}

func (e *Engine) registryDir() string { return filepath.Join(filepath.Dir(e.artifacts), "jobs") }

// saveRegistry records this engine's running jobs; called with jobs.mu held.
func (e *Engine) saveRegistry() {
	if e.jobs.registry == "" {
		if len(e.jobs.jobs) == 0 {
			return
		}
		if err := os.MkdirAll(e.registryDir(), 0o700); err != nil {
			return
		}
		e.jobs.registry = filepath.Join(e.registryDir(), fmt.Sprintf("%d-%d.json", os.Getpid(), time.Now().UnixNano()))
	}
	file := registryFile{Owner: os.Getpid()}
	for _, j := range e.jobs.jobs {
		if j.running() && j.pid > 0 {
			file.Jobs = append(file.Jobs, registryEntry{ID: j.id, PID: j.pid, Command: j.command, Started: j.started.UTC().Format(time.RFC3339)})
		}
	}
	if len(file.Jobs) == 0 {
		os.Remove(e.jobs.registry)
		return
	}
	b, _ := json.Marshal(file)
	tmp := e.jobs.registry + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		os.Rename(tmp, e.jobs.registry)
	}
}

// OrphanedJobs lists jobs left running by GoHarness processes that no longer
// exist. Registry files whose jobs have all exited are removed.
func (e *Engine) OrphanedJobs() []OrphanJob {
	entries, err := os.ReadDir(e.registryDir())
	if err != nil {
		return nil
	}
	var out []OrphanJob
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(e.registryDir(), entry.Name())
		b, err := os.ReadFile(path)
		var file registryFile
		if err != nil || len(b) > 1<<20 || json.Unmarshal(b, &file) != nil {
			continue
		}
		if file.Owner == os.Getpid() || platform.ProcessAlive(file.Owner) {
			continue // owned by a running GoHarness
		}
		live := 0
		for _, j := range file.Jobs {
			if j.PID > 0 && !platform.GroupExited(j.PID) {
				live++
				out = append(out, OrphanJob{File: path, ID: j.ID, PID: j.PID, Command: j.Command, Started: j.Started})
			}
		}
		if live == 0 {
			os.Remove(path)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Started < out[j].Started })
	return out
}

// StopOrphan stops an orphaned job's process group, but only after checking
// that the process still runs the recorded command, so a reused PID is
// never signalled. Where that cannot be checked (Windows), it refuses.
func (e *Engine) StopOrphan(o OrphanJob) error {
	cmdline, err := platform.ProcessCommand(o.PID)
	if err != nil {
		return fmt.Errorf("cannot verify process %d: %w; stop it yourself if it is yours", o.PID, err)
	}
	if !strings.Contains(cmdline, o.Command) {
		return fmt.Errorf("process %d now runs something else; it was not stopped", o.PID)
	}
	if err := platform.StopGroup(o.PID, 2*time.Second); err != nil {
		return err
	}
	if !platform.GroupExited(o.PID) {
		return errors.New("job did not stop")
	}
	e.OrphanedJobs() // prunes the registry file once its jobs are gone
	return nil
}
