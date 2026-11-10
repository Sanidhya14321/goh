// Package sandbox confines shell commands the model runs: writes only inside
// the project, temporary folders and paths the user lists; optionally no
// network; and no reading GoHarness's own credentials or sessions.
//
// macOS uses the built-in sandbox-exec. Linux uses Landlock, applied by
// re-running the goh binary as a small helper that restricts itself and then
// executes the command. Other systems are refused rather than run unconfined.
package sandbox

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Policy describes what a sandboxed command may do.
type Policy struct {
	// Writable are directories the command may write beneath (the project,
	// temporary folders and user-listed paths).
	Writable []string `json:"writable"`
	// Protected are directories whose contents the command may not read.
	Protected []string `json:"protected"`
	// Network allows network access when true.
	Network bool `json:"network"`
	// DenyFiles are individual files (for example ones the user's
	// file_rules deny) that may not be read or written even inside a
	// writable folder. Enforced on macOS; Landlock can only add access, so
	// Linux cannot hide a file inside a writable folder.
	DenyFiles []string `json:"deny_files,omitempty"`
	// Container, when set, runs the command in this running Docker
	// container instead (see core.SandboxConfig.Container).
	Container string `json:"container,omitempty"`
}

// helperArg is the hidden first argument that runs the Linux helper.
const helperArg = "__goh_sandbox"

// ErrUnsupported reports that this system has no supported sandbox.
var ErrUnsupported = errors.New("the shell sandbox is not supported on this system; turn \"sandbox\" off to run commands unconfined")

// Wrap rewrites cmd so it runs inside the sandbox described by p. A nil
// policy leaves cmd unchanged.
func Wrap(cmd *exec.Cmd, p *Policy) error {
	if p == nil {
		return nil
	}
	if p.Container != "" {
		return wrapContainer(cmd, p.Container)
	}
	clean := Policy{Network: p.Network, DenyFiles: p.DenyFiles}
	for _, d := range p.Writable {
		if r := resolve(d); r != "" {
			clean.Writable = append(clean.Writable, r)
		}
	}
	for _, d := range p.Protected {
		if r := resolve(d); r != "" {
			clean.Protected = append(clean.Protected, r)
		}
	}
	return wrap(cmd, clean)
}

// resolve makes dir absolute with symlinks resolved; missing paths resolve
// through their longest existing prefix so the rule still matches later.
func resolve(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	rest := ""
	for p := abs; ; p = filepath.Dir(p) {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Join(r, rest)
		}
		if filepath.Dir(p) == p {
			return abs
		}
		rest = filepath.Join(filepath.Base(p), rest)
	}
}

// TempDirs are the temporary folders a sandboxed command may write.
func TempDirs() []string {
	dirs := []string{os.TempDir(), "/tmp"}
	if v := os.Getenv("TMPDIR"); v != "" {
		dirs = append(dirs, v)
	}
	// macOS keeps this user's temporary (T) and cache (C) folders side by
	// side under /var/folders/xx/yyyy/; allow that user folder only.
	if t := filepath.Clean(os.TempDir()); filepath.Base(t) == "T" && strings.Contains(t, "/var/folders/") {
		dirs = append(dirs, filepath.Dir(t))
	}
	return dirs
}

func encodePolicy(p Policy) string {
	b, _ := json.Marshal(p)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodePolicy(s string) (Policy, error) {
	var p Policy
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err == nil {
		err = json.Unmarshal(b, &p)
	}
	return p, err
}
