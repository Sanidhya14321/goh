package tools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const globLimit = 100
const globAlternatives = 64
const globPatternBytes = 4096

type globProgram struct{ alternatives [][]string }

// compileGlob is an independently authored, bounded shell-style path matcher.
// A slash-free pattern selects basenames at any depth; ** as a complete path
// component selects zero or more directories. Matching never opens a file.
func compileGlob(pattern string) (globProgram, error) {
	var program globProgram
	if pattern == "" || len(pattern) > globPatternBytes || !utf8.ValidString(pattern) || strings.IndexByte(pattern, 0) >= 0 {
		return program, errors.New("glob pattern must be nonempty UTF-8 without NUL, at most 4096 bytes")
	}
	for strings.HasPrefix(pattern, "./") {
		pattern = strings.TrimPrefix(pattern, "./")
	}
	alternatives, err := expandBraces(pattern, 0)
	if err != nil {
		return program, err
	}
	for _, alternative := range alternatives {
		if strings.HasPrefix(alternative, "!") {
			return program, errors.New("negated glob patterns are unsupported; use a positive file pattern")
		}
		if alternative == "" || strings.HasPrefix(alternative, "/") {
			return program, errors.New("glob pattern must be relative to the search directory")
		}
		components := strings.Split(alternative, "/")
		if len(components) > 128 {
			return program, errors.New("glob pattern exceeds 128 components")
		}
		if len(components) == 1 {
			components = append([]string{"**"}, components...)
		}
		for i, component := range components {
			if component == "" || component == "." || component == ".." {
				return program, errors.New("glob pattern contains an empty or escaping path component")
			}
			component = normalizeClassNegation(component)
			if component != "**" {
				if _, err := path.Match(component, ""); err != nil {
					return program, fmt.Errorf("invalid glob pattern: %w", err)
				}
			}
			components[i] = component
		}
		program.alternatives = append(program.alternatives, components)
	}
	return program, nil
}
func normalizeClassNegation(component string) string {
	var b strings.Builder
	escaped, afterOpen := false, false
	for _, r := range component {
		if escaped {
			b.WriteRune(r)
			escaped = false
			afterOpen = false
			continue
		}
		if r == '\\' {
			b.WriteRune(r)
			escaped = true
			afterOpen = false
			continue
		}
		if afterOpen && r == '!' {
			b.WriteRune('^')
		} else {
			b.WriteRune(r)
		}
		afterOpen = r == '['
	}
	return b.String()
}
func expandBraces(pattern string, depth int) ([]string, error) {
	if depth > 8 {
		return nil, errors.New("glob brace nesting exceeds 8 levels")
	}
	start := -1
	nested := 0
	inClass := false
	escaped := false
	var separators []int
	for i, r := range pattern {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '[' {
			inClass = true
		}
		if r == ']' {
			inClass = false
		}
		if inClass {
			continue
		}
		switch r {
		case '{':
			if start < 0 {
				start = i
				nested = 1
			} else {
				nested++
			}
		case '}':
			if start < 0 {
				return nil, errors.New("unmatched closing brace in glob pattern")
			}
			nested--
			if nested != 0 {
				continue
			}
			if len(separators) == 0 {
				return nil, errors.New("glob braces require comma-separated alternatives")
			}
			ends := append(separators, i)
			begin := start + 1
			var out []string
			for _, end := range ends {
				expanded, err := expandBraces(pattern[:start]+pattern[begin:end]+pattern[i+1:], depth+1)
				if err != nil {
					return nil, err
				}
				if len(out)+len(expanded) > globAlternatives {
					return nil, errors.New("glob brace expansion exceeds 64 alternatives")
				}
				out = append(out, expanded...)
				begin = end + 1
			}
			return out, nil
		case ',':
			if start >= 0 && nested == 1 {
				separators = append(separators, i)
			}
		}
	}
	if start >= 0 {
		return nil, errors.New("unclosed brace in glob pattern")
	}
	return []string{pattern}, nil
}
func (g globProgram) matches(relative string) bool {
	parts := strings.Split(relative, "/")
	for _, pattern := range g.alternatives {
		// Dynamic programming bounds recursive ** matching by component pairs.
		type state struct{ pattern, file int }
		memo := map[state]bool{}
		known := map[state]bool{}
		var match func(int, int) bool
		match = func(i, j int) bool {
			s := state{i, j}
			if known[s] {
				return memo[s]
			}
			known[s] = true
			ok := false
			if i == len(pattern) {
				ok = j == len(parts)
			} else if pattern[i] == "**" {
				ok = match(i+1, j) || (j < len(parts) && match(i, j+1))
			} else if j < len(parts) {
				component, _ := path.Match(pattern[i], parts[j])
				ok = component && match(i+1, j+1)
			}
			memo[s] = ok
			return ok
		}
		if match(0, 0) {
			return true
		}
	}
	return false
}
func globFiles(ctx context.Context, root *os.Root, a args, deny func(string) bool) Result {
	program, err := compileGlob(a.Pattern)
	if err != nil {
		return failure(err)
	}
	files := make([]string, 0)
	visited, regular := 0, 0
	truncated := false
	err = fs.WalkDir(root.FS(), filepath.ToSlash(a.Path), func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		visited++
		if visited > maxFiles*2 {
			truncated = true
			return fs.SkipAll
		}
		if deny(p) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if p != a.Path && (d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == ".harness") {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		regular++
		if regular > maxFiles {
			truncated = true
			return fs.SkipAll
		}
		relative := p
		if a.Path != "." {
			relative = strings.TrimPrefix(p, a.Path+"/")
		}
		if program.matches(relative) {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return failure(err)
	}
	sort.Strings(files)
	count := len(files)
	if len(files) > globLimit {
		files = files[:globLimit]
		truncated = true
	}
	// Cap the serialized result, including escaped filenames, rather than only
	// the number of paths. Long filenames yield a clear truncation marker.
	selected := make([]string, 0, len(files))
	used := 0
	for _, p := range files {
		cost := len(p)*6 + 4
		if used+cost > MaxInline/2 {
			truncated = true
			break
		}
		selected = append(selected, p)
		used += cost
	}
	return result(map[string]any{"files": selected, "count": len(selected), "matched": count, "truncated": truncated, "path": a.Path, "pattern": a.Pattern, "visited_entries": visited})
}
