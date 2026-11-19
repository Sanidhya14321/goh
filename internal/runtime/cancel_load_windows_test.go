//go:build windows

package runtime

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// PowerShell writes its own PID, then sleeps; cmd.exe has no $$.
const sleepWritingPID = `powershell -NoProfile -Command "$PID | Out-File -Encoding ascii child.pid; Start-Sleep 30"`

func processAlive(pid int) bool {
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH").Output()
	return err == nil && strings.Contains(string(out), " "+strconv.Itoa(pid)+" ")
}

func killProcess(pid int) { exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run() }

// childStartWait bounds waiting for the child shell to start, before the
// measured cancellation: six PowerShell starts at once on a loaded runner
// have exceeded 30 s (CI runs 44125517188, 44553177409); start-up speed is
// not what the test measures.
const childStartWait = 120 * time.Second
