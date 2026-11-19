//go:build !windows

package runtime

import (
	"errors"
	"syscall"
	"time"
)

const sleepWritingPID = "echo $$ > child.pid; exec sleep 30"

func processAlive(pid int) bool { return !errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) }

func killProcess(pid int) { syscall.Kill(pid, syscall.SIGKILL) }

// childStartWait bounds waiting for the child shell to start, before the
// measured cancellation: a shell starts in milliseconds here.
const childStartWait = 30 * time.Second
