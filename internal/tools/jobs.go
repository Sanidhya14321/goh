package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Sanidhya14321/goh/internal/platform"
)

const (
	maxJobs      = 8
	jobTailBytes = 64 << 10
)

// job is a background shell process owned by this GoHarness process. Jobs
// never outlive it: StopJobs runs at exit, and a resumed session has no jobs
// (nothing is restarted or replayed).
type job struct {
	id, command, cwd string
	started          time.Time
	pid              int
	mu               sync.Mutex
	tail             []byte // last jobTailBytes of combined output
	total            int64
	done             chan struct{}
	exitCode         int
	state            string // running, exited, stopped
	ended            time.Time
}

func (j *job) Write(p []byte) (int, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.total += int64(len(p))
	j.tail = append(j.tail, p...)
	if over := len(j.tail) - jobTailBytes; over > 0 {
		j.tail = append([]byte(nil), j.tail[over:]...)
	}
	return len(p), nil
}

type jobTable struct {
	mu       sync.Mutex
	jobs     map[string]*job
	next     int
	registry string // this engine's file in the job registry
}

func jobDefinitions(s func(map[string]any, ...string) map[string]any, str func() map[string]any, number func(int, int) map[string]any) []Definition {
	return []Definition{
		{"job_start", "Start a long-running shell command in the background (dev server, watcher) and return a job id at once. Not sandboxed. Jobs stop when GoHarness exits and are never restarted. At most 8 at a time.", s(map[string]any{"command": str(), "cwd": str()}, "command")},
		{"job_output", "Read a background job's state and the tail of its output (last 64KiB retained). Without id, list jobs.", s(map[string]any{"id": str(), "limit": number(1, jobTailBytes)})},
		{"job_stop", "Stop a background job and its whole process group.", s(map[string]any{"id": str()}, "id")},
	}
}

type jobArgs struct {
	ID      string `json:"id"`
	Command string `json:"command"`
	Cwd     string `json:"cwd"`
	Limit   int    `json:"limit"`
}

func decodeJob(name string, raw json.RawMessage) (jobArgs, error) {
	var a jobArgs
	if len(raw) > 1<<20 {
		return a, errors.New("arguments exceed 1MiB")
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	if err := d.Decode(&a); err != nil {
		return a, fmt.Errorf("%s arguments: %w", name, err)
	}
	switch name {
	case "job_start":
		if strings.TrimSpace(a.Command) == "" || a.ID != "" || a.Limit != 0 {
			return a, errors.New("job_start takes command and optional cwd")
		}
	case "job_stop":
		if a.ID == "" || a.Command != "" || a.Cwd != "" || a.Limit != 0 {
			return a, errors.New("job_stop takes id")
		}
	case "job_output":
		if a.Command != "" || a.Cwd != "" || a.Limit < 0 || a.Limit > jobTailBytes {
			return a, errors.New("job_output takes optional id and limit")
		}
	}
	return a, nil
}

func isJobTool(name string) bool {
	return name == "job_start" || name == "job_output" || name == "job_stop"
}

func (e *Engine) prepareJob(name string, raw json.RawMessage) (string, bool, error) {
	a, err := decodeJob(name, raw)
	if err != nil {
		return "", name != "job_output", err
	}
	switch name {
	case "job_start":
		cwd, err := e.jobDir(a.Cwd)
		if err != nil {
			return "", true, err
		}
		return fmt.Sprintf("Background command: %s\nDirectory: %s\nRuns until stopped or GoHarness exits.", a.Command, cwd), true, nil
	case "job_stop":
		return "Stop background job " + a.ID, true, nil
	}
	return "Read background job output", false, nil
}

func (e *Engine) jobDir(cwd string) (string, error) {
	cwd, err := clean(cwd, true)
	if err != nil {
		return "", err
	}
	r, err := os.OpenRoot(e.root)
	if err != nil {
		return "", err
	}
	defer r.Close()
	if err := checked(r, cwd, false); err != nil {
		return "", err
	}
	if st, err := r.Stat(cwd); err != nil || !st.IsDir() {
		return "", errors.New("cwd must be a project directory")
	}
	return filepath.Join(e.root, cwd), nil
}

func (e *Engine) executeJob(ctx context.Context, name string, raw json.RawMessage) Result {
	if _, _, err := e.prepareJob(name, raw); err != nil {
		return failure(err)
	}
	a, _ := decodeJob(name, raw)
	switch name {
	case "job_start":
		return e.startJob(ctx, a)
	case "job_stop":
		j := e.findJob(a.ID)
		if j == nil {
			return failure(fmt.Errorf("no background job %q in this GoHarness process (jobs do not survive a restart)", a.ID))
		}
		stopJob(j)
		return result(j.summary(0))
	}
	if a.ID == "" {
		return result(map[string]any{"jobs": e.Jobs()})
	}
	j := e.findJob(a.ID)
	if j == nil {
		return failure(fmt.Errorf("no background job %q in this GoHarness process (jobs do not survive a restart)", a.ID))
	}
	limit := a.Limit
	if limit == 0 {
		limit = 8192
	}
	return result(j.summary(limit))
}

func (e *Engine) startJob(ctx context.Context, a jobArgs) Result {
	if err := ctx.Err(); err != nil {
		return failure(err)
	}
	dir, err := e.jobDir(a.Cwd)
	if err != nil {
		return failure(err)
	}
	e.jobs.mu.Lock()
	running := 0
	for _, j := range e.jobs.jobs {
		if j.running() {
			running++
		}
	}
	if running >= maxJobs {
		e.jobs.mu.Unlock()
		return failure(fmt.Errorf("%d background jobs are already running; stop one first", maxJobs))
	}
	e.jobs.next++
	j := &job{id: fmt.Sprintf("job-%d", e.jobs.next), command: a.Command, cwd: dir, started: time.Now(), done: make(chan struct{}), state: "running"}
	if e.jobs.jobs == nil {
		e.jobs.jobs = map[string]*job{}
	}
	e.jobs.jobs[j.id] = j
	e.jobs.mu.Unlock()

	cmd := shellCommand(a.Command)
	cmd.Dir = dir
	platform.NewGroup(cmd)
	reader, writer, err := os.Pipe()
	if err != nil {
		e.dropJob(j.id)
		return failure(err)
	}
	cmd.Stdout, cmd.Stderr = writer, writer
	if err := cmd.Start(); err != nil {
		reader.Close()
		writer.Close()
		e.dropJob(j.id)
		return failure(err)
	}
	writer.Close()
	j.pid = cmd.Process.Pid
	e.jobs.mu.Lock()
	e.saveRegistry()
	e.jobs.mu.Unlock()
	copied := make(chan struct{})
	go func() { _, _ = io.Copy(j, reader); reader.Close(); close(copied) }()
	go func() {
		err := cmd.Wait()
		// Descendants may hold the pipe; settle the group so output ends.
		platform.TerminateGroup(j.pid)
		select {
		case <-copied:
		case <-time.After(2 * time.Second):
			platform.KillGroup(j.pid)
			reader.Close()
		}
		j.mu.Lock()
		j.exitCode = 0
		if err != nil {
			j.exitCode = -1
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				j.exitCode = exit.ExitCode()
			}
		}
		if j.state == "running" {
			j.state = "exited"
		}
		j.ended = time.Now()
		j.mu.Unlock()
		close(j.done)
		e.jobs.mu.Lock()
		e.saveRegistry()
		e.jobs.mu.Unlock()
	}()
	return result(map[string]any{"id": j.id, "status": "running", "command": a.Command, "directory": dir,
		"note": "Use job_output to read output and job_stop to stop it. It stops when GoHarness exits."})
}

