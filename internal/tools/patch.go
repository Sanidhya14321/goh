package tools

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const maxPatchFiles = 100
const maxPatchTotal = 16 << 20

type patchArguments struct {
	PatchText string            `json:"patchText"`
	Expected  map[string]string `json:"expected_sha256,omitempty"`
}
type patchLine struct {
	kind byte
	text string
}
type patchHunk struct {
	anchor string
	lines  []patchLine
	eof    bool
}
type patchOperation struct {
	kind, path, destination string
	added                   []string
	hunks                   []patchHunk
}
type patchChange struct {
	kind, path, destination string
	before, after           []byte
	mode                    fs.FileMode
	temp                    string
}

func decodePatch(raw json.RawMessage) (patchArguments, error) {
	var a patchArguments
	if len(raw) > 1<<20 {
		return a, errors.New("patch arguments exceed 1MiB")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return a, errors.New("patch arguments must be a JSON object")
	}
	for key, value := range fields {
		if key != "patchText" && key != "expected_sha256" {
			return a, fmt.Errorf("unknown patch argument %q", key)
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return a, fmt.Errorf("null patch argument %q", key)
		}
	}
	if _, ok := fields["patchText"]; !ok {
		return a, errors.New("patchText is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&a); err != nil {
		return a, err
	}
	if err := contentCheck(a.PatchText); err != nil {
		return a, err
	}
	return a, nil
}
func parsePatch(text string) ([]patchOperation, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if strings.Count(text, "\n") > 65536 {
		return nil, errors.New("patch exceeds 65536 lines")
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(lines) < 2 || lines[0] != "*** Begin Patch" || lines[len(lines)-1] != "*** End Patch" {
		return nil, errors.New("patch must start with *** Begin Patch and finish with *** End Patch")
	}
	var operations []patchOperation
	for i := 1; i < len(lines)-1; {
		if len(operations) >= maxPatchFiles {
			return nil, errors.New("patch exceeds 100 file operations")
		}
		var op patchOperation
		header := lines[i]
		i++
		switch {
		case strings.HasPrefix(header, "*** Add File: "):
			op.kind = "add"
			op.path = strings.TrimPrefix(header, "*** Add File: ")
		case strings.HasPrefix(header, "*** Update File: "):
			op.kind = "update"
			op.path = strings.TrimPrefix(header, "*** Update File: ")
		case strings.HasPrefix(header, "*** Delete File: "):
			op.kind = "delete"
			op.path = strings.TrimPrefix(header, "*** Delete File: ")
		default:
			return nil, fmt.Errorf("invalid patch file header %q", header)
		}
		var err error
		op.path, err = clean(op.path, false)
		if err != nil {
			return nil, err
		}
		if op.kind == "add" {
			for i < len(lines)-1 && strings.HasPrefix(lines[i], "+") {
				op.added = append(op.added, lines[i][1:])
				i++
			}
		}
		if op.kind == "update" {
			if i < len(lines)-1 && strings.HasPrefix(lines[i], "*** Move to: ") {
				op.destination, err = clean(strings.TrimPrefix(lines[i], "*** Move to: "), false)
				if err != nil {
					return nil, err
				}
				if op.destination == op.path {
					return nil, errors.New("patch move destination equals source")
				}
				i++
			}
			for i < len(lines)-1 && (lines[i] == "@@" || strings.HasPrefix(lines[i], "@@ ")) {
				h := patchHunk{}
				if lines[i] != "@@" {
					h.anchor = strings.TrimPrefix(lines[i], "@@ ")
				}
				i++
				for i < len(lines)-1 {
					line := lines[i]
					if line == "*** End of File" {
						h.eof = true
						i++
						break
					}
					if len(line) == 0 || !strings.ContainsRune(" +-", rune(line[0])) {
						break
					}
					h.lines = append(h.lines, patchLine{line[0], line[1:]})
					i++
				}
				if len(h.lines) == 0 {
					return nil, errors.New("patch update hunk is empty")
				}
				op.hunks = append(op.hunks, h)
			}
			if len(op.hunks) == 0 && op.destination == "" {
				return nil, errors.New("patch update requires at least one hunk")
			}
		}
		operations = append(operations, op)
	}
	if len(operations) == 0 {
		return nil, errors.New("patch rejected: empty patch")
	}
	return operations, nil
}

type textLine struct{ text, ending string }

func splitTextLines(content []byte) []textLine {
	var lines []textLine
	remaining := string(content)
	for remaining != "" {
		line, rest, found := strings.Cut(remaining, "\n")
		ending := ""
		if found {
			ending = "\n"
			if strings.HasSuffix(line, "\r") {
				line = strings.TrimSuffix(line, "\r")
				ending = "\r\n"
			}
		}
		lines = append(lines, textLine{line, ending})
		if !found {
			break
		}
		remaining = rest
	}
	return lines
}
func applyPatchHunks(before []byte, hunks []patchHunk) ([]byte, error) {
	lines := splitTextLines(before)
	ending := "\n"
	for _, line := range lines {
		if line.ending != "" {
			ending = line.ending
			break
		}
	}
	out := make([]textLine, 0, len(lines))
	work := 0
	cursor := 0
	for _, h := range hunks {
		searchFrom := cursor
		if h.anchor != "" {
			found := -1
			for i := cursor; i < len(lines); i++ {
				if lines[i].text == h.anchor {
					if found >= 0 {
						return nil, errors.New("patch anchor is ambiguous")
					}
					found = i
				}
			}
			if found < 0 {
				return nil, errors.New("patch anchor not found")
			}
			searchFrom = found + 1
		}
		var old []string
		for _, line := range h.lines {
			if line.kind != '+' {
				old = append(old, line.text)
			}
		}
		position := -1
		if len(old) == 0 {
			if len(lines) == 0 {
				position = 0
			} else if h.eof {
				position = len(lines)
			} else {
				return nil, errors.New("insertion-only patch needs exact context or *** End of File")
			}
		} else {
			for i := searchFrom; i+len(old) <= len(lines); i++ {
				if h.eof && i+len(old) != len(lines) {
					continue
				}
				matches := true
				for j, text := range old {
					work++
					if work > 20000000 {
						return nil, errors.New("patch exact-context matching budget exhausted; supply more specific context")
					}
					if lines[i+j].text != text {
						matches = false
						break
					}
				}
				if matches {
					if position >= 0 {
						return nil, errors.New("patch context is ambiguous; supply more exact context")
					}
					position = i
				}
			}
		}
		if position < cursor {
			return nil, errors.New("patch exact context not found or hunks overlap")
		}
		out = append(out, lines[cursor:position]...)
		oldCursor := position
		for _, line := range h.lines {
			switch line.kind {
			case ' ':
				out = append(out, lines[oldCursor])
				oldCursor++
			case '-':
				oldCursor++
			case '+':
				out = append(out, textLine{line.text, ending})
			}
		}
		cursor = position + len(old)
	}
	out = append(out, lines[cursor:]...)
	// Preserve a missing final newline when replacing an unterminated final line.
	if len(before) > 0 && before[len(before)-1] != '\n' && cursor == len(lines) && len(out) > 0 {
		out[len(out)-1].ending = ""
	}
	var b strings.Builder
	for i, line := range out {
		b.WriteString(line.text)
		if line.ending == "" && i < len(out)-1 {
			b.WriteString(ending)
		} else {
			b.WriteString(line.ending)
		}
		if b.Len() > MaxFile {
			return nil, errors.New("patched file exceeds 4MiB")
		}
	}
	return []byte(b.String()), nil
}
func expectedPatchHash(a patchArguments, path string, before []byte) error {
	expected, ok := a.Expected[path]
	if !ok {
		return fmt.Errorf("expected_sha256[%q] is required for existing files", path)
	}
	if len(expected) != 64 || expected != hash(before) {
		return fmt.Errorf("stale patch source %s: expected_sha256 does not match", path)
	}
	return nil
}
func (e *Engine) planPatch(root *os.Root, a patchArguments) ([]patchChange, error) {
	operations, err := parsePatch(a.PatchText)
	if err != nil {
		return nil, err
	}
	claimed := map[string]bool{}
	expectedUsed := map[string]bool{}
	var changes []patchChange
	total := 0
	claim := func(path string) error {
		if claimed[path] {
			return fmt.Errorf("patch path used by multiple operations: %s", path)
		}
		claimed[path] = true
		return nil
	}
	for _, op := range operations {
		if err := claim(op.path); err != nil {
			return nil, err
		}
		if op.destination != "" {
			if err := claim(op.destination); err != nil {
				return nil, err
			}
		}
		change := patchChange{kind: op.kind, path: op.path, destination: op.destination, mode: 0644}
		if op.kind == "add" {
			if len(op.added) > 0 {
				change.after = []byte(strings.Join(op.added, "\n") + "\n")
			}
			if err := patchDestinationAbsent(root, op.path); err != nil {
				return nil, err
			}
		} else {
			change.before, change.mode, err = read(root, op.path)
			if err != nil {
				return nil, err
			}
			if err = e.protectGenerated(op.path, change.before); err != nil {
				return nil, err
			}
			if err = expectedPatchHash(a, op.path, change.before); err != nil {
				return nil, err
			}
			expectedUsed[op.path] = true
			if op.kind == "update" {
				change.after, err = applyPatchHunks(change.before, op.hunks)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", op.path, err)
				}
			}
			if op.destination != "" {
				if err = patchDestinationAbsent(root, op.destination); err != nil {
					return nil, err
				}
			}
		}
		if err := contentCheck(string(change.after)); err != nil {
			return nil, err
		}
		total += len(change.before) + len(change.after)
		if total > maxPatchTotal {
			return nil, errors.New("patch exceeds 16MiB aggregate file data")
		}
		changes = append(changes, change)
	}
	for path := range a.Expected {
		if !expectedUsed[path] {
			return nil, fmt.Errorf("expected_sha256 contains an unused or noncanonical path %q", path)
		}
	}
	if output := patchSuccess(changes); output.Error {
		return nil, errors.New("planned patch result exceeds 64KiB; split the patch into smaller groups")
	}
	return changes, nil
}
func patchDestinationAbsent(root *os.Root, path string) error {
	if err := checked(root, path, true); err != nil {
		return err
	}
	if _, err := root.Lstat(path); !os.IsNotExist(err) {
		return fmt.Errorf("patch destination exists or cannot be inspected: %s", path)
	}
	return nil
}
func patchPreview(project string, changes []patchChange) string {
	if len(changes) == 0 {
		return ""
	}
	var preview strings.Builder
	preview.WriteString("Apply patch (all files; independent publications, no multi-file transaction):\n")
	for i, change := range changes {
		fmt.Fprintf(&preview, "File %d: %s %s", i+1, strings.ToUpper(change.kind), filepath.Join(project, change.path))
		if change.destination != "" {
			fmt.Fprintf(&preview, " -> %s", filepath.Join(project, change.destination))
		}
		preview.WriteByte('\n')
		if preview.Len() > MaxInline {
			return ""
		}
	}
	per := (MaxInline - preview.Len() - 128) / len(changes)
	if per < 192 {
		return ""
	}
	for i, change := range changes {
		fmt.Fprintf(&preview, "\nFile %d before (-):\n", i+1)
		preview.WriteString(patchExcerpt("-", change.before, (per-96)/2))
		fmt.Fprintf(&preview, "File %d after (+):\n", i+1)
		preview.WriteString(patchExcerpt("+", change.after, (per-96)/2))
	}
	if preview.Len() > MaxInline {
		return ""
	}
	return preview.String()
}
func patchExcerpt(prefix string, content []byte, budget int) string {
	const marker = "[excerpt truncated]\n"
	var excerpt strings.Builder
	remaining := string(content)
	for {
		line, rest, found := strings.Cut(remaining, "\n")
		available := budget - excerpt.Len() - len(marker) - len(prefix) - 1
		if available < 0 {
			return excerpt.String() + marker
		}
		if len(line) > available {
			cut := available
			for cut > 0 && !utf8.ValidString(line[:cut]) {
				cut--
			}
			excerpt.WriteString(prefix)
			excerpt.WriteString(line[:cut])
			excerpt.WriteByte('\n')
			excerpt.WriteString(marker)
			return excerpt.String()
		}
		excerpt.WriteString(prefix)
		excerpt.WriteString(line)
		excerpt.WriteByte('\n')
		if !found {
			return excerpt.String()
		}
		remaining = rest
	}
}
func patchSuccess(changes []patchChange) Result {
	outcomes := make([]map[string]any, 0, len(changes))
	for _, change := range changes {
		outcome := map[string]any{"path": change.path, "operation": change.kind}
		if change.kind != "delete" {
			outcome["sha256"] = hash(change.after)
		}
		if change.destination != "" {
			outcome["destination"] = change.destination
			outcome["operation"] = "move"
		}
		outcomes = append(outcomes, outcome)
	}
	return result(map[string]any{"status": "completed", "files": outcomes})
}

func (e *Engine) preparePatch(raw json.RawMessage) (preview string, side bool, finalErr error) {
	defer func() {
		if finalErr != nil {
			finalErr = errors.New(patchErrorText(finalErr))
		}
	}()
	a, err := decodePatch(raw)
	if err != nil {
		return "", true, err
	}
	root, err := os.OpenRoot(e.root)
	if err != nil {
		return "", true, err
	}
	defer root.Close()
	changes, err := e.planPatch(root, a)
	if err != nil {
		return "", true, err
	}
	preview = patchPreview(e.root, changes)
	if preview == "" {
		return "", true, errors.New("patch paths exceed permission preview limit")
	}
	return preview, true, nil
}
func recheckPatch(root *os.Root, change patchChange) error {
	if change.kind == "add" {
		return patchDestinationAbsent(root, change.path)
	}
	current, _, err := read(root, change.path)
	if err != nil {
		return err
	}
	if hash(current) != hash(change.before) {
		return fmt.Errorf("stale patch source changed before publication: %s", change.path)
	}
	if change.destination != "" {
		return patchDestinationAbsent(root, change.destination)
	}
	return nil
}
func syncPatchParent(root *os.Root, path string) error {
	parent, err := root.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer parent.Close()
	return platform.SyncDir(parent)
}
func stagePatch(root *os.Root, change *patchChange) error {
	if change.kind == "delete" {
		return nil
	}
	destination := change.path
	if change.destination != "" {
		destination = change.destination
	}
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	change.temp = filepath.Join(filepath.Dir(destination), ".harness-patch-"+hex.EncodeToString(nonce[:]))
	f, err := root.OpenFile(change.temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, change.mode)
	if err != nil {
		return err
	}
	_, err = f.Write(change.after)
	if err == nil {
		err = f.Chmod(change.mode)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	return err
}
func publishPatch(root *os.Root, change patchChange) (changed bool, err error) {
	switch {
	case change.kind == "delete":
		err = root.Remove(change.path)
		changed = err == nil
		if err == nil {
			err = syncPatchParent(root, change.path)
		}
	case change.kind == "add":
		err = root.Link(change.temp, change.path)
		changed = err == nil
		if err == nil {
			err = syncPatchParent(root, change.path)
		}
	case change.destination != "":
		err = root.Link(change.temp, change.destination)
		changed = err == nil
		if err != nil {
			return
		}
		if err = syncPatchParent(root, change.destination); err != nil {
			return
		}
		current, _, readErr := read(root, change.path)
		if readErr != nil {
			return true, readErr
		}
		if hash(current) != hash(change.before) {
			return true, errors.New("move destination published, but source changed; source retained")
		}
		err = root.Remove(change.path)
		if err == nil {
			err = syncPatchParent(root, change.path)
		}
	default:
		err = root.Rename(change.temp, change.path)
		changed = err == nil
		if err == nil {
			err = syncPatchParent(root, change.path)
		}
	}
	return
}
func patchErrorText(err error) string {
	message := err.Error()
	if len(message) > 2048 {
		message = message[:2048]
		for !utf8.ValidString(message) {
			message = message[:len(message)-1]
		}
		message += " [error truncated]"
	}
	return message
}
func patchSafeFailure(err error) Result { return failure(errors.New(patchErrorText(err))) }
func patchExecutionFailure(err error, published []string) Result {
	message := patchErrorText(err)
	if len(published) == 0 {
		return patchSafeFailure(errors.New(message))
	}
	report := map[string]any{"status": "interrupted", "error": message, "published": published, "published_count": len(published), "published_truncated": false, "effects": "Some patch publications may have completed; inspect files before acknowledging recovery. No rollback or automatic replay."}
	for {
		encoded := mustPatchJSON(report)
		if len(encoded) <= MaxInline {
			return Result{Error: true, Content: encoded}
		}
		published = published[:len(published)-1]
		report["published"] = published
		report["published_truncated"] = true
	}
}

func mustPatchJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
func (e *Engine) executePatch(ctx context.Context, raw json.RawMessage) Result {
	if err := ctx.Err(); err != nil {
		return patchSafeFailure(err)
	}
	a, err := decodePatch(raw)
	if err != nil {
		return patchSafeFailure(err)
	}
	root, err := os.OpenRoot(e.root)
	if err != nil {
		return patchSafeFailure(err)
	}
	defer root.Close()
	changes, err := e.planPatch(root, a)
	if err != nil {
		return patchSafeFailure(err)
	}
	if patchPreview(e.root, changes) == "" {
		return patchSafeFailure(errors.New("patch paths exceed permission preview limit"))
	}
	defer func() {
		for _, change := range changes {
			if change.temp != "" {
				root.Remove(change.temp)
			}
		}
	}()
	for i := range changes {
		if err := ctx.Err(); err != nil {
			return patchSafeFailure(err)
		}
		if err := stagePatch(root, &changes[i]); err != nil {
			return patchSafeFailure(err)
		}
	}
	// Every file is validated before the first visible publication, and each
	// individual source is checked again just before its publication.
	for _, change := range changes {
		if err := recheckPatch(root, change); err != nil {
			return patchSafeFailure(err)
		}
	}
	published := []string{}
	success := patchSuccess(changes)
	for _, change := range changes {
		if err := ctx.Err(); err != nil {
			return patchExecutionFailure(err, published)
		}
		if err := recheckPatch(root, change); err != nil {
			return patchExecutionFailure(err, published)
		}
		if err := ctx.Err(); err != nil {
			return patchExecutionFailure(err, published)
		}
		changed, err := publishPatch(root, change)
		destination := change.path
		if change.destination != "" {
			destination = change.destination
		}
		if changed {
			published = append(published, destination)
		}
		if err != nil {
			return patchExecutionFailure(err, published)
		}
	}
	return success
}
