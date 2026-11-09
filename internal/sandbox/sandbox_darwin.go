//go:build darwin

package sandbox

import (
	"os/exec"
	"strings"
)

const sandboxExec = "/usr/bin/sandbox-exec"

// HelperMain is unused on macOS; sandbox-exec applies the profile.
func HelperMain() {}

func wrap(cmd *exec.Cmd, p Policy) error {
	if _, err := exec.LookPath(sandboxExec); err != nil {
		return ErrUnsupported
	}
	args := append([]string{sandboxExec, "-p", profile(p), cmd.Path}, cmd.Args[1:]...)
	cmd.Path, cmd.Args = sandboxExec, args
	return nil
}

// profile builds a Seatbelt policy: everything allowed except writes outside
// the writable paths, reads of protected paths and, unless allowed, network
// access other than local Unix sockets.
func profile(p Policy) string {
	var b strings.Builder
	b.WriteString("(version 1)\n(allow default)\n(deny file-write*)\n(allow file-write*")
	for _, d := range p.Writable {
		b.WriteString(" (subpath " + quote(d) + ")")
	}
	b.WriteString(` (literal "/dev/null") (literal "/dev/tty") (literal "/dev/dtracehelper") (regex #"^/dev/fd/") (regex #"^/dev/ttys"))` + "\n")
	if len(p.Protected) > 0 {
		b.WriteString("(deny file-read* file-write*")
		for _, d := range p.Protected {
			b.WriteString(" (subpath " + quote(d) + ")")
		}
		b.WriteString(")\n")
	}
	if !p.Network {
		b.WriteString("(deny network*)\n(allow network* (remote unix-socket))\n")
	}
	return b.String()
}

// quote renders a Seatbelt string literal.
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
