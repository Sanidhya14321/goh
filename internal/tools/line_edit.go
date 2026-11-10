package tools

import (
	"bytes"
	"errors"
	"fmt"
)

// lineEdit replaces lines a.Start..a.End (1-based, inclusive) with a.New.
// Line numbers are anchored by the caller's whole-file expected_sha256, which
// was already checked, so they refer to exactly the content the model read.
// If old_text is also given it must equal the replaced lines (a second anchor).
func lineEdit(before []byte, a args, enabled bool) ([]byte, error) {
	if !enabled {
		return nil, errors.New("line-range edits are not enabled; use old_text")
	}
	if a.All {
		return nil, errors.New("replace_all cannot be combined with a line range")
	}
	if a.End == 0 {
		a.End = a.Start
	}
	lines := bytes.SplitAfter(before, []byte("\n"))
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	if a.Start < 1 || a.End < a.Start || a.End > len(lines) {
		return nil, fmt.Errorf("line range %d-%d is outside the file's %d lines", a.Start, a.End, len(lines))
	}
	old := bytes.Join(lines[a.Start-1:a.End], nil)
	if a.Old != "" && string(bytes.TrimRight(old, "\n")) != stringsTrimRight(a.Old) {
		return nil, fmt.Errorf("old_text does not match lines %d-%d; read the file again", a.Start, a.End)
	}
	repl := []byte(a.New)
	if len(repl) > 0 && !bytes.HasSuffix(repl, []byte("\n")) && bytes.HasSuffix(old, []byte("\n")) {
		repl = append(repl, '\n') // keep the line structure
	}
	var out bytes.Buffer
	out.Write(bytes.Join(lines[:a.Start-1], nil))
	out.Write(repl)
	out.Write(bytes.Join(lines[a.End:], nil))
	if out.Len() > MaxFile {
		return nil, errors.New("result exceeds 4MiB")
	}
	return out.Bytes(), nil
}

func stringsTrimRight(s string) string { return string(bytes.TrimRight([]byte(s), "\n")) }
