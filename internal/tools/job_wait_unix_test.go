//go:build !windows

package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// job_output with wait_seconds returns as soon as the job exits or its output
// matches until, not after the whole wait.
func TestJobOutputWaits(t *testing.T) {
	e := jobEngine(t)
	id := runJob(t, e, "job_start", map[string]string{"command": "sleep 0.3; echo built"})["id"]
	start := time.Now()
	v := runJob(t, e, "job_output", map[string]any{"id": id, "wait_seconds": 30})
	if v["waited"] != "exited" || v["status"] != "exited" || !strings.Contains(v["output"].(string), "built") || time.Since(start) > 10*time.Second {
		t.Fatal(v, time.Since(start))
	}

	id = runJob(t, e, "job_start", map[string]string{"command": "sleep 0.3; echo 'Listening on :8080'; sleep 30"})["id"]
	start = time.Now()
	v = runJob(t, e, "job_output", map[string]any{"id": id, "wait_seconds": 30, "until": `Listening on :\d+`})
	if v["waited"] != "matched" || v["status"] != "running" || time.Since(start) > 10*time.Second {
		t.Fatal(v, time.Since(start))
	}
	v = runJob(t, e, "job_output", map[string]any{"id": id, "wait_seconds": 1, "until": "never printed"})
	if v["waited"] != "timeout" || v["status"] != "running" {
		t.Fatal(v)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	b, _ := json.Marshal(map[string]any{"id": id, "wait_seconds": 30})
	start = time.Now()
	r := e.Execute(ctx, "job_output", b)
	if !strings.Contains(r.Content, `"waited":"cancelled"`) || time.Since(start) > 5*time.Second {
		t.Fatal(r.Content, time.Since(start))
	}
}

func TestJobOutputWaitValidation(t *testing.T) {
	e := jobEngine(t)
	for _, args := range []map[string]any{
		{"wait_seconds": 5},                              // no id
		{"id": "job-1", "until": "x"},                    // until without wait
		{"id": "job-1", "wait_seconds": 601},             // too long
		{"id": "job-1", "wait_seconds": 5, "until": "("}, // bad pattern
	} {
		if v := runJob(t, e, "job_output", args); v["_error"] == nil {
			t.Fatalf("%v accepted: %v", args, v)
		}
	}
	if v := runJob(t, e, "job_start", map[string]any{"command": "true", "wait_seconds": 5}); v["_error"] == nil {
		t.Fatal("job_start accepted wait_seconds")
	}
}

// A kept job writes to a file; until reads that file.
func TestKeptJobOutputWaitsForMatch(t *testing.T) {
	e := jobEngine(t)
	e.KeepJobs = true
	id := runJob(t, e, "job_start", map[string]string{"command": "sleep 0.3; echo ready; sleep 30"})["id"]
	if v := runJob(t, e, "job_output", map[string]any{"id": id, "wait_seconds": 30, "until": "^ready"}); v["waited"] != "matched" {
		t.Fatal(v)
	}
}