func (j *job) running() bool {
	select {
	case <-j.done:
		return false
	default:
		return true
	}
}

func stopJob(j *job) {
	if !j.running() {
		return
	}
	j.mu.Lock()
	j.state = "stopped"
	j.mu.Unlock()
	platform.TerminateGroup(j.pid)
	select {
	case <-j.done:
	case <-time.After(2 * time.Second):
		platform.KillGroup(j.pid)
		select {
		case <-j.done:
		case <-time.After(2 * time.Second):
		}
	}
}

func (j *job) summary(limit int) map[string]any {
	j.mu.Lock()
	defer j.mu.Unlock()
	state := j.state
	if state == "running" && !j.running() {
		state = "exited"
	}
	v := map[string]any{"id": j.id, "command": j.command, "status": state, "output_bytes": j.total,
		"started": j.started.UTC().Format(time.RFC3339)}
	if state != "running" {
		v["exit_code"] = j.exitCode
	}
	if limit > 0 {
		tail := j.tail
		if len(tail) > limit {
			tail = tail[len(tail)-limit:]
		}
		v["output"] = strings.ToValidUTF8(string(tail), "�")
		v["output_truncated"] = j.total > int64(len(tail))
	}
	return v
}

func (e *Engine) findJob(id string) *job {
	e.jobs.mu.Lock()
	defer e.jobs.mu.Unlock()
	return e.jobs.jobs[id]
}

func (e *Engine) dropJob(id string) {
	e.jobs.mu.Lock()
	defer e.jobs.mu.Unlock()
	delete(e.jobs.jobs, id)
}

// Jobs summarizes background jobs, oldest first.
func (e *Engine) Jobs() []map[string]any {
	e.jobs.mu.Lock()
	list := make([]*job, 0, len(e.jobs.jobs))
	for _, j := range e.jobs.jobs {
		list = append(list, j)
	}
	e.jobs.mu.Unlock()
	sort.Slice(list, func(a, b int) bool { return list[a].started.Before(list[b].started) })
	out := make([]map[string]any, 0, len(list))
	for _, j := range list {
		out = append(out, j.summary(0))
	}
	return out
}

// StopJobs stops every running background job. Call it before exiting.
func (e *Engine) StopJobs() {
	e.jobs.mu.Lock()
	list := make([]*job, 0, len(e.jobs.jobs))
	for _, j := range e.jobs.jobs {
		list = append(list, j)
	}
	e.jobs.mu.Unlock()
	var wg sync.WaitGroup
	for _, j := range list {
		wg.Add(1)
		go func(j *job) { defer wg.Done(); stopJob(j) }(j)
	}
	wg.Wait()
}
