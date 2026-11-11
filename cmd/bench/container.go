package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/platform"
)

// gradeEnv is the host environment for running model-edited code.
func gradeEnv() []string {
	var env []string
	for _, kv := range platform.ChildEnv() {
		name, _, _ := strings.Cut(kv, "=")
		if name != "SSH_AUTH_SOCK" && name != "DISPLAY" && name != "WAYLAND_DISPLAY" {
			env = append(env, kv)
		}
	}
	return env
}

// cacheVolume holds downloaded dependencies (Go modules and build cache,
// pip packages) across containers, so graded runs need no network.
const cacheVolume = "goh-bench-cache"

var cacheEnv = []string{"GOMODCACHE=/cache/gomod", "GOCACHE=/cache/gobuild", "GOFLAGS=-mod=mod",
	"PIP_CACHE_DIR=/cache/pip", "PYTHONUSERBASE=/cache/pyuser", "PATH=/cache/pyuser/bin:/usr/local/go/bin:/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}

// startContainer starts a detached container with mount mounted at the same
// path. network true is the setup container: it alone may write the shared
// dependency cache. Agent and grading containers (network false) run with
// --network none, mount the cache read-only and build in a private cache, so
// the agent cannot plant anything the grader would later read.
func startContainer(image, mount string, network bool) (string, error) {
	name := fmt.Sprintf("goh-bench-%d", time.Now().UnixNano())
	cache := cacheVolume + ":/cache"
	if !network {
		cache += ":ro"
	}
	args := []string{"run", "-d", "--rm", "--name", name, "-v", mount + ":" + mount, "-v", cache,
		"--memory", "4g", "--pids-limit", "1024", "--security-opt", "no-new-privileges"}
	if !network {
		args = append(args, "--network", "none")
	}
	for _, e := range cacheEnv {
		if !network && strings.HasPrefix(e, "GOCACHE=") {
			e = "GOCACHE=/tmp/gobuild"
		}
		args = append(args, "-e", e)
	}
	args = append(args, "--entrypoint", "sleep", image, "infinity")
	if out, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
		return "", fmt.Errorf("docker run: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return name, nil
}

func stopContainer(name string) {
	if name != "" {
		exec.Command("docker", "rm", "-f", name).Run()
	}
}

// csh runs model-touched code: in container name, or on the host when name
// is empty. On the host it gets only GoHarness's filtered child environment
// (no credentials, tokens or agent sockets); it is not sandboxed, so tasks
// with untrusted code should set an image.
func csh(name, dir, command string, seconds int) (int, string) {
	if name == "" {
		return shEnv(dir, command, seconds, gradeEnv())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "exec", "-w", dir, name, "timeout", fmt.Sprint(seconds), "/bin/sh", "-c", command)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return 0, string(out)
	}
	if ee := (*exec.ExitError)(nil); asExit(err, &ee) {
		return ee.ExitCode(), string(out)
	}
	return -1, string(out) + err.Error()
}
