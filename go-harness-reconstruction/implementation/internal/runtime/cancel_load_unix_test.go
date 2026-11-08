//go:build !windows

package runtime

import (
	"errors"
	"syscall"
)

const sleepWritingPID = "echo $$ > child.pid; exec sleep 30"

func processAlive(pid int) bool { return !errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) }

func killProcess(pid int) { syscall.Kill(pid, syscall.SIGKILL) }
