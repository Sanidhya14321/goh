package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileMentionsInlineTextFiles(t *testing.T) {
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o600)
	os.WriteFile(filepath.Join(root, "blob.bin"), []byte{0, 1, 2}, 0o600)
	os.Mkdir(filepath.Join(root, "dir"), 0o700)
	os.Symlink("/etc/hosts", filepath.Join(root, "escape.txt"))
	got, files := a.withFileMentions("fix @main.go, see @blob.bin @dir @escape.txt @missing.go @general @shot.png @main.go")
	if strings.Join(files, ",") != "main.go" {
		t.Fatalf("attached %v", files)
	}
	if !strings.Contains(got, "<file path=\"main.go\">\npackage main\n</file>") || strings.Count(got, "<file") != 1 {
		t.Fatalf("%q", got)
	}
	if !strings.HasPrefix(got, "fix @main.go, see @blob.bin") {
		t.Fatal("original prompt text changed")
	}
}

func TestQuotedMentionsAttachPathsWithSpaces(t *testing.T) {
	a, root := testAgent(t)
	os.MkdirAll(filepath.Join(root, "docs"), 0o700)
	os.WriteFile(filepath.Join(root, "docs", "space file.txt"), []byte("SPACED BODY\n"), 0o600)
	os.WriteFile(filepath.Join(root, "plain.txt"), []byte("PLAIN BODY\n"), 0o600)
	got, files := a.withFileMentions(`compare @"docs/space file.txt" with @plain.txt.`)
	if strings.Join(files, ",") != "docs/space file.txt,plain.txt" || !strings.Contains(got, "SPACED BODY") {
		t.Fatalf("files %v", files)
	}
	if m := mentions(`mail a@b.c and (@x.go) @"unterminated`); strings.Join(m, ",") != "x.go,\"unterminated" {
		t.Fatalf("mentions %q", m)
	}
}
