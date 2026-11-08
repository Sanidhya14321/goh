//go:build !windows

package tools

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"example.com/independent-harness/internal/platform"
	"example.com/independent-harness/internal/testutil"
)

func TestJobStopKillsProcessTree(t *testing.T) {
	e := jobEngine(t)
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	started := runJob(t, e, "job_start", map[string]string{"command": "sleep 60 & echo $! > " + pidFile + "; echo ready; wait"})
	id := started["id"].(string)
	waitFor(t, func() bool {
		return strings.Contains(runJob(t, e, "job_output", map[string]string{"id": id})["output"].(string), "ready")
	})
	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	grandchild, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	stopped := runJob(t, e, "job_stop", map[string]string{"id": id})
	if stopped["status"] != "stopped" {
		t.Fatal(stopped)
	}
	waitFor(t, func() bool { return syscall.Kill(grandchild, 0) != nil })
}

func TestOrphanedJobsFoundAndStoppedSafely(t *testing.T) {
	sessions := t.TempDir()
	project := testutil.TempDir(t)
	a, err := New(project, filepath.Join(sessions, "A-artifacts"))
	if err != nil {
		t.Fatal(err)
	}
	started := runJob(t, a, "job_start", map[string]string{"command": "sleep 60"})
	if started["_error"] != nil {
		t.Fatal(started)
	}
	files, _ := filepath.Glob(filepath.Join(sessions, "jobs", "*.json"))
	if len(files) != 1 {
		t.Fatal("job not registered", files)
	}
	// Simulate GoHarness being killed: the registry owner is a dead process.
	dead := exec.Command("true")
	dead.Run()
	b, _ := os.ReadFile(files[0])
	var reg registryFile
	json.Unmarshal(b, &reg)
	reg.Owner = dead.Process.Pid
	b, _ = json.Marshal(reg)
	os.WriteFile(files[0], b, 0o600)

	other, _ := New(project, filepath.Join(sessions, "B-artifacts"))
	orphans := other.OrphanedJobs()
	if len(orphans) != 1 || orphans[0].Command != "sleep 60" {
		t.Fatal("orphan not found", orphans)
	}
	tampered := orphans[0]
	tampered.Command = "some other program"
	if err := other.StopOrphan(tampered); err == nil || !strings.Contains(err.Error(), "something else") {
		t.Fatal("must refuse when the process command does not match", err)
	}
	if platformGroupExited(orphans[0].PID) {
		t.Fatal("refusal still stopped the process")
	}
	if err := other.StopOrphan(orphans[0]); err != nil {
		t.Fatal(err)
	}
	if !platformGroupExited(orphans[0].PID) {
		t.Fatal("orphan still running")
	}
	if left, _ := filepath.Glob(filepath.Join(sessions, "jobs", "*.json")); len(left) != 0 {
		t.Fatal("registry not pruned", left)
	}
	a.StopJobs()
}

func platformGroupExited(pid int) bool { return platform.GroupExited(pid) }
