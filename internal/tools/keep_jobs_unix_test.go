//go:build !windows

package tools

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// A kept job writes to a log file, not a pipe, so it survives GoHarness
// exiting; job_output reads the file's tail.
func TestKeptJobWritesToLog(t *testing.T) {
	e := jobEngine(t)
	e.KeepJobs = true
	started := runJob(t, e, "job_start", map[string]string{"command": "echo first; echo ready; sleep 60"})
	if !strings.Contains(started["note"].(string), "keeps running after GoHarness exits") {
		t.Fatal(started)
	}
	id := started["id"].(string)
	waitFor(t, func() bool {
		return strings.Contains(runJob(t, e, "job_output", map[string]string{"id": id})["output"].(string), "ready")
	})
	j := e.findJob(id)
	if j.log == "" {
		t.Fatal("no log file")
	}
	if out, err := exec.Command("sh", "-c", "ls -l /proc/"+strconv.Itoa(j.pid)+"/fd/1 2>/dev/null || lsof -a -p "+strconv.Itoa(j.pid)+" -d 1 -Fn").CombinedOutput(); err == nil && strings.Contains(string(out), "pipe") {
		t.Fatalf("job stdout is a pipe: %s", out)
	}
	got := runJob(t, e, "job_output", map[string]any{"id": id, "limit": 6})
	if got["output"] != "ready\n" || got["output_truncated"] != true || got["output_bytes"].(float64) != 12 {
		t.Fatal(got)
	}
	if _, err := os.Stat(j.log); err != nil {
		t.Fatal(err)
	}
	for _, d := range e.Definitions() {
		if d.Name == "job_start" && !strings.Contains(d.Description, "keep running after GoHarness exits") {
			t.Fatal(d.Description)
		}
	}
}

func TestShellNotesStoppedBackgroundProcesses(t *testing.T) {
	e := jobEngine(t)
	left := runJob(t, e, "shell", map[string]any{"command": "sleep 30 > /dev/null 2>&1 &"})
	if !strings.Contains(left["note"].(string), "job_start") {
		t.Fatal(left)
	}
	plain := runJob(t, e, "shell", map[string]any{"command": "echo hi"})
	if _, ok := plain["note"]; ok {
		t.Fatal(plain)
	}
}
