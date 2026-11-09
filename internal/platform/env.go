package platform

import (
	"os"
	"strings"
	"sync"
)

// Child processes started by GoHarness on the model's behalf (shell commands,
// background jobs, formatters, MCP and LSP servers) do not inherit the whole
// environment: API keys, tokens and cloud credentials in the user's shell
// would otherwise be readable by any approved or injected command. Only the
// names below, plus names the user lists in env_passthrough, are passed on.
var safeEnvNames = []string{
	// POSIX basics, terminal and locale.
	"PATH", "HOME", "USER", "LOGNAME", "SHELL", "TERM", "COLORTERM", "TERM_PROGRAM",
	"LANG", "LANGUAGE", "TZ", "TMPDIR", "PWD", "EDITOR", "VISUAL", "PAGER",
	"DISPLAY", "WAYLAND_DISPLAY", "SSH_AUTH_SOCK", "CI", "NO_COLOR",
	// Toolchain locations (paths and switches, not credentials).
	"GOPATH", "GOROOT", "GOCACHE", "GOMODCACHE", "GOFLAGS", "GOPRIVATE", "GOTOOLCHAIN", "CGO_ENABLED",
	"CARGO_HOME", "RUSTUP_HOME", "JAVA_HOME", "NVM_DIR", "NODE_PATH", "VIRTUAL_ENV",
	"CONDA_PREFIX", "PYENV_ROOT", "PYTHONPATH", "HOMEBREW_PREFIX", "ANDROID_HOME", "DOTNET_ROOT",
	// Windows system variables needed to run anything.
	"SystemRoot", "SystemDrive", "ComSpec", "PATHEXT", "WINDIR", "USERPROFILE", "APPDATA",
	"LOCALAPPDATA", "TEMP", "TMP", "HOMEDRIVE", "HOMEPATH", "ProgramFiles", "ProgramFiles(x86)",
	"ProgramData", "CommonProgramFiles", "NUMBER_OF_PROCESSORS", "OS", "PROCESSOR_ARCHITECTURE",
	"USERNAME", "COMPUTERNAME", "PSModulePath",
}

// safeEnvPrefixes cover locale and XDG directory variables.
var safeEnvPrefixes = []string{"LC_", "XDG_"}

var (
	envMu       sync.RWMutex
	passthrough []string
)

// SetEnvPassthrough sets the extra variable names (from user configuration)
// that child processes may receive. "NAME_*" passes every name with that
// prefix; a lone "*" passes the whole environment (an explicit opt-out).
func SetEnvPassthrough(names []string) {
	envMu.Lock()
	passthrough = append([]string(nil), names...)
	envMu.Unlock()
}

// ChildEnv returns the filtered environment for a child process.
func ChildEnv() []string { return filterEnv(os.Environ()) }

// ChildEnvMap is ChildEnv as a map, for callers that overlay explicit values.
func ChildEnvMap() map[string]string {
	m := map[string]string{}
	for _, kv := range ChildEnv() {
		if k, v, ok := strings.Cut(kv, "="); ok {
			m[k] = v
		}
	}
	return m
}

func filterEnv(environ []string) []string {
	envMu.RLock()
	extra := passthrough
	envMu.RUnlock()
	out := make([]string, 0, len(safeEnvNames))
	for _, kv := range environ {
		name, _, ok := strings.Cut(kv, "=")
		if ok && name != "" && envAllowed(name, extra) {
			out = append(out, kv)
		}
	}
	return out
}

func envAllowed(name string, extra []string) bool {
	// Names compare case-insensitively: Windows treats them that way, and on
	// Unix no safe name differs from a secret only by case.
	for _, s := range safeEnvNames {
		if strings.EqualFold(name, s) {
			return true
		}
	}
	upper := strings.ToUpper(name)
	for _, p := range safeEnvPrefixes {
		if strings.HasPrefix(upper, p) {
			return true
		}
	}
	for _, e := range extra {
		switch {
		case e == "*":
			return true
		case strings.HasSuffix(e, "*") && len(e) > 1:
			if strings.HasPrefix(upper, strings.ToUpper(strings.TrimSuffix(e, "*"))) {
				return true
			}
		case strings.EqualFold(name, e):
			return true
		}
	}
	return false
}
