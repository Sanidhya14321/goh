//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// testShell runs the user's test command through cmd.exe, verbatim.
func testShell(command string) *exec.Cmd {
	comspec := os.Getenv("ComSpec")
	if comspec == "" || !filepath.IsAbs(comspec) {
		comspec = filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	}
	cmd := exec.Command(comspec)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `/d /s /c "` + command + `"`}
	return cmd
}
