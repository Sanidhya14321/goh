//go:build !windows

package tools

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
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

