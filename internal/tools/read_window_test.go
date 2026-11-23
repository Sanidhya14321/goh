package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/testutil"
)

type readResult struct {
	Content   string `json:"content"`
	SHA       string `json:"sha256"`
	Start     int    `json:"start_line"`
	End       int    `json:"end_line"`
	Total     int    `json:"total_lines"`
	Truncated bool   `json:"truncated"`
	Next      int    `json:"next_start_line"`
	Note      string `json:"note"`
}

func readLines(t *testing.T, e *Engine, args string) readResult {
	t.Helper()
	r := e.Execute(context.Background(), "read_file", json.RawMessage(args))
	if r.Error {
		t.Fatalf("read_file %s failed: %s", args, r.Content)
	}
	var out readResult
	if err := json.Unmarshal([]byte(r.Content), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// A file too large for one result: a whole-file read reports its length
// and how to read it; an explicit range returns the whole lines that fit
// and how to continue, and paging from next_start_line covers every line.
func TestReadFileLargeFileWindow(t *testing.T) {
	root := testutil.TempDir(t)
	e, err := New(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for i := 1; i <= 6000; i++ {
		fmt.Fprintf(&b, "line %04d: \"quoted\" text that is long enough to matter\n", i)
	}
	os.WriteFile(filepath.Join(root, "big.go"), []byte(b.String()), 0o644)
	// A whole-file read says how long the file is and how to read it,
	// without filling the conversation with its first screens.
	whole := e.Execute(context.Background(), "read_file", json.RawMessage(`{"path":"big.go"}`))
	if !whole.Error || !strings.Contains(whole.Content, "6000 lines") || !strings.Contains(whole.Content, "start_line") || len(whole.Content) > 300 {
		t.Fatalf("whole-file read of a large file: %s", whole.Content)
	}
	// An explicit range that is too large returns the lines that fit.
	first := readLines(t, e, `{"path":"big.go","start_line":1,"end_line":6000}`)
	if !first.Truncated || first.Start != 1 || first.Total != 6000 || first.Next != first.End+1 || !strings.Contains(first.Note, "start_line=") {
		t.Fatalf("window metadata: %+v", first)
	}
	if len(first.Content) > readWindowBytes || !strings.HasPrefix(first.Content, "line 0001") || !strings.HasSuffix(first.Content, fmt.Sprintf("line %04d: \"quoted\" text that is long enough to matter", first.End)) {
		t.Fatalf("window is not whole lines 1..%d", first.End)
	}
	seen, start := 0, 1
	for start <= 6000 {
		w := readLines(t, e, fmt.Sprintf(`{"path":"big.go","start_line":%d}`, start))
		if w.SHA != first.SHA {
			t.Fatal("window sha256 is not the whole file's")
		}
		seen += w.End - w.Start + 1
		if !w.Truncated {
			break
		}
		start = w.Next
	}
	if seen != 6000 {
		t.Fatalf("paging covered %d lines", seen)
	}
	// An explicit range that fits is returned as asked, without a note.
	small := readLines(t, e, `{"path":"big.go","start_line":10,"end_line":12}`)
	if small.Truncated || small.Content != "line 0010: \"quoted\" text that is long enough to matter\nline 0011: \"quoted\" text that is long enough to matter\nline 0012: \"quoted\" text that is long enough to matter" {
		t.Fatalf("small range: %+v", small)
	}
	// One line longer than a window shows its start and says so.
	os.WriteFile(filepath.Join(root, "min.js"), []byte(strings.Repeat("é", 80<<10)+"\n"), 0o644)
	long := readLines(t, e, `{"path":"min.js","start_line":1}`)
	if !long.Truncated || !strings.Contains(long.Note, "longer than one result") || !strings.HasPrefix(long.Content, "é") || strings.ContainsRune(long.Content, '�') {
		t.Fatalf("long line: %+v", long.Note)
	}
}

// total_lines counts lines as an editor does: a final newline ends the last
// line rather than starting another, and an empty file has none. A whole
// read returns the file exactly, final newline included.
func TestReadFileCountsLinesLikeAnEditor(t *testing.T) {
	root := testutil.TempDir(t)
	e, err := New(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		body  string
		total int
	}{"newline.txt": {"a\nb\n", 2}, "open.txt": {"a\nb", 2}, "empty.txt": {"", 0}, "blank.txt": {"\n", 1}} {
		os.WriteFile(filepath.Join(root, name), []byte(c.body), 0o644)
		r := readLines(t, e, `{"path":"`+name+`"}`)
		if r.Total != c.total || r.Content != c.body {
			t.Errorf("%s: total %d content %q, want %d %q", name, r.Total, r.Content, c.total, c.body)
		}
	}
	if r := readLines(t, e, `{"path":"newline.txt","start_line":2}`); r.Content != "b\n" || r.End != 2 {
		t.Errorf("last line: %+v", r)
	}
	if r := e.Execute(context.Background(), "read_file", json.RawMessage(`{"path":"newline.txt","start_line":3}`)); !r.Error {
		t.Error("a line past the end was read")
	}
}
