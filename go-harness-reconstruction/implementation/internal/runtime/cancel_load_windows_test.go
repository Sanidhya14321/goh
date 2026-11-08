//go:build windows

package runtime

import (
	"os/exec"
	"strconv"
	"strings"
)

// PowerShell writes its own PID, then sleeps; cmd.exe has no $$.
const sleepWritingPID = `powershell -NoProfile -Command "$PID | Out-File -Encoding ascii child.pid; Start-Sleep 30"`

func processAlive(pid int) bool {
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH").Output()
	return err == nil && strings.Contains(string(out), " "+strconv.Itoa(pid)+" ")
}

func killProcess(pid int) { exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run() }
