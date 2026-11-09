package commands

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/core"
)

// Folders scanned in the project for Markdown command files, as in OpenCode
// (.opencode/command) and other tools. Each NAME.md defines /NAME.
var commandDirs = []string{".opencode/command", ".opencode/commands", ".claude/commands", ".harness/commands"}

// Skipped explains a command file that was not loaded.
type Skipped struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// Discover reads project command files. Only regular top-level *.md files are
// read (no symlinks, no subfolders), each bounded by MaxTemplate. Frontmatter
// "description" is used; other keys such as agent or model are ignored, so a
// repository cannot change the mode, model or provider a command runs with.
// The first folder in commandDirs wins for duplicate names.
func Discover(project string) (map[string]core.CommandConfig, []Skipped) {
	found := map[string]core.CommandConfig{}
	var skipped []Skipped
	for _, sub := range commandDirs {
		dir := filepath.Join(project, sub)
		if info, err := os.Lstat(dir); err != nil || !info.IsDir() {
			continue // missing, or a symlinked folder
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			skipped = append(skipped, Skipped{dir, err.Error()})
			continue
		}
		for _, e := range entries {
			name, ok := strings.CutSuffix(e.Name(), ".md")
			if !ok || !e.Type().IsRegular() {
				continue
			}
			path := filepath.Join(dir, e.Name())
			if !namePattern.MatchString(name) || reserved[name] {
				skipped = append(skipped, Skipped{path, "invalid or reserved command name"})
				continue
			}
			if _, dup := found[name]; dup {
				skipped = append(skipped, Skipped{path, "duplicate of an earlier command folder"})
				continue
			}
			def, reason := readCommandFile(path)
			if reason != "" {
				skipped = append(skipped, Skipped{path, reason})
				continue
			}
			found[name] = def
		}
	}
	sort.Slice(skipped, func(i, j int) bool { return skipped[i].Path < skipped[j].Path })
	return found, skipped
}

func readCommandFile(path string) (core.CommandConfig, string) {
	f, err := os.Open(path)
	if err != nil {
		return core.CommandConfig{}, err.Error()
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, MaxTemplate+4096+1))
	if err != nil {
		return core.CommandConfig{}, err.Error()
	}
	if len(b) > MaxTemplate+4096 || !utf8.Valid(b) {
		return core.CommandConfig{}, "too large or not UTF-8"
	}
	text := strings.ReplaceAll(string(b), "\r\n", "\n")
	var def core.CommandConfig
	if rest, ok := strings.CutPrefix(text, "---\n"); ok {
		front, body, closed := strings.Cut(rest, "\n---")
		if !closed {
			return def, "unterminated frontmatter"
		}
		for _, line := range strings.Split(front, "\n") {
			if key, value, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(key) == "description" {
				def.Description = strings.Trim(strings.TrimSpace(value), `"'`)
			}
		}
		text = strings.TrimPrefix(strings.TrimPrefix(body, "\n"), "\n")
	}
	def.Template = strings.TrimSpace(text)
	if def.Template == "" || len(def.Template) > MaxTemplate {
		return def, "empty or too large template"
	}
	if len(def.Description) > 8192 {
		return def, "description too long"
	}
	return def, ""
}

// Merge combines project command files with configured commands; configured
// commands always win. File commands beyond the 256-command limit are dropped.
func Merge(configured, files map[string]core.CommandConfig) map[string]core.CommandConfig {
	out := make(map[string]core.CommandConfig, len(configured)+len(files))
	for k, v := range configured {
		out[k] = v
	}
	names := make([]string, 0, len(files))
	for k := range files {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		if _, ok := out[k]; !ok && len(out) < 256 {
			out[k] = files[k]
		}
	}
	return out
}
