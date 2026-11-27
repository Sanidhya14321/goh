//go:build !windows

package main

import "os/exec"

// testShell runs the user's test command through the platform shell.
func testShell(command string) *exec.Cmd { return exec.Command("/bin/sh", "-c", command) }
