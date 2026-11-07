//go:build windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// Windows has no O_NOFOLLOW or O_NONBLOCK; OpenNoFollow checks reparse points.
const (
	NoFollow = 0
	NonBlock = 0
)

func openNoFollow(path string, flags int, perm os.FileMode) (*os.File, error) {
	// Refuse a final reparse point (symlink or junction). This check-then-open
	// is weaker than O_NOFOLLOW against a concurrent swap; callers also confine
	// paths with os.Root.
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || info.Mode()&os.ModeIrregular != 0 {
			return nil, &os.PathError{Op: "open", Path: path, Err: errors.New("refusing to follow a link or reparse point")}
		}
	}
	return os.OpenFile(path, flags, perm)
}

// TryLock takes an exclusive, nonblocking lock on the whole file.
func TryLock(f *os.File) error {
	// Windows locks are mandatory, so lock one byte far beyond any real data:
	// writers still exclude each other, while readers such as session previews
	// can read the file contents.
	ol := windows.Overlapped{Offset: 0xFFFFFFFE, OffsetHigh: 0x7FFFFFFF}
	err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &ol)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return ErrLocked
	}
	return err
}

// NewGroup starts cmd in a new process group so taskkill /T can reach its tree.
func NewGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NEW_PROCESS_GROUP
}

func taskkill(pid int, force bool) error {
	args := []string{"/PID", strconv.Itoa(pid), "/T"}
	if force {
		args = append(args, "/F")
	}
	out, err := exec.Command("taskkill", args...).CombinedOutput()
	if err != nil && !GroupExited(pid) {
		return fmt.Errorf("taskkill: %v: %s", err, out)
	}
	return nil
}

// TerminateGroup stops the process tree rooted at pid. Windows has no
// graceful stop signal for console process trees: taskkill without /F fails
// for them ("can only be terminated forcefully"), so this kills forcefully.
// Native CI found the non-forced form reporting false cleanup failures.
func TerminateGroup(pid int) error { return taskkill(pid, true) }

// KillGroup forcibly stops the tree rooted at pid.
func KillGroup(pid int) error { return taskkill(pid, true) }

// GroupExited reports whether the root process is gone. Windows has no
// process-group probe, so descendants are covered by taskkill /T instead.
func GroupExited(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return true
	}
	defer windows.CloseHandle(h)
	event, err := windows.WaitForSingleObject(h, 0)
	return err == nil && event == windows.WAIT_OBJECT_0
}

// StopGroup terminates, waits up to grace, then kills the tree.
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

// SyncDir is a no-op: Windows cannot flush a directory handle, and NTFS
// journals directory metadata itself.
func SyncDir(*os.File) error { return nil }

// On Windows, Unix permission bits do not describe access. Files created under
// the user profile inherit owner-only ACLs by default; GoHarness relies on
// that and does not inspect ACLs. This is weaker than the Unix mode check.
func PrivateFile(info os.FileInfo, _ os.FileMode) bool { return info.Mode().IsRegular() }
func PrivateDir(info os.FileInfo) bool                 { return info.IsDir() }
func PrivateNoGroup(info os.FileInfo) bool             { return true }
