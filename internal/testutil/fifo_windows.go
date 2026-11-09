//go:build windows

package testutil

import "testing"

// Mkfifo skips the calling test: Windows has no filesystem named pipes, so
// the special-file case those tests exercise cannot occur.
func Mkfifo(t testing.TB, path string) error {
	t.Helper()
	t.Skip("named pipes in the filesystem are Unix-only")
	return nil
}
