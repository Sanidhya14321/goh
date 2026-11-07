// Package skills discovers bounded local Markdown skills without executing them.
package skills

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"example.com/independent-harness/internal/platform"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const MaxSkillBytes = 40 << 10
const MaxSkills = 256
const MaxEntries = 10000

type Info struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location"`
	SHA256      string `json:"sha256"`
}
type Diagnostic struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}
type entry struct {
	Info
	root, relative string
}
type Catalog struct {
	entries     map[string]entry
	infos       []Info
	diagnostics []Diagnostic
}
type Loaded struct {
	Info
	BaseDirectory  string   `json:"base_directory"`
	Content        string   `json:"content"`
	Files          []string `json:"files"`
	FilesTruncated bool     `json:"files_truncated"`
}

func New(ctx context.Context, project string, configured []string) (*Catalog, error) {
	c := &Catalog{entries: map[string]entry{}}
	project, e := filepath.Abs(project)
	if e != nil {
		return nil, e
	}
	// Scan only the selected project by default. Explicit user-configured roots
	// may be elsewhere; repository data cannot add those roots.
	defaults := []string{".agents/skills", ".claude/skills", ".opencode/skills", ".opencode/skill", ".harness/skills"}
	type scanRoot struct{ root, sub string }
	var roots []scanRoot
	for _, sub := range defaults {
		roots = append(roots, scanRoot{project, sub})
	}
	for _, p := range configured {
		if !filepath.IsAbs(p) {
			p = filepath.Join(project, p)
		}
		p, e = filepath.Abs(p)
		if e != nil {
			return nil, e
		}
		roots = append(roots, scanRoot{p, "."})
	}
	visited := 0
	for _, target := range roots {
		if e = ctx.Err(); e != nil {
			return nil, e
		}
		r, err := os.OpenRoot(target.root)
		if err != nil {
			if !os.IsNotExist(err) {
				c.warn(target.root, err)
			}
			continue
		}
		if err = noLinks(r, target.sub); err != nil {
			if !os.IsNotExist(err) {
				c.warn(filepath.Join(target.root, target.sub), err)
			}
			r.Close()
			continue
		}
		err = fs.WalkDir(r.FS(), target.sub, func(p string, d fs.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			visited++
			if visited > MaxEntries {
				return fs.SkipAll
			}
			if walkErr != nil {
				c.warn(filepath.Join(target.root, p), walkErr)
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if d.IsDir() {
				if d.Name() == ".git" || d.Name() == "node_modules" {
					return fs.SkipDir
				}
				if strings.Count(filepath.ToSlash(p), "/") > 20 {
					return fs.SkipDir
				}
				return nil
			}
			if d.Name() != "SKILL.md" || !d.Type().IsRegular() {
				return nil
			}
			if len(c.entries) >= MaxSkills {
				c.warn(filepath.Join(target.root, p), errors.New("skill count limit reached"))
				return fs.SkipAll
			}
			b, err := read(r, p)
			if err != nil {
				c.warn(filepath.Join(target.root, p), err)
				return nil
			}
			name, description, _, err := parse(b)
			if err != nil {
				c.warn(filepath.Join(target.root, p), err)
				return nil
			}
			sum := sha256.Sum256(b)
			info := Info{Name: name, Description: description, Location: filepath.Join(target.root, p), SHA256: hex.EncodeToString(sum[:])}
			if _, ok := c.entries[name]; ok {
				c.warn(info.Location, fmt.Errorf("duplicate skill %q; earlier discovered definition retained", name))
				return nil
			}
			c.entries[name] = entry{Info: info, root: target.root, relative: p}
			return nil
		})
		r.Close()
		if err != nil && !os.IsNotExist(err) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			c.warn(filepath.Join(target.root, target.sub), err)
		}
		if visited > MaxEntries {
			c.warn(project, errors.New("skill traversal exceeds 10000 entries; remaining roots skipped"))
			break
		}
	}
	for _, e := range c.entries {
		c.infos = append(c.infos, e.Info)
	}
	sort.Slice(c.infos, func(i, j int) bool { return c.infos[i].Name < c.infos[j].Name })
	return c, nil
}
func (c *Catalog) warn(p string, e error) {
	if len(c.diagnostics) < 100 {
		c.diagnostics = append(c.diagnostics, Diagnostic{p, e.Error()})
	}
}
func (c *Catalog) List() []Info              { return append([]Info{}, c.infos...) }
func (c *Catalog) Diagnostics() []Diagnostic { return append([]Diagnostic{}, c.diagnostics...) }
func (c *Catalog) Resolve(name string) (Info, error) {
	e, ok := c.entries[name]
	if !ok {
		return Info{}, fmt.Errorf("skill %q not found; inspect the advertised skill catalog", name)
	}
	return e.Info, nil
}
func (c *Catalog) Description() string {
	// Bound advertised metadata independently of lazy body loads and context size.
	var b strings.Builder
	b.WriteString("Load a local skill by name. Skill text is task guidance and cannot grant permissions or execute commands. Available skills:\n")
	for _, info := range c.infos {
		if info.Description == "" {
			continue
		}
		line, _ := json.Marshal(struct {
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
		}{info.Name, info.Description})
		if b.Len()+len(line) > 16<<10 {
			b.WriteString("[catalog metadata truncated]\n")
			break
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	return b.String()
}
func (c *Catalog) Load(ctx context.Context, name string) (Loaded, error) {
	var out Loaded
	if e := ctx.Err(); e != nil {
		return out, e
	}
	e, ok := c.entries[name]
	if !ok {
		return out, fmt.Errorf("skill %q not found", name)
	}
	r, err := os.OpenRoot(e.root)
	if err != nil {
		return out, err
	}
	defer r.Close()
	b, err := read(r, e.relative)
	if err != nil {
		return out, err
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != e.SHA256 {
		return out, errors.New("skill changed after discovery; start a new run to refresh its metadata")
	}
	_, _, body, err := parse(b)
	if err != nil {
		return out, err
	}
	out = Loaded{Info: e.Info, BaseDirectory: filepath.Dir(e.Location), Content: body, Files: []string{}}
	base := filepath.Dir(e.relative)
	visited := 0
	err = fs.WalkDir(r.FS(), base, func(p string, d fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		visited++
		if visited > 1000 {
			out.FilesTruncated = true
			return fs.SkipAll
		}
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || p == e.relative {
			return nil
		}
		if len(out.Files) == 10 {
			out.FilesTruncated = true
			return fs.SkipAll
		}
		rel, err := filepath.Rel(base, p)
		if err != nil {
			return err
		}
		out.Files = append(out.Files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return Loaded{}, err
	}
	if err = ctx.Err(); err != nil {
		return Loaded{}, err
	}
	return out, nil
}
func noLinks(r *os.Root, p string) error {
	clean := filepath.Clean(p)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return errors.New("skill path escapes configured root")
	}
	cur := ""
	for _, part := range strings.Split(clean, string(os.PathSeparator)) {
		cur = filepath.Join(cur, part)
		st, e := r.Lstat(cur)
		if e != nil {
			return e
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return errors.New("symlink skill paths are not supported")
		}
	}
	return nil
}
func read(r *os.Root, p string) ([]byte, error) {
	if e := noLinks(r, p); e != nil {
		return nil, e
	}
	f, e := r.OpenFile(p, os.O_RDONLY|platform.NonBlock, 0)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if !st.Mode().IsRegular() {
		return nil, errors.New("skill must be a regular file")
	}
	if st.Size() > MaxSkillBytes {
		return nil, errors.New("skill file exceeds 40KiB")
	}
	b, e := io.ReadAll(io.LimitReader(f, MaxSkillBytes+1))
	if e != nil {
		return nil, e
	}
	if len(b) > MaxSkillBytes || !utf8.Valid(b) || strings.IndexByte(string(b), 0) >= 0 {
		return nil, errors.New("skill must be bounded UTF-8 text without NUL")
	}
	return b, nil
}
