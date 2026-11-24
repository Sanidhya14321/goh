package tools

import (
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/Sanidhya14321/goh/internal/ignore"
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

// skip reports whether p should be left out. A directory that is kept has
// its own ignore files loaded by walk once its listing is read.
func (f *ignoreFilter) skip(p string, d fs.DirEntry) bool {
	p = path.Clean(p)
	return p != f.start && p != "." && f.matcher.Ignored(p, d.IsDir())
}

// walk is fs.WalkDir over root that loads each directory's ignore rules
// before its entries are visited, opening only the ignore files its listing
// contains (trying both names in every directory was most of a project
// walk's time). Each directory is opened from its already open parent, one
// system call, instead of resolving its whole path again.
func (f *ignoreFilter) walk(root *os.Root, start string, fn fs.WalkDirFunc) error {
	dir, err := root.OpenRoot(start)
	var info fs.FileInfo
	if err == nil {
		defer dir.Close()
		info, err = dir.Stat(".")
	}
	if err != nil {
		// Not a directory (or unreadable): visit it as a single entry.
		info, err = root.Lstat(start)
		if err != nil {
			err = fn(start, nil, err)
		} else {
			err = fn(start, fs.FileInfoToDirEntry(info), nil)
		}
	} else {
		err = f.walkDir(dir, start, fs.FileInfoToDirEntry(info), fn)
	}
	if err == fs.SkipDir || err == fs.SkipAll {
		return nil
	}
	return err
}

// walkDir visits directory name, already open as dir.
func (f *ignoreFilter) walkDir(dir *os.Root, name string, d fs.DirEntry, fn fs.WalkDirFunc) error {
	if err := fn(name, d, nil); err != nil {
		if err == fs.SkipDir {
			err = nil // the directory is skipped, its siblings are not
		}
		return err
	}
	entries, err := readDirSorted(dir)
	if err != nil {
		if err = fn(name, d, err); err != nil {
			if err == fs.SkipDir {
				err = nil
			}
			return err
		}
	}
	for _, e := range entries {
		if n := e.Name(); n == ".gitignore" || n == ".ignore" {
			f.load(path.Clean(name))
			break // load reads both names
		}
	}
	for _, e := range entries {
		child := path.Join(name, e.Name())
		if !e.IsDir() {
			if err := fn(child, e, nil); err != nil {
				if err == fs.SkipDir {
					break
				}
				return err
			}
			continue
		}
		sub, openErr := dir.OpenRoot(e.Name())
		if openErr != nil {
			// As fs.WalkDir: the directory is visited, then its error.
			err := fn(child, e, nil)
			if err == nil {
				err = fn(child, e, openErr)
			}
			if err != nil && err != fs.SkipDir {
				return err
			}
			continue
		}
		err = f.walkDir(sub, child, e, fn)
		sub.Close()
		if err != nil {
			if err == fs.SkipDir {
				break
			}
			return err
		}
	}
	return nil
}

func readDirSorted(dir *os.Root) ([]fs.DirEntry, error) {
	file, err := dir.Open(".")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	entries, err := file.ReadDir(-1)
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, err
}
