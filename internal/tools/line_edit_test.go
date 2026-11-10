package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLineEdits(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	orig := "one\ntwo\nthree\nfour\n"
	os.WriteFile(file, []byte(orig), 0o644)
	e, err := New(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	run := func(m map[string]any) Result {
		b, _ := json.Marshal(m)
		return e.Execute(context.Background(), "edit_file", b)
	}
	sum := hash([]byte(orig))
	if r := run(map[string]any{"path": "a.txt", "start_line": 2, "end_line": 3, "new_text": "TWO", "expected_sha256": sum}); !r.Error {
		t.Fatal("line edit accepted while disabled")
	}
	e.LineEdits = true
	for _, bad := range []map[string]any{
		{"path": "a.txt", "start_line": 4, "end_line": 9, "new_text": "x", "expected_sha256": sum},
		{"path": "a.txt", "start_line": 2, "end_line": 2, "old_text": "nope", "new_text": "x", "expected_sha256": sum},
		{"path": "a.txt", "start_line": 2, "new_text": "x", "expected_sha256": "stale"},
	} {
		if r := run(bad); !r.Error {
			t.Fatalf("accepted %v", bad)
		}
	}
	if r := run(map[string]any{"path": "a.txt", "start_line": 2, "end_line": 3, "old_text": "two\nthree", "new_text": "TWO", "expected_sha256": sum}); r.Error {
		t.Fatal(r.Content)
	}
	got, _ := os.ReadFile(file)
	if string(got) != "one\nTWO\nfour\n" {
		t.Fatalf("got %q", got)
	}
	if r := run(map[string]any{"path": "a.txt", "start_line": 1, "new_text": "", "expected_sha256": hash(got)}); r.Error {
		t.Fatal(r.Content)
	}
	got, _ = os.ReadFile(file)
	if string(got) != "TWO\nfour\n" {
		t.Fatalf("delete got %q", got)
	}
	var schema bool
	for _, d := range e.Definitions() {
		if d.Name == "edit_file" {
			_, schema = d.Schema["properties"].(map[string]any)["start_line"]
		}
	}
	for _, d := range Definitions() {
		if d.Name == "edit_file" && strings.Contains(d.Description, "start_line") {
			t.Fatal("default schema exposes line edits")
		}
	}
	if !schema {
		t.Fatal("enabled schema lacks start_line")
	}
}
