package tools

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

// walk visits what fs.WalkDir visits, in the same order, and applies nested
// ignore files before their directory's entries.
func TestIgnoreWalkMatchesWalkDir(t *testing.T) {
	root := t.TempDir()
	write := func(p, s string) {
		os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755)
		os.WriteFile(filepath.Join(root, p), []byte(s), 0o644)
	}
	for _, p := range []string{"a.go", "a-b.go", "a/x.go", "a/b/c/deep.go", ".github/w.yml", "z/z.go", "skip/s.go"} {
		write(p, "x\n")
	}
	write(".gitignore", "*.log\n")
	write("a/.ignore", "b/c/\n")
	write("a/b/kept.log", "x\n")
	write("z/.gitignore", "z.go\n")
	os.Symlink(filepath.Join(root, "a"), filepath.Join(root, "link"))
	if runtime.GOOS != "windows" {
		os.MkdirAll(filepath.Join(root, "locked"), 0o755)
		os.Chmod(filepath.Join(root, "locked"), 0)
		t.Cleanup(func() { os.Chmod(filepath.Join(root, "locked"), 0o755) })
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	type visit struct {
		p   string
		err bool
	}
	record := func(out *[]visit) fs.WalkDirFunc {
		return func(p string, d fs.DirEntry, err error) error {
			*out = append(*out, visit{p, err != nil})
			if p == "skip" {
				return fs.SkipDir
			}
			return nil
		}
	}
	var want, got []visit
	fs.WalkDir(r.FS(), ".", record(&want))
	newIgnoreFilter(r, ".").walk(r, ".", record(&got))
	if !slices.Equal(got, want) {
		t.Fatalf("walk visited\n%v\nfs.WalkDir visited\n%v", got, want)
	}

	var kept []string
	f := newIgnoreFilter(r, ".")
	f.walk(r, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || f.skip(p, d) {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			kept = append(kept, p)
		}
		return nil
	})
	for _, p := range []string{"a/b/c/deep.go", "a/b/kept.log", "z/z.go"} {
		if slices.Contains(kept, p) {
			t.Fatalf("%s is ignored but was kept: %v", p, kept)
		}
	}
	for _, p := range []string{".github/w.yml", "a-b.go", "a/x.go"} {
		if !slices.Contains(kept, p) {
			t.Fatalf("%s was dropped: %v", p, kept)
		}
	}
}
