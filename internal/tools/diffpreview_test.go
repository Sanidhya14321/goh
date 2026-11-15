package tools

import (
	"strings"
	"testing"
)

// A one-line edit in a long file previews as that line with context, not as
// the whole file twice, and the final newline is not shown as a blank line.
func TestDiffPreviewShowsOnlyTheChange(t *testing.T) {
	var lines []string
	for i := 1; i <= 500; i++ {
		lines = append(lines, "line "+strings.Repeat("x", i%7))
	}
	before := strings.Join(lines, "\n") + "\n"
	lines[249] = "changed"
	after := strings.Join(lines, "\n") + "\n"
	got := diffPreview("f.txt", []byte(before), []byte(after), false)
	want := "--- f.txt\n+++ f.txt\n@@ -247,7 +247,7 @@\n"
	if !strings.HasPrefix(got, want) {
		t.Fatalf("got:\n%s", got)
	}
	if n := strings.Count(got, "\n"); n != 3+7+1 {
		t.Fatalf("preview has %d lines, want 11:\n%s", n, got)
	}
	if !strings.Contains(got, "-"+strings.Split(before, "\n")[249]+"\n+changed\n") {
		t.Fatalf("change missing:\n%s", got)
	}
	small := diffPreview("n.txt", []byte("alpha\n"), []byte("beta\n"), false)
	if small != "--- n.txt\n+++ n.txt\n@@ -1,1 +1,1 @@\n-alpha\n+beta\n" {
		t.Fatalf("small edit:\n%q", small)
	}
	created := diffPreview("c.txt", nil, []byte("a\nb\n"), true)
	if !strings.Contains(created, "(new file)\n@@ -1,0 +1,2 @@\n+a\n+b\n") {
		t.Fatalf("new file:\n%q", created)
	}
	if got := diffPreview("e.txt", []byte("a"), []byte("a\n"), false); !strings.Contains(got, "final newline") {
		t.Fatalf("newline-only change:\n%q", got)
	}
	// Two distant changes are two hunks.
	lines[10], lines[400] = "first", "second"
	if got := diffPreview("f.txt", []byte(before), []byte(strings.Join(lines, "\n")+"\n"), false); strings.Count(got, "@@ -") != 3 {
		t.Fatalf("want three hunks:\n%s", got)
	}
}
