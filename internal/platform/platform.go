// Package platform isolates operating-system differences: no-follow opens,
// advisory file locks and process-group lifecycle. Callers keep one code path.
package platform

import (
	"errors"
	"os"
)

// ErrLocked reports that another process holds a lock.
var ErrLocked = errors.New("file is locked by another process")

// OpenNoFollow opens path refusing a final symbolic link (or, on Windows, any
// reparse point). flags are ordinary os.O_* flags.
func OpenNoFollow(path string, flags int, perm os.FileMode) (*os.File, error) {
	return openNoFollow(path, flags, perm)
}
