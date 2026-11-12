package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The agent's container must not reach the grade tree or outlive its run.
// Runs only where Docker and a local alpine image are available.
func TestAgentContainerCannotReachGradeTree(t *testing.T) {
	if exec.Command("docker", "image", "inspect", "alpine:latest").Run() != nil {
		t.Skip("docker or the alpine:latest image is not available")
	}
	// Docker Desktop shares only some host folders (not macOS's temp), so
	// work beside the test; a folder the daemon cannot see skips below.
	base, err := os.MkdirTemp(".", ".container-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(base)
	if base, err = filepath.Abs(base); err != nil {
		t.Fatal(err)
	}
	work, grade := filepath.Join(base, "agent"), filepath.Join(base, "grade")
	os.MkdirAll(work, 0o755)
	os.MkdirAll(grade, 0o755)
	box, err := startContainer("alpine:latest", work, false)
	if err != nil {
		t.Skip("cannot start a container: ", err)
	}
	defer stopContainer(box)
	_, out := csh(box, work, "echo planted > ../grade/marker; echo planted > "+grade+"/marker2; echo ok > done", 30)
	if b, err := os.ReadFile(filepath.Join(work, "done")); err != nil || string(b) != "ok\n" {
		t.Skipf("the daemon cannot see this folder (%v): %s", err, out)
	}
	for _, m := range []string{"marker", "marker2"} {
		if _, err := os.Stat(filepath.Join(grade, m)); err == nil {
			t.Fatalf("agent container wrote %s into the grade tree", m)
		}
	}
	stopContainer(box)
	if out, _ := exec.Command("docker", "ps", "-q", "--filter", "name="+box).Output(); len(out) > 0 {
		t.Fatal("agent container still running")
	}
}
