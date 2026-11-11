// Package tools implements bounded project operations. Permissions are enforced by
// the caller before Execute; Prepare is a side-effect-free admission check.
package tools

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"github.com/Sanidhya14321/goh/internal/sandbox"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

const MaxFile = 4 << 20
const MaxInline = 64 << 10
const maxFiles = 10000

type Definition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"parameters"`
}
type Result struct {
	Content string `json:"content"`
	Error   bool   `json:"error,omitempty"`
}
type Engine struct {
	root, artifacts string
	// AllowGeneratedEdits is set from user configuration before the engine is used.
	AllowGeneratedEdits bool
	// LineEdits lets edit_file replace a numbered line range (start_line,
	// end_line) instead of matching old_text. Off unless configured, for A/B
	// comparison; the whole-file expected_sha256 guard still applies.
	LineEdits bool
	// DenyPath reports project-relative paths (slash-separated) that the
	// user's file_rules deny. Denied files are not read, searched, listed or
	// mapped, as well as not edited.
	DenyPath func(rel string) bool
	// Sandbox, when set, confines shell commands and background jobs.
	Sandbox  *sandbox.Policy
	mapMu    sync.Mutex
	mapCache map[string]mapCacheEntry
	mapOrder []string
	jobs     *jobTable // shared with an engine that takes over (ShareJobs)
}
type args struct {
	Path     string `json:"path,omitempty"`
	Pattern  string `json:"pattern,omitempty"`
	Include  string `json:"include,omitempty"`
	Content  string `json:"content,omitempty"`
	Expected string `json:"expected_sha256,omitempty"`
	Old      string `json:"old_text,omitempty"`
	New      string `json:"new_text,omitempty"`
	All      bool   `json:"replace_all,omitempty"`
	Start    int    `json:"start_line,omitempty"`
	End      int    `json:"end_line,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Command  string `json:"command,omitempty"`
	Cwd      string `json:"cwd,omitempty"`
	Timeout  int    `json:"timeout_seconds,omitempty"`
}

func New(root, artifactDir string) (*Engine, error) {
	p, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	p, err = filepath.EvalSymlinks(p)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, errors.New("project must be a directory")
	}
	if artifactDir == "" {
		artifactDir = filepath.Join(os.TempDir(), "harness-output")
	}
	artifactDir, err = filepath.Abs(artifactDir)
	if err != nil {
		return nil, err
	}
	return &Engine{root: p, artifacts: artifactDir, jobs: &jobTable{}}, nil
}
func Definitions() []Definition { return (*Engine)(nil).Definitions() }
func (e *Engine) Definitions() []Definition {
	defs := baseDefinitions()
	if e != nil && e.LineEdits {
		for i, d := range defs {
			if d.Name != "edit_file" {
				continue
			}
			props := map[string]any{}
			for k, v := range d.Schema["properties"].(map[string]any) {
				props[k] = v
			}
			props["start_line"] = map[string]any{"type": "integer", "minimum": 1}
			props["end_line"] = map[string]any{"type": "integer", "minimum": 1}
			defs[i].Schema = map[string]any{"type": "object", "properties": props,
				"required": []string{"path", "new_text", "expected_sha256"}, "additionalProperties": false}
			defs[i].Description = "Edit a file against expected_sha256 (from your latest read_file). Either replace exact old_text (one occurrence unless replace_all), or replace lines start_line..end_line inclusive, numbered as read_file shows them, with new_text (empty new_text deletes the lines). With both, old_text must equal those lines."
		}
	}
	if e != nil && e.Sandbox != nil {
		for i, d := range defs {
			if d.Name == "shell" {
				defs[i].Description = strings.Replace(d.Description, "This is not sandboxed.", "Runs inside the configured OS sandbox: writes outside the project and temporary folders fail, and network access may be blocked.", 1)
			}
		}
	}
	return defs
}

