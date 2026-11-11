package tools

import (
	"os"
	"path/filepath"
	"strings"
)

// CanonicalRel returns the project-relative spelling of rel as it exists on
// disk: symlinked folders are resolved and, on case-insensitive filesystems,
// each name takes its stored case. Path rules compare spellings, so a rule
// on "secret.txt" must also see "SECRET.TXT" and "link/secret.txt" as that
// file. It returns rel unchanged (slash-separated) when the path does not
// exist or resolves outside the project.
func CanonicalRel(root, rel string) string {
	rel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return rel
	}
	full, err := filepath.EvalSymlinks(filepath.Join(realRoot, filepath.FromSlash(rel)))
	if err != nil {
		return rel
	}
	r, err := filepath.Rel(realRoot, full)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return rel
	}
	// Recover the stored case of each component.
	dir := realRoot
	parts := strings.Split(r, string(filepath.Separator))
	for i, name := range parts {
		if entries, err := os.ReadDir(dir); err == nil {
			exact, folded := false, ""
			for _, e := range entries {
				if e.Name() == name {
					exact = true
					break
				}
				if folded == "" && strings.EqualFold(e.Name(), name) {
					folded = e.Name()
				}
			}
			if !exact && folded != "" {
				parts[i] = folded
			}
		}
		dir = filepath.Join(dir, parts[i])
	}
	return strings.Join(parts, "/")
}
