//go:build !darwin && !linux

package sandbox

import "os/exec"

// HelperMain is unused where no sandbox is supported.
func HelperMain() {}

func wrap(*exec.Cmd, Policy) error { return ErrUnsupported }

// Supported reports whether commands can be sandboxed here.
func Supported() bool { return false }

// NestedRestrictionsEnforced reports whether a path inside a writable folder
// can be restricted again; there is no sandbox here.
func NestedRestrictionsEnforced() bool { return false }