func baseDefinitions() []Definition {
	s := func(properties map[string]any, required ...string) map[string]any {
		if required == nil {
			required = []string{}
		}
		return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	}
	str := func() map[string]any { return map[string]any{"type": "string"} }
	number := func(min, max int) map[string]any {
		return map[string]any{"type": "integer", "minimum": min, "maximum": max}
	}
	path := func() map[string]any { return map[string]any{"path": str()} }
	read := path()
	read["start_line"] = number(1, MaxFile)
	read["end_line"] = number(1, MaxFile)
	search := path()
	search["pattern"] = str()
	search["limit"] = number(1, 1000)
	list := path()
	list["limit"] = number(1, maxFiles)
	write := path()
	write["content"] = str()
	write["expected_sha256"] = str()
	edit := path()
	edit["old_text"] = str()
	edit["new_text"] = str()
	edit["expected_sha256"] = str()
	edit["replace_all"] = map[string]any{"type": "boolean"}
	return append([]Definition{
		{"repo_map", "Inspect a bounded declaration map (functions, types, classes, methods) without bodies: Go is parsed exactly; Python, JavaScript/TypeScript, Rust, Java/Kotlin/C#/Scala, C/C++ and Ruby use line patterns that can miss unusual forms. Optional path/query filters and max 500 symbols; reports lines, content hashes and scan bounds. No external process.", s(map[string]any{"path": str(), "query": str(), "limit": number(1, 500)})},
		{"read_artifact", "Read a retained shell/formatter output artifact by session-local artifact_id; byte offset or tail, max 8192 bytes. Use base64 for exact bytes; discarded output beyond the retained cap is unavailable.", s(map[string]any{"id": str(), "offset": map[string]any{"type": "integer", "minimum": 0}, "limit": number(1, 8192), "tail": map[string]any{"type": "boolean"}, "encoding": map[string]any{"type": "string", "enum": []string{"text", "base64"}}}, "id")},
		{"list_files", "List project files, skipping .git, node_modules and symlinks; at most 10000 entries.", s(list)},
		{"read_file", "Read UTF-8 text with line numbers and full-file SHA-256; at most 4MiB.", s(read, "path")},
		{"grep", "Search UTF-8 project text by Go regular expression with optional directory/file path and include glob; return at most 100 ordered lines. Filesystem safety/exclusions apply; no shell is invoked.", s(map[string]any{"pattern": str(), "path": str(), "include": str()}, "pattern")},
		{"search", "Search project text with a Go regular expression; return bounded matching lines.", s(search, "pattern")},
		{"glob", "Find files by glob pattern (*, ?, character classes, ** directories, {a,b} alternatives). Optional project-relative directory path. Return at most 100 lexically sorted project-relative paths; skip .git, node_modules, .harness and symlinks.", s(map[string]any{"pattern": str(), "path": str()}, "pattern")},
		{"apply_patch", "Apply an exact multi-file patch with Add/Update/Delete/Move headers. Existing source paths require expected_sha256 hash map; new destinations must not exist and parent directories must exist. Publications are individually atomic, not a multi-file transaction.", s(map[string]any{"patchText": str(), "expected_sha256": map[string]any{"type": "object", "additionalProperties": str()}}, "patchText")},
		{"create_file", "Create a new UTF-8 file; fail if it exists. Parent directory must exist.", s(map[string]any{"path": str(), "content": str()}, "path", "content")},
		{"write_file", "Create a file, or replace an existing file with expected_sha256.", s(write, "path", "content")},
		{"edit_file", "Replace exact text against expected_sha256; one occurrence unless replace_all.", s(edit, "path", "old_text", "new_text", "expected_sha256")},
		{"shell", "Run a shell command with owned process-group cleanup. This is not sandboxed.", s(map[string]any{"command": str(), "cwd": str(), "timeout_seconds": number(1, 600)}, "command")},
	}, jobDefinitions(s, str, number)...)
}
func decode(name string, raw json.RawMessage) (args, error) {
	var a args
	if len(raw) > 1<<20 {
		return a, errors.New("arguments exceed 1MiB")
	}
	// Use each tool's exact property list even when callers bypass JSON Schema.
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return a, errors.New("arguments must be a JSON object")
	}
	var def *Definition
	// Decode against the widest schema; edit_file rejects a line range
	// itself when line edits are not enabled.
	for _, d := range (&Engine{LineEdits: true}).Definitions() {
		if d.Name == name {
			v := d
			def = &v
			break
		}
	}
	if def == nil {
		return a, fmt.Errorf("unknown tool %q", name)
	}
	props := def.Schema["properties"].(map[string]any)
	for k, v := range obj {
		if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return a, fmt.Errorf("null argument %q", k)
		}
		if _, ok := props[k]; !ok {
			return a, fmt.Errorf("unknown argument %q", k)
		}
	}
	for _, k := range def.Schema["required"].([]string) {
		if _, ok := obj[k]; !ok {
			return a, fmt.Errorf("missing argument %q", k)
		}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&a); err != nil {
		return a, err
	}
	for _, k := range []string{"start_line", "end_line", "limit", "timeout_seconds"} {
		if rawNumber, ok := obj[k]; ok {
			var n int
			if err := json.Unmarshal(rawNumber, &n); err != nil || n < 1 {
				return a, fmt.Errorf("%s must be a positive integer", k)
			}
		}
	}
	if a.Start > MaxFile || a.End > MaxFile {
		return a, errors.New("line number exceeds file limit")
	}
	if name == "shell" {
		if strings.TrimSpace(a.Command) == "" {
			return a, errors.New("command is empty")
		}
		if a.Timeout == 0 {
			a.Timeout = 120
		}
		if a.Timeout < 1 || a.Timeout > 600 {
			return a, errors.New("timeout_seconds must be 1..600")
		}
	}
	if a.Start < 0 || a.End < 0 || a.Limit < 0 || a.Limit > maxFiles {
		return a, errors.New("invalid line or result limit")
	}
	if name == "search" && a.Limit > 1000 {
		return a, errors.New("search limit exceeds 1000")
	}
	if a.End > 0 && a.Start > a.End {
		return a, errors.New("end_line precedes start_line")
	}
	return a, nil
}
func clean(path string, allowRoot bool) (string, error) {
	if path == "" {
		if allowRoot {
			return ".", nil
		}
		return "", errors.New("path is required")
	}
	if filepath.IsAbs(path) {
		return "", errors.New("absolute paths are forbidden")
	}
	p := filepath.Clean(path)
	if p == ".." || strings.HasPrefix(p, ".."+string(os.PathSeparator)) {
		return "", errors.New("path escapes project")
	}
	if p == "." && !allowRoot {
		return "", errors.New("file path required")
	}
	return p, nil
}
func checked(r *os.Root, path string, allowMissing bool) error {
	parts := strings.Split(filepath.Clean(path), string(os.PathSeparator))
	p := ""
	for i, c := range parts {
		p = filepath.Join(p, c)
		st, err := r.Lstat(p)
		if err != nil {
			if allowMissing && i == len(parts)-1 && os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return errors.New("symlink paths are unsupported")
		}
		if i < len(parts)-1 && !st.IsDir() {
			return errors.New("parent is not a directory")
		}
	}
	return nil
}
func read(r *os.Root, p string) ([]byte, fs.FileMode, error) {
	if err := checked(r, p, false); err != nil {
		return nil, 0, err
	}
	lst, err := r.Lstat(p)
	if err != nil {
		return nil, 0, err
	}
	if !lst.Mode().IsRegular() {
		return nil, 0, errors.New("only regular files supported")
	}
	f, err := r.OpenFile(p, os.O_RDONLY|platform.NonBlock, 0)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	if !st.Mode().IsRegular() {
		return nil, 0, errors.New("only regular files supported")
	}
	if st.Size() > MaxFile {
		return nil, 0, errors.New("file exceeds 4MiB")
	}
	b, err := io.ReadAll(io.LimitReader(f, MaxFile+1))
	if err != nil {
		return nil, 0, err
	}
	if len(b) > MaxFile {
		return nil, 0, errors.New("file exceeds 4MiB")
	}
	if !utf8.Valid(b) || bytes.IndexByte(b, 0) >= 0 {
		return nil, 0, errors.New("file is not UTF-8 text")
	}
	return b, st.Mode().Perm(), nil
}
func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func contentCheck(s string) error {
	if len(s) > MaxFile {
		return errors.New("file exceeds 4MiB")
	}
	if !utf8.ValidString(s) || strings.IndexByte(s, 0) >= 0 {
		return errors.New("content must be UTF-8 text without NUL")
	}
	return nil
}
func (e *Engine) mutation(r *os.Root, name string, a args) (before, after []byte, mode fs.FileMode, create bool, err error) {
	if err = contentCheck(a.Content); err != nil {
		return
	}
	if err = contentCheck(a.New); err != nil {
		return
	}
	if err = checked(r, a.Path, true); err != nil {
		return
	}
	if name == "create_file" || a.Expected == "" && name == "write_file" {
		if _, statErr := r.Lstat(a.Path); !os.IsNotExist(statErr) {
			err = errors.New("destination already exists or cannot be inspected")
			return
		}
		create = true
		after = []byte(a.Content)
		mode = 0644
		return
	}
	before, mode, err = read(r, a.Path)
	if err != nil {
		return
	}
	if err = e.protectGenerated(a.Path, before); err != nil {
		return
	}
	if a.Expected == "" || hash(before) != a.Expected {
		err = errors.New("stale file: expected_sha256 does not match")
		return
	}
	if name == "edit_file" && (a.Start > 0 || a.End > 0) {
		after, err = lineEdit(before, a, e != nil && e.LineEdits)
		return
	}
	if name == "edit_file" {
		if a.Old == "" {
			err = errors.New("old_text must not be empty")
			return
		}
		n := bytes.Count(before, []byte(a.Old))
		if n == 0 && !a.All {
			// Exact text not found: accept a whitespace-tolerant match only
			// when it is unique (ambiguity is refused, never guessed).
			if replaced, ok, why := flexibleReplace(string(before), a.Old, a.New); ok {
				after = []byte(replaced)
				if len(after) > MaxFile {
					err = errors.New("result exceeds 4MiB")
				}
				return
			} else {
				err = errors.New(why)
				return
			}
		}
		if n == 0 || n > 1 && !a.All {
			err = fmt.Errorf("old_text has %d matches; expected exactly one (add surrounding lines to make it unique, or set replace_all)%s", n, matchLines(string(before), a.Old))
			return
		}
		count := 1
		if a.All {
			count = -1
		}
		after = bytes.Replace(before, []byte(a.Old), []byte(a.New), count)
	} else {
		after = []byte(a.Content)
	}
	if len(after) > MaxFile {
		err = errors.New("result exceeds 4MiB")
	}
	return
}
func (e *Engine) Prepare(name string, raw json.RawMessage) (string, bool, error) {
	if isJobTool(name) {
		return e.prepareJob(name, raw)
	}
	if name == "read_artifact" {
		return e.prepareArtifact(raw)
	}
	if name == "repo_map" {
		return e.prepareRepoMap(raw)
	}
	if name == "apply_patch" {
		return e.preparePatch(raw)
	}
	a, err := decode(name, raw)
	if err != nil {
		return "", false, err
	}
	side := name == "shell" || fileMutation(name)
	r, err := os.OpenRoot(e.root)
	if err != nil {
		return "", side, err
	}
	defer r.Close()
	if name == "shell" {
		a.Cwd, err = clean(a.Cwd, true)
		if err == nil {
			err = checked(r, a.Cwd, false)
		}
		if err != nil {
			return "", side, err
		}
		st, err := r.Stat(a.Cwd)
		if err != nil || !st.IsDir() {
			return "", side, errors.New("cwd must be a project directory")
		}
		return fmt.Sprintf("Command: %s\nDirectory: %s\nTimeout: %ds", a.Command, filepath.Join(e.root, a.Cwd), a.Timeout), true, nil
	}
	a.Path, err = clean(a.Path, name == "list_files" || name == "search" || name == "grep" || name == "glob")
	if err != nil {
		return "", side, err
	}
	if side {
		before, after, _, create, err := e.mutation(r, name, a)
		if err != nil {
			return "", true, err
		}
		return diffPreview(filepath.Join(e.root, a.Path), before, after, create), true, nil
	}
	if err = checked(r, a.Path, false); err != nil {
		return "", false, err
	}
	if name == "search" || name == "grep" {
		if name == "grep" && a.Pattern == "" {
			return "", false, errors.New("grep pattern is required")
		}
		_, err = regexp.Compile(a.Pattern)
		if err == nil && name == "grep" && a.Include != "" {
			_, err = compileGlob(a.Include)
		}
	}
	if name == "glob" {
		_, err = compileGlob(a.Pattern)
		if err == nil {
			var info fs.FileInfo
			info, err = r.Stat(a.Path)
			if err == nil && !info.IsDir() {
				err = errors.New("glob path must be a directory")
			}
		}
	}
	return filepath.Join(e.root, a.Path), false, err
}
func diffPreview(path string, before, after []byte, create bool) string {
	var out strings.Builder
	header := "--- " + path + "\n+++ " + path + "\n"
	if len(header) > 2048 {
		header = header[:2048] + " [path preview truncated]\n"
	}
	out.WriteString(header)
	if create {
		out.WriteString("(new file)\n")
	}
	// Allocate both sides a separate budget so large old contents never hide
	// all proposed new contents from the approval view.
	for _, block := range []struct {
		prefix string
		text   []byte
	}{{"-", before}, {"+", after}} {
		budget := (MaxInline - 4096) / 2
		text := string(block.text)
		for {
			line, rest, found := strings.Cut(text, "\n")
			needed := len(line) + 2
			if needed > budget {
				n := budget - 2
				if n < 0 {
					n = 0
				}
				if n > len(line) {
					n = len(line)
				}
				for n > 0 && !utf8.ValidString(line[:n]) {
					n--
				}
				out.WriteString(block.prefix)
				out.WriteString(line[:n])
				out.WriteString("\n[preview truncated; narrow the edit for complete review]\n")
				break
			}
			out.WriteString(block.prefix)
			out.WriteString(line)
			out.WriteByte('\n')
			budget -= needed
			if !found {
				break
			}
			text = rest
		}
	}
	return out.String()
}

