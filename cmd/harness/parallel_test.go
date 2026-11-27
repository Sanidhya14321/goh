package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParallelArguments(t *testing.T) {
	tasks := filepath.Join(t.TempDir(), "tasks.txt")
	os.WriteFile(tasks, []byte("# c\n\n"), 0o600)
	for _, bad := range [][]string{
		{},
		{"--file", tasks},
		{"--jobs", "9", "a"},
		{"--name", "../x", "a"},
		{"--format", "yaml", "a"},
		{"a", "--", "--project", "/tmp"},
		{"a", "--", "--prompt=x"},
		{"--models", "a,b", "t", "--", "--model", "c"},
		{"--models", "a,b,c,d,e,f,g,h,i", "t1", "t2"},
		append([]string{}, strings.Split(strings.Repeat("t ", 17), " ")[:17]...),
	} {
		if err := runParallel(bad, &bytes.Buffer{}); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestParallelSummary(t *testing.T) {
	results := []parallelResult{
		{Task: "fix the parser\nmore detail", Branch: "goh/p-1", Worktree: "/r-p-1", Session: "s1", Status: "completed", Changed: 2, Commit: "abc1234", Reply: "Fixed.\nDetails"},
		{Task: "add docs", Model: "m-large", Tokens: 1200, Branch: "goh/p-2", Worktree: "/r-p-2", Status: "needs_permission", Error: "explicit permission required"},
	}
	var b bytes.Buffer
	if failed := writeParallel(&b, results, "text"); failed != 1 {
		t.Fatal(failed)
	}
	out := b.String()
	for _, want := range []string{"1. completed  goh/p-1  2 files", "task: fix the parser\n", "reply: Fixed.\n", "commit: abc1234", "goh chat --session s1", "2. needs_permission  goh/p-2  0 files  0s  m-large  1200 tokens", "error: explicit permission required"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in\n%s", want, out)
		}
	}
	b.Reset()
	writeParallel(&b, results, "json")
	var decoded struct{ Tasks []parallelResult }
	if json.Unmarshal(b.Bytes(), &decoded) != nil || len(decoded.Tasks) != 2 || decoded.Tasks[0].Changed != 2 {
		t.Fatal(b.String())
	}
}
