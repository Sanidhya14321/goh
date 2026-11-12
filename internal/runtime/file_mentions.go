package runtime

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// mentionText is the part of a prompt the user wrote, without attached file
// contents (core.FileBlock): mentions inside an attachment are file content,
// not the user's request, so they never attach further files.
func mentionText(prompt string) string {
	return core.StripFileBlocks(prompt)
}

// mentions returns the @paths in the user's own text, in order. A path with
// spaces or quotes is written @"path with spaces" (Go string quoting, as the
// file picker inserts it); other mentions end at whitespace, with trailing
// punctuation dropped.
func mentions(prompt string) []string {
	text := mentionText(prompt)
	var out []string
	for i := 0; i < len(text); i++ {
		if text[i] != '@' || (i > 0 && !unicode.IsSpace(rune(text[i-1])) && text[i-1] != '(') {
			continue
		}
		rest := text[i+1:]
		if strings.HasPrefix(rest, `"`) {
			if q, err := strconv.QuotedPrefix(rest); err == nil {
				if path, err := strconv.Unquote(q); err == nil && path != "" {
					out = append(out, path)
				}
				i += len(q)
				continue
			}
		}
		end := strings.IndexFunc(rest, unicode.IsSpace)
		if end < 0 {
			end = len(rest)
		}
		if path := strings.TrimRight(rest[:end], ".,;:!?)\"'"); path != "" {
			out = append(out, path)
		}
		i += end
	}
	return out
}

// pathDenied applies file_rules to a project-relative path both as written
// and as it exists on disk (symlinked folders resolved, stored case), so a
// different spelling of a denied file is denied too.
func (a *Agent) pathDenied(rel string) bool {
	return tools.DeniedPath(a.Store.Project, rel, func(p string, fold bool) bool {
		return a.Config.FileDecisionFold([]string{p}, fold) == "deny"
	})
}

const (
	maxMentionFiles     = 5
	maxMentionFileBytes = 64 << 10
	maxMentionTotal     = 256 << 10
)

// withFileMentions inlines text files named as @path, as OpenCode does, so
// the model sees them without a read round trip. Only regular UTF-8 files
// inside the project are attached, read through os.Root; images, missing
// paths, directories, binaries and oversized files are left as plain
// mentions for the model to handle with its tools.
func (a *Agent) withFileMentions(prompt string) (string, []string) {
	root, err := os.OpenRoot(a.Store.Project)
	if err != nil {
		return prompt, nil
	}
	defer root.Close()
	var b strings.Builder
	var attached []string
	seen := map[string]bool{}
	total := 0
	for _, path := range mentions(prompt) {
		if seen[path] || imageTypes[strings.ToLower(filepath.Ext(path))] != "" || len(attached) == maxMentionFiles {
			continue
		}
		seen[path] = true
		clean := filepath.Clean(filepath.FromSlash(path))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			continue
		}
		if a.pathDenied(clean) {
			continue // file_rules deny covers attachments too
		}
		info, err := root.Lstat(clean)
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxMentionFileBytes {
			continue
		}
		f, err := root.Open(clean)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(f, maxMentionFileBytes+1))
		f.Close()
		if err != nil || len(data) > maxMentionFileBytes || bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) || total+len(data) > maxMentionTotal {
			continue
		}
		total += len(data)
		name := filepath.ToSlash(clean)
		attached = append(attached, name)
		b.WriteString("\n\n" + core.FileBlock(name, strings.TrimRight(string(data), "\n")))
	}
	return prompt + b.String(), attached
}
