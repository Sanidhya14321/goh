package tools

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// fingerprintLimit bounds the walk; larger projects report no fingerprint.
const fingerprintLimit = 20000

// cacheDirs are build and test caches that checks write as a side effect;
// they never count as changes even when no ignore file lists them.
var cacheDirs = map[string]bool{".git": true, "node_modules": true, "__pycache__": true, ".pytest_cache": true,
	".mypy_cache": true, ".ruff_cache": true, ".tox": true, ".venv": true, ".gradle": true, ".next": true, ".harness": true}

// Fingerprint summarises the project's files that are not ignored (path, size,
// mode, modification time and a change stamp), so a caller can tell whether a
// command changed them. The modification time can be set back after a
// write; the change stamp cannot (see changeStamp). ok is false when the
// project is too large or cannot be read.
// Folders in e.StateDirs (GoHarness's own session storage, when it sits
// inside the project) are skipped too.
func (e *Engine) Fingerprint() (sum string, ok bool) {
	skipRel := map[string]bool{}
	base := e.root
	if r, err := filepath.EvalSymlinks(base); err == nil {
		base = r
	}
	for _, d := range e.StateDirs {
		if r, err := filepath.EvalSymlinks(d); err == nil {
			d = r
		}
		if rel, err := filepath.Rel(base, d); err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			skipRel[filepath.ToSlash(rel)] = true
		}
	}
	root, err := os.OpenRoot(e.root)
	if err != nil {
		return "", false
	}
	defer root.Close()
	filter := newIgnoreFilter(root, ".")
	h := sha256.New()
	n := 0
	err = filter.walk(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil // removed during the walk
			}
			// An unreadable folder hides its changes, so the coverage is
			// unknown rather than "unchanged".
			return err
		}
		if p == "." {
			return nil
		}
		if d.IsDir() && (cacheDirs[d.Name()] || skipRel[p]) || filter.skip(p, d) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if n++; n > fingerprintLimit {
			return fs.SkipAll
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		var b [24]byte
		binary.LittleEndian.PutUint64(b[0:], uint64(info.Size()))
		binary.LittleEndian.PutUint64(b[8:], uint64(info.ModTime().UnixNano()))
		binary.LittleEndian.PutUint64(b[16:], uint64(info.Mode()))
		h.Write([]byte(p))
		h.Write([]byte{0})
		h.Write(b[:])
		h.Write(changeStamp(root, p, info))
		return nil
	})
	if err != nil || n > fingerprintLimit {
		return "", false
	}
	return hex.EncodeToString(h.Sum(nil)), true
}
