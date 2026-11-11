package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/testutil"
)

func jobEngine(t *testing.T) *Engine {
	t.Helper()
	e, err := New(testutil.TempDir(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(e.StopJobs)
	return e
}

func runJob(t *testing.T, e *Engine, name string, args any) map[string]any {
	t.Helper()
	b, _ := json.Marshal(args)
	r := e.Execute(context.Background(), name, b)
	var v map[string]any
	if err := json.Unmarshal([]byte(r.Content), &v); err != nil {
		if !r.Error {
			t.Fatal(r.Content)
		}
		v = map[string]any{"text": r.Content}
	}
	if r.Error {
		v["_error"] = true
	}
	return v
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	for i := 0; i < 300; i++ {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not reached")
}

func TestJobOutputAndExit(t *testing.T) {
	e := jobEngine(t)
	started := runJob(t, e, "job_start", map[string]string{"command": "echo job-says-hello"})
	id, _ := started["id"].(string)
	if started["_error"] != nil || id == "" || started["status"] != "running" {
		t.Fatal(started)
	}
	waitFor(t, func() bool { return runJob(t, e, "job_output", map[string]string{"id": id})["status"] == "exited" })
	out := runJob(t, e, "job_output", map[string]string{"id": id})
	if !strings.Contains(out["output"].(string), "job-says-hello") || out["exit_code"].(float64) != 0 {
		t.Fatal(out)
	}
	list := runJob(t, e, "job_output", map[string]string{})
	if jobs := list["jobs"].([]any); len(jobs) != 1 {
		t.Fatal(list)
	}
	if v := runJob(t, e, "job_output", map[string]string{"id": "job-99"}); v["_error"] == nil {
		t.Fatal("unknown job accepted")
	}
}

func TestJobLimitsAndValidation(t *testing.T) {
	testutil.RequireUnixShell(t)
	e := jobEngine(t)
	for i := 0; i < maxJobs; i++ {
		if v := runJob(t, e, "job_start", map[string]string{"command": "sleep 60"}); v["_error"] != nil {
			t.Fatal(v)
		}
	}
	if v := runJob(t, e, "job_start", map[string]string{"command": "sleep 60"}); v["_error"] == nil {
		t.Fatal("ninth job accepted")
	}
	start := time.Now()
	e.StopJobs()
	for _, j := range e.Jobs() {
		if j["status"] != "stopped" {
			t.Fatal("job still running after StopJobs", j)
		}
	}
	if time.Since(start) > 10*time.Second {
		t.Fatal("StopJobs too slow")
	}
	for _, bad := range []string{`{"command":""}`, `{"command":"x","cwd":"../outside"}`, `{"command":"x","extra":1}`} {
		if r := e.Execute(context.Background(), "job_start", json.RawMessage(bad)); !r.Error {
			t.Fatal("accepted", bad)
		}
	}
	if _, side, _ := e.Prepare("job_output", json.RawMessage(`{}`)); side {
		t.Fatal("job_output must not need approval")
	}
	if _, side, _ := e.Prepare("job_start", json.RawMessage(`{"command":"x"}`)); !side {
		t.Fatal("job_start must need approval")
	}
}

func TestShareJobsKeepsJobsAcrossRebuiltChat(t *testing.T) {
	project, artifacts := testutil.TempDir(t), t.TempDir()
	old, err := New(project, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	started := runJob(t, old, "job_start", map[string]string{"command": "sleep 30"})
	id, _ := started["id"].(string)
	next, err := New(project, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(next.StopJobs)
	if err := old.ShareJobs(next); err != nil {
		t.Fatal(err)
	}
	if got := runJob(t, next, "job_output", map[string]string{"id": id}); got["status"] != "running" {
		t.Fatalf("shared job: %v", got)
	}
	other, _ := New(t.TempDir(), artifacts)
	if old.ShareJobs(other) == nil {
		t.Fatal("jobs moved to another project")
	}
}
