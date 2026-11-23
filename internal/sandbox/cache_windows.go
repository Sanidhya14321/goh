//go:build windows

package sandbox

// privateDir: Windows has no native command sandbox, so caches are never
// redirected there.
func privateDir(string) bool { return false }
