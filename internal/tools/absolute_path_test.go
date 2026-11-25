package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Absolute paths inside the project work like relative ones in every file
// tool; one outside is refused with the project named.
func TestAbsolutePathsInsideTheProject(t *testing.T) {
	e, root := engine(t)
	os.MkdirAll(filepath.Join(root, "pkg"), 0o755)
	os.WriteFile(filepath.Join(root, "pkg", "a.txt"), []byte("one\n"), 0o644)
	abs := filepath.Join(root, "pkg", "a.txt")
	call := func(name string, args map[string]any) Result {
		b, _ := json.Marshal(args)
		if _, _, err := e.Prepare(name, b); err != nil {
			return Result{Error: true, Content: err.Error()}
		}
		return e.Execute(context.Background(), name, b)
	}
	if r := call("read_file", map[string]any{"path": abs}); r.Error || !strings.Contains(r.Content, "one") {
		t.Fatalf("read_file: %s", r.Content)
	}
	if r := call("write_file", map[string]any{"path": filepath.Join(root, "pkg", "b.txt"), "content": "two\n"}); r.Error {
		t.Fatalf("write_file: %s", r.Content)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "pkg", "b.txt")); string(b) != "two\n" {
		t.Fatalf("b.txt = %q", b)
	}
	patch := "*** Begin Patch\n*** Update File: " + abs + "\n@@\n-one\n+uno\n*** End Patch\n"
	sum := hash([]byte("one\n"))
	if r := call("apply_patch", map[string]any{"patchText": patch, "expected_sha256": map[string]string{abs: sum}}); r.Error {
		t.Fatalf("apply_patch: %s", r.Content)
	}
	if b, _ := os.ReadFile(abs); string(b) != "uno\n" {
		t.Fatalf("a.txt = %q", b)
	}
	if r := call("grep", map[string]any{"pattern": "uno", "path": filepath.Join(root, "pkg")}); r.Error || !strings.Contains(r.Content, "pkg/a.txt") {
		t.Fatalf("grep: %s", r.Content)
	}
	outside := filepath.Join(filepath.Dir(root), "elsewhere.txt")
	if r := call("read_file", map[string]any{"path": outside}); !r.Error || !strings.Contains(r.Content, "outside the project") {
		t.Fatalf("outside path: %s", r.Content)
	}
}

// A path through a symlinked parent of the project (/tmp on macOS) maps too.
func TestAbsolutePathThroughSymlink(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlinks unavailable")
	}
	os.MkdirAll(filepath.Join(real, "proj"), 0o755)
	root, _ := filepath.EvalSymlinks(filepath.Join(real, "proj"))
	for _, p := range []string{filepath.Join(link, "proj", "new", "x.go"), filepath.Join(link, "proj")} {
		rel, err := projectRelative(root, p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if want := map[bool]string{true: ".", false: filepath.Join("new", "x.go")}[strings.HasSuffix(p, "proj")]; rel != want {
			t.Fatalf("%s -> %q, want %q", p, rel, want)
		}
	}
}
