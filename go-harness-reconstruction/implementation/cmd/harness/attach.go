package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	maxAttachBytes      = 256 << 10
	maxAttachTotalBytes = 1 << 20
)

var attachImageExt = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true}

// attachFiles renders `goh run --file` attachments. Text files are inlined in
// <file> blocks; images become @path mentions, which the runtime validates and
// attaches. Paths are read through os.Root, so they must stay in the project.
func attachFiles(project string, paths []string) (string, error) {
	root, err := os.OpenRoot(project)
	if err != nil {
		return "", err
	}
	defer root.Close()
	var out strings.Builder
	total := 0
	for _, p := range paths {
		clean := filepath.Clean(filepath.FromSlash(p))
		if filepath.IsAbs(clean) {
			if rel, err := filepath.Rel(root.Name(), clean); err == nil {
				clean = rel
			}
		}
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("--file %s: must be inside the project", p)
		}
		name := filepath.ToSlash(clean)
		if attachImageExt[strings.ToLower(filepath.Ext(clean))] {
			if strings.ContainsAny(name, " \t\n") {
				return "", fmt.Errorf("--file %s: image paths cannot contain spaces", p)
			}
			out.WriteString(" @" + name)
			continue
		}
		f, err := root.Open(clean)
		if err != nil {
			return "", fmt.Errorf("--file %s: %w", p, err)
		}
		info, err := f.Stat()
		if err == nil && !info.Mode().IsRegular() {
			err = errors.New("not a regular file")
		}
		var b []byte
		if err == nil {
			b, err = io.ReadAll(io.LimitReader(f, maxAttachBytes+1))
		}
		f.Close()
		if err != nil {
			return "", fmt.Errorf("--file %s: %w", p, err)
		}
		if len(b) > maxAttachBytes {
			return "", fmt.Errorf("--file %s: larger than 256 KiB", p)
		}
		if bytes.IndexByte(b, 0) >= 0 || !utf8.Valid(b) {
			return "", fmt.Errorf("--file %s: not a text file", p)
		}
		if total += len(b); total > maxAttachTotalBytes {
			return "", errors.New("--file attachments exceed 1 MiB in total")
		}
		fmt.Fprintf(&out, "\n\n<file path=%q>\n%s\n</file>", name, strings.TrimRight(string(b), "\n"))
	}
	return out.String(), nil
}
