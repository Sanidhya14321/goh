//go:build darwin || linux

package platform

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// NoFollow and NonBlock are OR-ed into os.OpenFile flags.
const (
	NoFollow = syscall.O_NOFOLLOW
	NonBlock = syscall.O_NONBLOCK
)

func openNoFollow(path string, flags int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(path, flags|syscall.O_NOFOLLOW, perm)
}

// TryLock takes an exclusive, nonblocking advisory lock held until f closes.
func TryLock(f *os.File) error {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return ErrLocked
		}
		return err
	}
	return nil
}

// NewGroup makes cmd lead a new process group so its descendants can be
// signalled together.
func NewGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// TerminateGroup asks the group led by pid to exit. A group that already
// exited is not an error.
func TerminateGroup(pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

// KillGroup forcibly stops the group. A group that already exited is not an error.
func KillGroup(pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

// GroupExited reports whether no process remains in the group.
func GroupExited(pid int) bool { return errors.Is(syscall.Kill(-pid, 0), syscall.ESRCH) }

// StopGroup terminates, waits up to grace, then kills. It returns an error
// only if the forced kill could not be delivered.
func StopGroup(pid int, grace time.Duration) error {
	TerminateGroup(pid)
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if GroupExited(pid) {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return KillGroup(pid)
}
