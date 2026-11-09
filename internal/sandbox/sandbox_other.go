//go:build !darwin && !linux

package sandbox

import "os/exec"

// HelperMain is unused where no sandbox is supported.
func HelperMain() {}

func wrap(*exec.Cmd, Policy) error { return ErrUnsupported }
