// Package ignore matches paths against .gitignore rules, following git's
// documented pattern format (gitignore(5)). It is an independent
// implementation: comments, negation, directory-only rules, anchoring, "**"
// and nested .gitignore files. Paths use forward slashes, relative to the
// project root.
package ignore

import (
	"bufio"
	"io"
	"path"
	"strings"
)

type rule struct {
	base     string   // directory holding the .gitignore, "" for root
	segments []string // pattern split on "/"
	negate   bool
	dirOnly  bool
	anchored bool
}

// Matcher holds rules from any number of .gitignore files. Rules added later
// (deeper files, later lines) take precedence, as in git.
type Matcher struct {
	rules []rule
}

// maxRules bounds memory for pathological ignore files.
const maxRules = 4096

// Add parses one .gitignore located in directory base ("" or "." for root).
func (m *Matcher) Add(base string, r io.Reader) {
	if base == "." {
		base = ""
	}
	s := bufio.NewScanner(io.LimitReader(r, 1<<20))
	for s.Scan() && len(m.rules) < maxRules {
		line := strings.TrimRight(s.Text(), "\r")
		// Trailing spaces are ignored unless escaped.
		for strings.HasSuffix(line, " ") && !strings.HasSuffix(line, `\ `) {
			line = line[:len(line)-1]
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var ru rule
		ru.base = base
		if strings.HasPrefix(line, "!") {
			ru.negate, line = true, line[1:]
		} else if strings.HasPrefix(line, `\!`) || strings.HasPrefix(line, `\#`) {
			line = line[1:]
		}
		if strings.HasSuffix(line, "/") {
			ru.dirOnly, line = true, strings.TrimRight(line, "/")
		}
		// A slash at the start or middle anchors the pattern to base.
		if strings.Contains(line, "/") {
			ru.anchored = true
			line = strings.TrimPrefix(line, "/")
		}
		if line == "" {
			continue
		}
		ru.segments = strings.Split(line, "/")
		m.rules = append(m.rules, ru)
	}
}

// Ignored reports whether rel (slash-separated, relative to the root) is
// ignored. isDir selects directory-only rules. Callers prune ignored
// directories, so a file inside an ignored directory is never asked about,
// matching git's rule that a file cannot be re-included under one.
func (m *Matcher) Ignored(rel string, isDir bool) bool {
	ignored := false
	for i := range m.rules {
		ru := &m.rules[i]
		if ru.dirOnly && !isDir {
			continue
		}
		p := rel
		if ru.base != "" {
			if !strings.HasPrefix(rel, ru.base+"/") {
				continue
			}
			p = rel[len(ru.base)+1:]
		}
		if ru.match(p) {
			ignored = !ru.negate
		}
	}
	return ignored
}

func (ru *rule) match(p string) bool {
	parts := strings.Split(p, "/")
	if !ru.anchored {
		// An unanchored pattern has one segment and matches any path component
		// at the end of the path (the entry itself).
		ok, _ := path.Match(ru.segments[0], parts[len(parts)-1])
		return ok
	}
	return matchSegments(ru.segments, parts)
}

// matchSegments matches pattern segments against path segments, where "**"
// matches zero or more whole segments.
func matchSegments(pat, parts []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			rest := pat[1:]
			if len(rest) == 0 {
				return true // trailing "**" matches everything inside
			}
			for i := 0; i <= len(parts); i++ {
				if matchSegments(rest, parts[i:]) {
					return true
				}
			}
			return false
		}
		if len(parts) == 0 {
			return false
		}
		if ok, _ := path.Match(pat[0], parts[0]); !ok {
			return false
		}
		pat, parts = pat[1:], parts[1:]
	}
	return len(parts) == 0
}
