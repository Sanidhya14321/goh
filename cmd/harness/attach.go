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

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tools"
)

const (
	maxAttachBytes      = 256 << 10
	maxAttachTotalBytes = 1 << 20
)

var attachImageExt = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true}

// attachFiles renders `goh run --file` attachments. Text files are inlined in
// <file> blocks; images become @path mentions, which the runtime validates and
// attaches. Paths are read through os.Root, so they must stay in the project.
// A path denied by file_rules (in any spelling) is refused, and anything but
// a regular file is refused before it is opened, so a named pipe cannot hang
// the CLI.
func attachFiles(project string, paths []string, denied func(rel string, fold bool) bool) (string, error) {
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
		if tools.DeniedPath(root.Name(), name, denied) {
			return "", fmt.Errorf("--file %s: access to this path is denied by the user's file_rules", p)
		}
		if attachImageExt[strings.ToLower(filepath.Ext(clean))] {
			if strings.ContainsAny(name, " \t\n") {
				return "", fmt.Errorf("--file %s: image paths cannot contain spaces", p)
			}
			out.WriteString(" @" + name)
			continue
		}
		if info, err := root.Lstat(clean); err != nil {
			return "", fmt.Errorf("--file %s: %w", p, err)
		} else if !info.Mode().IsRegular() {
			return "", fmt.Errorf("--file %s: not a regular file", p)
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
		out.WriteString("\n\n" + core.FileBlock(name, strings.TrimRight(string(b), "\n")))
	}
	return out.String(), nil
}

// commandPrompt turns `goh run --command NAME words` into "/NAME words",
// which the runtime expands like a slash command typed in the TUI.
func commandPrompt(name, message string) (string, error) {
	name = strings.TrimPrefix(name, "/")
	if name == "" || strings.ContainsAny(name, " \t\n/") {
		return "", errors.New("--command needs a single command name")
	}
	return strings.TrimSpace("/" + name + " " + message), nil
}
