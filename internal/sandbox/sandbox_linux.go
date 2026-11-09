//go:build linux

package sandbox

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func wrap(cmd *exec.Cmd, p Policy) error {
	abi, err := landlockABI()
	if err != nil || abi < 1 {
		return fmt.Errorf("%w (Landlock is not available in this kernel)", ErrUnsupported)
	}
	if !p.Network && abi < 4 {
		return errors.New(`this kernel's Landlock cannot block the network (needs ABI 4, Linux 6.7+); set "sandbox": {"network": true} to sandbox files only`)
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	args := append([]string{self, helperArg, encodePolicy(p), "--", cmd.Path}, cmd.Args[1:]...)
	cmd.Path, cmd.Args = self, args
	return nil
}

// HelperMain runs the sandbox helper when the process was started as one; it
// never returns in that case. Call it first thing in main (and TestMain).
func HelperMain() {
	if len(os.Args) < 5 || os.Args[1] != helperArg || os.Args[3] != "--" {
		return
	}
	p, err := decodePolicy(os.Args[2])
	if err == nil {
		err = restrict(p)
	}
	if err == nil {
		err = syscall.Exec(os.Args[4], os.Args[4:], os.Environ())
	}
	fmt.Fprintln(os.Stderr, "goh sandbox:", err)
	os.Exit(126)
}

func landlockABI() (int, error) {
	v, _, e := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0, unix.LANDLOCK_CREATE_RULESET_VERSION)
	if e != 0 {
		return 0, e
	}
	return int(v), nil
}

const (
	readRights = unix.LANDLOCK_ACCESS_FS_READ_FILE | unix.LANDLOCK_ACCESS_FS_READ_DIR
	writeV1    = unix.LANDLOCK_ACCESS_FS_WRITE_FILE | unix.LANDLOCK_ACCESS_FS_REMOVE_DIR | unix.LANDLOCK_ACCESS_FS_REMOVE_FILE |
		unix.LANDLOCK_ACCESS_FS_MAKE_CHAR | unix.LANDLOCK_ACCESS_FS_MAKE_DIR | unix.LANDLOCK_ACCESS_FS_MAKE_REG |
		unix.LANDLOCK_ACCESS_FS_MAKE_SOCK | unix.LANDLOCK_ACCESS_FS_MAKE_FIFO | unix.LANDLOCK_ACCESS_FS_MAKE_BLOCK |
		unix.LANDLOCK_ACCESS_FS_MAKE_SYM
)

// restrict confines this process (and what it executes) with Landlock.
// Landlock only grants, so reads are granted everywhere except inside the
// protected folders: along the path to each one, every sibling is granted.
func restrict(p Policy) error {
	abi, err := landlockABI()
	if err != nil {
		return err
	}
	write := uint64(writeV1)
	if abi >= 2 {
		write |= unix.LANDLOCK_ACCESS_FS_REFER
	}
	if abi >= 3 {
		write |= unix.LANDLOCK_ACCESS_FS_TRUNCATE
	}
	handled := write
	if len(p.Protected) > 0 {
		handled |= readRights
	}
	attr := unix.LandlockRulesetAttr{Handled_access_fs: handled}
	if !p.Network {
		attr.Handled_access_net = unix.LANDLOCK_ACCESS_NET_BIND_TCP | unix.LANDLOCK_ACCESS_NET_CONNECT_TCP
	}
	fd, _, e := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr), 0)
	if e != 0 {
		return fmt.Errorf("landlock ruleset: %w", e)
	}
	ruleset := int(fd)
	defer unix.Close(ruleset)
	add := func(path string, access uint64) error {
		f, err := unix.Open(path, unix.O_PATH|unix.O_CLOEXEC, 0)
		if err != nil {
			return nil // a missing path needs no rule
		}
		defer unix.Close(f)
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			// File rules accept only file rights.
			access &= unix.LANDLOCK_ACCESS_FS_READ_FILE | unix.LANDLOCK_ACCESS_FS_WRITE_FILE | unix.LANDLOCK_ACCESS_FS_TRUNCATE
		}
		rule := unix.LandlockPathBeneathAttr{Allowed_access: access & handled, Parent_fd: int32(f)}
		if rule.Allowed_access == 0 {
			return nil
		}
		_, _, e := unix.Syscall6(unix.SYS_LANDLOCK_ADD_RULE, uintptr(ruleset), unix.LANDLOCK_RULE_PATH_BENEATH, uintptr(unsafe.Pointer(&rule)), 0, 0, 0)
		if e != 0 {
			return fmt.Errorf("landlock rule %s: %w", path, e)
		}
		return nil
	}
	for _, d := range append(append([]string{}, p.Writable...), "/dev/null", "/dev/tty", "/dev/pts", "/proc/self/fd") {
		if err := add(d, write|readRights); err != nil {
			return err
		}
	}
	if len(p.Protected) > 0 {
		for _, d := range readableAround(p.Protected) {
			if err := add(d, readRights); err != nil {
				return err
			}
		}
	}
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return err
	}
	if _, _, e := unix.Syscall(unix.SYS_LANDLOCK_RESTRICT_SELF, uintptr(ruleset), 0, 0); e != 0 {
		return fmt.Errorf("landlock restrict: %w", e)
	}
	return nil
}

// readableAround returns paths to grant read access to so that everything
// except the protected folders stays readable: every sibling of each folder
// on the way from / to a protected folder.
func readableAround(protected []string) []string {
	blocked := map[string]bool{} // protected folders and their ancestors
	for _, p := range protected {
		for d := filepath.Clean(p); ; d = filepath.Dir(d) {
			blocked[d] = true
			if d == "/" {
				break
			}
		}
	}
	var grant []string
	for dir := range blocked {
		if isProtected(dir, protected) {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			child := filepath.Join(dir, e.Name())
			if !blocked[child] {
				grant = append(grant, child)
			}
		}
	}
	sort.Strings(grant)
	return grant
}

func isProtected(dir string, protected []string) bool {
	for _, p := range protected {
		if dir == p || strings.HasPrefix(dir, p+"/") {
			return true
		}
	}
	return false
}
