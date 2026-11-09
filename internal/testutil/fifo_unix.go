//go:build darwin || linux

package testutil

import (
	"syscall"
	"testing"
)

// Mkfifo creates a named pipe for special-file refusal tests.
func Mkfifo(t testing.TB, path string) error {
	t.Helper()
	return syscall.Mkfifo(path, 0600)
}
