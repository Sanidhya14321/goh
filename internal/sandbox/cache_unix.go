//go:build !windows

package sandbox

import (
	"os"
	"syscall"
)

// privateDir reports whether dir is a real directory (not a link) that this
// user owns and nobody else can write or read.
func privateDir(dir string) bool {
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) == os.Getuid()
}
