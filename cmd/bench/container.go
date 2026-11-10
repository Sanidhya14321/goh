package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// cacheVolume holds downloaded dependencies (Go modules and build cache,
// pip packages) across containers, so graded runs need no network.
const cacheVolume = "goh-bench-cache"

var cacheEnv = []string{"GOMODCACHE=/cache/gomod", "GOCACHE=/cache/gobuild", "GOFLAGS=-mod=mod",
	"PIP_CACHE_DIR=/cache/pip", "PYTHONUSERBASE=/cache/pyuser", "PATH=/cache/pyuser/bin:/usr/local/go/bin:/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}

// startContainer starts a detached container with mount mounted at the same
// path. network false means --network none.
func startContainer(image, mount string, network bool) (string, error) {
	name := fmt.Sprintf("goh-bench-%d", time.Now().UnixNano())
	args := []string{"run", "-d", "--rm", "--name", name, "-v", mount + ":" + mount, "-v", cacheVolume + ":/cache",
		"--memory", "4g", "--pids-limit", "1024", "--security-opt", "no-new-privileges"}
	if !network {
		args = append(args, "--network", "none")
	}
	for _, e := range cacheEnv {
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

// csh runs command in container name (or on the host when name is empty).
func csh(name, dir, command string, seconds int) (int, string) {
	if name == "" {
		return sh(dir, command, seconds)
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
