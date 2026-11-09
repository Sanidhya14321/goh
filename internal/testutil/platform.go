package testutil

import (
	"path/filepath"
	"runtime"
	"testing"
)

// UnixModes reports whether file permission bits are meaningful. On Windows,
// access is governed by ACLs and Go reports only 0666/0444.
func UnixModes() bool { return runtime.GOOS != "windows" }

// RequireUnixShell skips tests whose fixtures are /bin/sh scripts or POSIX
// shell syntax. Windows shell behavior needs its own cmd.exe coverage.
func RequireUnixShell(t testing.TB) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX shell syntax")
	}
}

// TempDir returns a test directory with symlinks and Windows 8.3 short names
// resolved, matching how GoHarness canonicalizes a project root.
func TempDir(t testing.TB) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
