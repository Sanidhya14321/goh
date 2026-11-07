//go:build darwin || linux

package tools

import "os/exec"

func shellCommand(command string) *exec.Cmd { return exec.Command("/bin/sh", "-c", command) }
