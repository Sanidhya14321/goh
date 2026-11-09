package runtime

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

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
	for _, field := range strings.Fields(prompt) {
		path, ok := strings.CutPrefix(field, "@")
		path = strings.TrimRight(path, ".,;:!?)\"'")
		if !ok || path == "" || seen[path] || imageTypes[strings.ToLower(filepath.Ext(path))] != "" || len(attached) == maxMentionFiles {
			continue
		}
		seen[path] = true
		clean := filepath.Clean(filepath.FromSlash(path))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			continue
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
		fmt.Fprintf(&b, "\n\n<file path=%q>\n%s\n</file>", name, strings.TrimRight(string(data), "\n"))
	}
	return prompt + b.String(), attached
}
