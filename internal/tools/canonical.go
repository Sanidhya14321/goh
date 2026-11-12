package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// CanonicalRel returns the project-relative spelling of rel as it exists on
// disk: symlinked folders are resolved and, on filesystems that fold names
// (case or Unicode normalization, as APFS and NTFS do), each existing name
// takes its stored spelling. Path rules compare spellings, so a rule on
// "secret.txt" must also see "SECRET.TXT" and "link/secret.txt" as that
// file. The longest existing prefix is resolved and a missing tail (a file
// about to be created) is kept as typed. It returns rel unchanged
// (slash-separated) when nothing exists or the path resolves outside the
// project.
func CanonicalRel(root, rel string) string {
	rel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if rel == "." || filepath.IsAbs(filepath.FromSlash(rel)) {
		return rel
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return rel
	}
	parts := strings.Split(rel, "/")
	n, resolved := len(parts), ""
	for ; n > 0; n-- {
		full, err := filepath.EvalSymlinks(filepath.Join(realRoot, filepath.FromSlash(strings.Join(parts[:n], "/"))))
		if err == nil {
			resolved = full
			break
		}
	}
	if n == 0 {
		return rel
	}
	r, err := filepath.Rel(realRoot, resolved)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return rel
	}
	var out []string
	if r != "." {
		out = storedNames(realRoot, strings.Split(r, string(filepath.Separator)))
	}
	for _, tail := range parts[n:] {
		if tail == ".." {
			return rel // a missing folder followed by "..": keep it as typed
		}
		out = append(out, tail)
	}
	if len(out) == 0 {
		return "."
	}
	return strings.Join(out, "/")
}

// storedNames replaces each existing component with the name the directory
// stores for it, when the typed name is an alias of it.
func storedNames(root string, parts []string) []string {
	dir := root
	for i, name := range parts {
		if entries, err := os.ReadDir(dir); err == nil {
			typed, typedErr := os.Lstat(filepath.Join(dir, name))
			stored := ""
			for _, e := range entries {
				if e.Name() == name {
					stored = ""
					break
				}
				if stored == "" && typedErr == nil && FoldName(e.Name()) == FoldName(name) {
					if info, err := os.Lstat(filepath.Join(dir, e.Name())); err == nil && os.SameFile(info, typed) {
						stored = e.Name()
					}
				}
			}
			if stored != "" {
				parts[i] = stored
			}
		}
		dir = filepath.Join(dir, parts[i])
	}
	return parts
}

// FoldName is the comparison key for name-folding filesystems: Unicode NFC
// with case folded.
func FoldName(s string) string {
	return strings.ToLower(norm.NFC.String(s))
}

var (
	foldCache    sync.Map // real root -> bool
	defaultFolds = runtime.GOOS == "darwin" || runtime.GOOS == "windows"
)

// FoldsNames reports whether the filesystem holding root treats names that
// differ in case (and, with it, Unicode normalization) as one file. It
// probes the root folder's own name; when that has no letters to swap it
// assumes the platform default (folding on macOS and Windows), the safe
// choice for deny rules.
func FoldsNames(root string) bool {
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		real = root
	}
	if v, ok := foldCache.Load(real); ok {
		return v.(bool)
	}
	folds := defaultFolds
	base := filepath.Base(real)
	swapped := strings.Map(func(r rune) rune {
		if unicode.IsUpper(r) {
			return unicode.ToLower(r)
		}
		return unicode.ToUpper(r)
	}, base)
	if swapped != base {
		folds = false
		if a, err := os.Stat(real); err == nil {
			if b, err := os.Stat(filepath.Join(filepath.Dir(real), swapped)); err == nil && os.SameFile(a, b) {
				folds = true
			}
		}
	}
	foldCache.Store(real, folds)
	return folds
}

// DeniedPath reports whether deny refuses rel (project-relative) in any of
// its spellings: as typed and as stored on disk, compared folded where the
// filesystem folds names.
func DeniedPath(root, rel string, deny func(spelling string, fold bool) bool) bool {
	if deny == nil || rel == "" || rel == "." {
		return false
	}
	fold := FoldsNames(root)
	typed := filepath.ToSlash(rel)
	if deny(typed, fold) {
		return true
	}
	canonical := CanonicalRel(root, rel)
	return canonical != typed && deny(canonical, fold)
}
