package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// cacheVars are build and package caches that tools keep outside the
// project and temporary folders (Go's build cache under ~/Library/Caches or
// ~/.cache, npm's and pip's caches). A sandboxed command cannot write them,
// so `go test` failed at its first cache write and the model had to retry
// with its own temporary cache.
var cacheVars = []struct{ name, dir string }{
	{"GOCACHE", "go-build"},
	{"npm_config_cache", "npm"},
	{"PIP_CACHE_DIR", "pip"},
	{"XDG_CACHE_HOME", "xdg"},
}

// withSandboxCaches points every cache variable the environment leaves
// unset at a folder inside the writable temporary area. The user's own
// caches stay untouched (a sandboxed command cannot poison them), and later
// sandboxed commands reuse the same folder, so builds stay incremental. The
// folder is per user and must be a private directory this user owns;
// otherwise the environment is returned unchanged.
func withSandboxCaches(env []string) []string {
	if env == nil {
		env = os.Environ()
	}
	base := filepath.Join(os.TempDir(), fmt.Sprintf("goh-sandbox-cache-%d", os.Getuid()))
	if err := os.MkdirAll(base, 0o700); err != nil || !privateDir(base) {
		return env
	}
	set := map[string]bool{}
	for _, kv := range env {
		if k, _, ok := strings.Cut(kv, "="); ok {
			set[k] = true
		}
	}
	out := append([]string(nil), env...)
	for _, v := range cacheVars {
		if !set[v.name] {
			out = append(out, v.name+"="+filepath.Join(base, v.dir))
		}
	}
	return out
}
