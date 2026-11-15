// Package platform isolates operating-system differences: open flags,
// advisory file locks and process-group lifecycle. Callers keep one code path.
package platform

import "errors"

// ErrLocked reports that another process holds a lock.
var ErrLocked = errors.New("file is locked by another process")
