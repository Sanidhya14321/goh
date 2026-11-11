package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDenyCoversOtherSpellings(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "sub"), 0o700)
	os.WriteFile(filepath.Join(root, "sub", "deny.txt"), []byte("x"), 0o600)
	os.Symlink("sub", filepath.Join(root, "alias"))
	e, err := New(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	e.DenyPath = func(rel string) bool { return rel == "sub/deny.txt" }
	spellings := []string{"sub/deny.txt", "alias/deny.txt", "./sub/../sub/deny.txt"}
	if _, err := os.Stat(filepath.Join(root, "SUB", "DENY.TXT")); err == nil {
		spellings = append(spellings, "SUB/DENY.TXT")
	}
	for _, s := range spellings {
		if !e.denied(s) {
			t.Errorf("%s not denied", s)
		}
	}
	if e.denied("sub") || e.denied("missing.txt") {
		t.Fatal("unrelated path denied")
	}
}
