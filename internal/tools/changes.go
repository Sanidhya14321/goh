package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MutationPaths is an admission-time inventory, not a snapshot of the entire
// process filesystem. Shell commands and formatter side effects are not inferred.
func (e *Engine) MutationPaths(name string, raw json.RawMessage) ([]string, error) {
	switch name {
	case "create_file", "write_file", "edit_file":
		a, err := decode(name, raw)
		if err != nil {
			return nil, err
		}
		p, err := clean(a.Path, false)
		if err != nil {
			return nil, err
		}
		return []string{p}, nil
	case "apply_patch":
		a, err := decodePatch(raw)
		if err != nil {
			return nil, err
		}
		ops, err := parsePatch(a.PatchText)
		if err != nil {
			return nil, err
		}
		paths := map[string]bool{}
		for _, op := range ops {
			paths[op.path] = true
			if op.destination != "" {
				paths[op.destination] = true
			}
		}
		out := make([]string, 0, len(paths))
		for p := range paths {
			out = append(out, p)
		}
		sort.Strings(out)
		return out, nil
	}
	return nil, nil
}

// FileDigest reports current bytes after post-edit formatting without returning
// the entire file to the model. Missing files are represented explicitly.
func (e *Engine) FileDigest(path string) (map[string]any, error) {
	p, err := clean(path, false)
	if err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(e.root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if err = checked(r, p, true); err != nil {
		return nil, err
	}
	if _, err = r.Lstat(p); os.IsNotExist(err) {
		return map[string]any{"path": p, "exists": false}, nil
	}
	b, _, err := read(r, p)
	if err != nil {
		return nil, err
	}
	return map[string]any{"path": p, "exists": true, "sha256": hash(b), "bytes": len(b)}, nil
}
func quoteArgument(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }

// RunCommand reuses the owned-process lifecycle and bounded artifact writer.
// Every argument is quoted literally; only explicitly authorized callers use it.
func (e *Engine) RunCommand(ctx context.Context, command string, argv []string, env map[string]string, timeout int) Result {
	if timeout < 1 || timeout > 120 || !filepath.IsAbs(command) {
		return failure(errors.New("invalid formatter command or timeout"))
	}
	var b strings.Builder
	b.WriteString("exec ")
	if len(env) > 0 {
		b.WriteString("/usr/bin/env ")
		keys := make([]string, 0, len(env))
		for k := range env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(quoteArgument(k + "=" + env[k]))
			b.WriteByte(' ')
		}
	}
	b.WriteString(quoteArgument(command))
	for _, arg := range argv {
		b.WriteByte(' ')
		b.WriteString(quoteArgument(arg))
	}
	return e.shell(ctx, args{Command: b.String(), Cwd: ".", Timeout: timeout})
}
