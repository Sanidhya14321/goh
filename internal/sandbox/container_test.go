package sandbox

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestWrapContainerArgs(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}
	cmd := exec.Command("/bin/sh", "-c", "echo hi")
	cmd.Dir = "/work/p"
	cmd.Env = []string{"PATH=/usr/bin", "SECRET_TOKEN=x"}
	if err := Wrap(cmd, &Policy{Container: "goh-c1"}); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(cmd.Args[1:], " ")
	if got != "exec -i -w /work/p goh-c1 /bin/sh -c echo hi" {
		t.Fatalf("args %q", got)
	}
	for _, kv := range cmd.Env {
		if strings.HasPrefix(kv, "SECRET_TOKEN") {
			t.Fatal("host environment passed to docker")
		}
	}
	if err := Wrap(exec.Command("true"), &Policy{Container: "bad name;rm"}); err == nil {
		t.Fatal("invalid name accepted")
	}
}

// Runs only where a container named by GOH_TEST_CONTAINER is running.
func TestContainerExec(t *testing.T) {
	name := os.Getenv("GOH_TEST_CONTAINER")
	if name == "" {
		t.Skip("set GOH_TEST_CONTAINER to a running container")
	}
	cmd := exec.Command("/bin/sh", "-c", "cat /etc/os-release | head -1; test ! -e /Users")
	if err := Wrap(cmd, &Policy{Container: name}); err != nil {
		t.Fatal(err)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v %s", err, out)
	}
}
