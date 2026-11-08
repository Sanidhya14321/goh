package tools

import (
	"io/fs"
	"os"
	"path"
	"strings"

	"example.com/independent-harness/internal/ignore"
)

// ignoreFilter applies .gitignore and .ignore rules during a walk, as
// ripgrep-based tools do. Rules from the project root down to the walk's
// start are loaded first; each directory's files are added on entry. The
// start path itself is never filtered, so an explicitly requested ignored
// directory can still be searched.
type ignoreFilter struct {
	root    *os.Root
	start   string
	matcher *ignore.Matcher
}

func newIgnoreFilter(root *os.Root, start string) *ignoreFilter {
	f := &ignoreFilter{root: root, start: path.Clean(start), matcher: &ignore.Matcher{}}
	dir := "."
	if f.start != "." {
		f.load(dir) // ancestors of the start; the start loads on entry
		for _, part := range strings.Split(f.start, "/") {
			dir = path.Join(dir, part)
			if dir == f.start {
				break
			}
			f.load(dir)
		}
	}
	return f
}

func (f *ignoreFilter) load(dir string) {
	for _, name := range []string{".gitignore", ".ignore"} {
		file, err := f.root.Open(path.Join(dir, name))
		if err != nil {
			continue
		}
		if info, err := file.Stat(); err == nil && info.Mode().IsRegular() && info.Size() <= 1<<20 {
			base := dir
			if base == "." {
				base = ""
			}
			f.matcher.Add(base, file)
		}
		file.Close()
	}
}

// skip reports whether p should be left out; directories that are kept
// have their own ignore files loaded.
func (f *ignoreFilter) skip(p string, d fs.DirEntry) bool {
	p = path.Clean(p)
	if p != f.start && p != "." && f.matcher.Ignored(p, d.IsDir()) {
		return true
	}
	if d.IsDir() {
		f.load(p)
	}
	return false
}
