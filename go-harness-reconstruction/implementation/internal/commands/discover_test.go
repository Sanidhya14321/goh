package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverCommandFiles(t *testing.T) {
	p := t.TempDir()
	write(t, filepath.Join(p, ".opencode/command/review.md"), "---\ndescription: \"Review changes\"\nagent: build\nmodel: other/model\n---\n\nReview $ARGUMENTS carefully.\n")
	write(t, filepath.Join(p, ".claude/commands/review.md"), "shadowed")
	write(t, filepath.Join(p, ".claude/commands/plain.md"), "Just a template\r\n")
	write(t, filepath.Join(p, ".harness/commands/compact.md"), "reserved")
	write(t, filepath.Join(p, ".harness/commands/bad name.md"), "x")
	write(t, filepath.Join(p, ".harness/commands/empty.md"), "---\ndescription: d\n---\n")
	write(t, filepath.Join(p, ".harness/commands/open.md"), "---\ndescription: d\nno end")
	write(t, filepath.Join(p, ".harness/commands/notes.txt"), "ignored")
	write(t, filepath.Join(p, ".harness/commands/sub/deep.md"), "ignored")
	write(t, filepath.Join(p, ".harness/commands/big.md"), strings.Repeat("x", MaxTemplate+5000))
	if runtime.GOOS != "windows" {
		write(t, filepath.Join(p, "outside.md"), "secret")
		os.Symlink(filepath.Join(p, "outside.md"), filepath.Join(p, ".harness/commands/link.md"))
	}
	found, skipped := Discover(p)
	if len(found) != 2 {
		t.Fatalf("found %v", found)
	}
	if r := found["review"]; r.Description != "Review changes" || r.Template != "Review $ARGUMENTS carefully." {
		t.Fatalf("%+v", r)
	}
	if found["plain"].Template != "Just a template" {
		t.Fatalf("%q", found["plain"].Template)
	}
	reasons := map[string]string{}
	for _, s := range skipped {
		reasons[filepath.Base(s.Path)] = s.Reason
	}
	for _, name := range []string{"review.md", "compact.md", "bad name.md", "empty.md", "open.md", "big.md"} {
		if reasons[name] == "" {
			t.Fatal("expected skip reason for", name, reasons)
		}
	}
	if _, ok := found["link"]; ok {
		t.Fatal("followed a symlinked command file")
	}
	if err := Validate(found); err != nil {
		t.Fatal(err)
	}
}

func TestConfiguredCommandsWin(t *testing.T) {
	merged := Merge(map[string]core.CommandConfig{"review": {Template: "mine"}},
		map[string]core.CommandConfig{"review": {Template: "repo"}, "extra": {Template: "x"}})
	if merged["review"].Template != "mine" || merged["extra"].Template != "x" {
		t.Fatal(merged)
	}
	files := map[string]core.CommandConfig{}
	for i := 0; i < 300; i++ {
		files[strings.Repeat("a", 1)+string(rune('a'+i%26))+strings.Repeat("b", i/26+1)] = core.CommandConfig{Template: "x"}
	}
	if got := Merge(nil, files); len(got) != 256 || Validate(got) != nil {
		t.Fatal("limit", len(got))
	}
}
