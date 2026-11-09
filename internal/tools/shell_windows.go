//go:build windows

package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// shellCommand runs command through cmd.exe. CmdLine is set verbatim so the
// model's quoting reaches cmd.exe unchanged instead of being re-escaped.
func shellCommand(command string) *exec.Cmd {
	comspec := os.Getenv("ComSpec")
	if comspec == "" || !filepath.IsAbs(comspec) {
		comspec = filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	}
	cmd := exec.Command(comspec)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `/d /s /c "` + command + `"`}
	return cmd
}
