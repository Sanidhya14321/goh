package sandbox

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var containerName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

// wrapContainer runs cmd inside a running Docker container with docker exec,
// in the same working directory (the project is mounted at the same path).
// The command gets the container's environment, not the host's; only the
// docker client itself receives the DOCKER_* settings it needs.
func wrapContainer(cmd *exec.Cmd, name string) error {
	if !containerName.MatchString(name) {
		return errors.New("sandbox container name is invalid")
	}
	docker, err := exec.LookPath("docker")
	if err != nil {
		return errors.New("sandbox container is set but the docker command was not found")
	}
	args := []string{docker, "exec", "-i"}
	if cmd.Dir != "" {
		args = append(args, "-w", cmd.Dir)
	}
	args = append(args, name, cmd.Path)
	cmd.Args = append(args, cmd.Args[1:]...)
	cmd.Path = docker
	env := []string{}
	for _, kv := range cmd.Env {
		if k, _, _ := strings.Cut(kv, "="); k == "PATH" || k == "HOME" {
			env = append(env, kv)
		}
	}
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "DOCKER_") {
			env = append(env, kv)
		}
	}
	cmd.Env = env
	return nil
}