func failure(err error) Result { return Result{Content: err.Error(), Error: true} }
func result(v any) Result {
	b, err := json.Marshal(v)
	if err != nil {
		return failure(err)
	}
	if len(b) > MaxInline {
		return failure(errors.New("tool result exceeds 64KiB; narrow the query"))
	}
	return Result{Content: string(b)}
}
func fileMutation(name string) bool {
	return name == "create_file" || name == "write_file" || name == "edit_file"
}

func (e *Engine) Execute(ctx context.Context, name string, raw json.RawMessage) Result {
	if fileMutation(name) || name == "apply_patch" {
		if paths, err := e.MutationPaths(name, raw); err == nil {
			if err := e.CheckWritable(paths); err != nil {
				return failure(err)
			}
		}
	}
	if isJobTool(name) {
		return e.executeJob(ctx, name, raw)
	}
	if name == "read_artifact" {
		return e.readArtifact(ctx, raw)
	}
	if name == "repo_map" {
		return e.repoMap(ctx, raw)
	}
	if name == "apply_patch" {
		return e.executePatch(ctx, raw)
	}
	if err := ctx.Err(); err != nil {
		return failure(err)
	}
	// File mutations validate their freshly read inputs below. Preparing here
	// would read the file and build a permission preview that nobody consumes.
	if !fileMutation(name) {
		if _, _, err := e.Prepare(name, raw); err != nil {
			return failure(err)
		}
	}
	if err := ctx.Err(); err != nil {
		return failure(err)
	}
	a, err := decode(name, raw)
	if err != nil {
		return failure(err)
	}
	if name == "shell" {
		return e.shell(ctx, a)
	}
	a.Path, err = clean(a.Path, name == "list_files" || name == "search" || name == "grep" || name == "glob")
	if err != nil {
		return failure(err)
	}
	r, err := os.OpenRoot(e.root)
	if err != nil {
		return failure(err)
	}
	defer r.Close()
	switch name {
	case "read_file":
		if e.denied(a.Path) {
			return failure(errDenied)
		}
		b, _, err := read(r, a.Path)
		if err != nil {
			return failure(err)
		}
		total := bytes.Count(b, []byte("\n")) + 1
		if a.Start == 0 {
			a.Start = 1
		}
		if a.End == 0 || a.End > total {
			a.End = total
		}
		if a.Start > total {
			return failure(errors.New("start_line exceeds file length"))
		}
		startOffset, endOffset, line := 0, len(b), 1
		for i, c := range b {
			if c == '\n' {
				if line < a.Start {
					startOffset = i + 1
				}
				if line == a.End {
					endOffset = i
					break
				}
				line++
			}
		}
		return result(map[string]any{"path": a.Path, "content": string(b[startOffset:endOffset]), "sha256": hash(b), "start_line": a.Start, "end_line": a.End, "total_lines": total})
	case "glob":
		return globFiles(ctx, r, a, e.denied)
	case "list_files", "search", "grep":
		return inventory(ctx, r, name, a, e.denied)
	case "edit_file", "write_file", "create_file":
		before, after, mode, create, err := e.mutation(r, name, a)
		if err != nil {
			return failure(err)
		}
		nonce := make([]byte, 12)
		if _, err = rand.Read(nonce); err != nil {
			return failure(err)
		}
		temp := filepath.Join(filepath.Dir(a.Path), ".harness-"+hex.EncodeToString(nonce))
		f, err := r.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if err != nil {
			return failure(err)
		}
		defer r.Remove(temp)
		_, err = f.Write(after)
		if err == nil {
			err = f.Chmod(mode)
		}
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return failure(err)
		}
		// Recheck after staging. External writers are not a universal compare-and-swap transaction.
		if err = checked(r, a.Path, true); err != nil {
			return failure(err)
		}
		if err := ctx.Err(); err != nil {
			return failure(err)
		}
		if create {
			err = r.Link(temp, a.Path)
		} else {
			current, _, readErr := read(r, a.Path)
			if readErr != nil {
				return failure(readErr)
			}
			if hash(current) != hash(before) {
				return failure(errors.New("stale file changed during publication"))
			}
			if err := ctx.Err(); err != nil {
				return failure(err)
			}
			err = r.Rename(temp, a.Path)
		}
		if err != nil {
			return failure(err)
		}
		parent, syncErr := r.Open(filepath.Dir(a.Path))
		if syncErr != nil {
			return failure(fmt.Errorf("file published but parent directory could not be opened for synchronization: %w", syncErr))
		}
		syncErr = platform.SyncDir(parent)
		parent.Close()
		if syncErr != nil {
			return failure(fmt.Errorf("file published but parent directory synchronization failed: %w", syncErr))
		}
		return result(map[string]any{"path": a.Path, "sha256": hash(after), "bytes": len(after), "created": create})
	}
	return failure(errors.New("unknown tool"))
}
func inventory(ctx context.Context, r *os.Root, name string, a args, deny func(string) bool) Result {
	limit := a.Limit
	if limit == 0 {
		if name == "search" || name == "grep" {
			limit = 100
		} else {
			limit = 1000
		}
	}
	var files []string
	visited := 0
	truncated := false
	ignored := newIgnoreFilter(r, filepath.ToSlash(a.Path))
	err := fs.WalkDir(r.FS(), filepath.ToSlash(a.Path), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
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
			if ignored.skip(p, d) {
				return fs.SkipDir
			}
			return nil
		}
		if ignored.skip(p, d) {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		files = append(files, p)
		if len(files) >= maxFiles {
			truncated = true
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return failure(err)
	}
	sort.Strings(files)
	if name == "list_files" {
		if len(files) > limit {
			files = files[:limit]
			truncated = true
		}
		return result(map[string]any{"files": files, "truncated": truncated})
	}
	re, err := regexp.Compile(a.Pattern)
	if err != nil {
		return failure(err)
	}
	type match struct {
		Path string `json:"path"`
		Line int    `json:"line"`
		Text string `json:"text"`
	}
	var include globProgram
	if name == "grep" && a.Include != "" {
		include, err = compileGlob(a.Include)
		if err != nil {
			return failure(err)
		}
	}
	matches := []match{}
	bytesUsed := 0
	skipped := 0
	for _, p := range files {
		if err := ctx.Err(); err != nil {
			return failure(err)
		}
		if name == "grep" && a.Include != "" {
			relative := p
			if a.Path != "." {
				relative = strings.TrimPrefix(p, a.Path+"/")
				if p == a.Path {
					relative = filepath.Base(p)
				}
			}
			if !include.matches(relative) {
				continue
			}
		}
		b, _, err := read(r, p)
		if err != nil {
			skipped++
			continue
		}
		i := 0
		for l := range strings.SplitSeq(string(b), "\n") {
			i++
			if re.MatchString(l) {
				if len(matches) >= limit || bytesUsed+len(l)+len(p)+100 > MaxInline/2 {
					truncated = true
					break
				}
				matches = append(matches, match{p, i, l})
				bytesUsed += len(l) + len(p) + 100
			}
		}
		if truncated && len(matches) >= limit {
			break
		}
	}
	return result(map[string]any{"matches": matches, "truncated": truncated, "skipped_files": skipped})
}

// denied reports whether a project-relative path is denied by file_rules.
func (e *Engine) denied(rel string) bool {
	if e == nil || e.DenyPath == nil || rel == "" || rel == "." {
		return false
	}
	return e.DenyPath(filepath.ToSlash(rel)) || e.DenyPath(CanonicalRel(e.root, rel))
}

// errDenied is returned for direct access to a denied path.
var errDenied = errors.New("access to this path is denied by the user's file_rules")

// CheckWritable refuses mutations of Git metadata in any folder (a planted
// hook or config runs code on the next git command) and paths the user's
// file_rules deny. It applies whatever the grants or checkpoint mode.
func (e *Engine) CheckWritable(paths []string) error {
	for _, p := range paths {
		for _, part := range strings.Split(filepath.ToSlash(p), "/") {
			if strings.EqualFold(part, ".git") {
				return fmt.Errorf("refusing to change Git metadata (%s); edit the repository with git commands instead", p)
			}
		}
		if e.denied(p) {
			return fmt.Errorf("%w: %s", errDenied, p)
		}
	}
	return nil
}

// sandboxPolicy adds files the user's file_rules deny to the sandbox policy,
// so a shell command cannot read them either (enforced where the sandbox can
// hide single files; see sandbox.Policy.DenyFiles).
func (e *Engine) sandboxPolicy() *sandbox.Policy {
	if e.Sandbox == nil || e.DenyPath == nil {
		return e.Sandbox
	}
	p := *e.Sandbox
	p.DenyFiles = append([]string(nil), p.DenyFiles...)
	visited := 0
	_ = filepath.WalkDir(e.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if visited++; visited > 20000 {
			return fs.SkipAll
		}
		rel, rerr := filepath.Rel(e.root, path)
		if rerr != nil || rel == "." {
			return nil
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules") {
			return fs.SkipDir
		}
		if !d.IsDir() && e.denied(rel) {
			p.DenyFiles = append(p.DenyFiles, path)
		}
		return nil
	})
	return &p
}
