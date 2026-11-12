package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
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
	e.DenyPath = func(rel string, _ bool) bool { return rel == "sub/deny.txt" }
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

func TestDenyCoversNewFilesAndUnicodeSpellingsOnFoldingFilesystems(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "private"), 0o700)
	composed := "café.txt"    // é as one code point
	decomposed := "café.txt" // e + combining acute
	os.WriteFile(filepath.Join(root, composed), []byte("x"), 0o600)
	e, err := New(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := core.Config{FileRules: []core.ShellRule{{Match: "private/*", Action: "deny"}, {Match: composed, Action: "deny"}}}
	e.DenyPath = func(rel string, fold bool) bool { return c.FileDecisionFold([]string{rel}, fold) == "deny" }
	// A file that does not exist yet, under a folder typed in another case.
	if !e.denied("private/new.txt") {
		t.Fatal("private/new.txt not denied")
	}
	if FoldsNames(root) {
		for _, s := range []string{"PRIVATE/new.txt", "Private/NEW.txt", decomposed, "CAFÉ.TXT"} {
			if !e.denied(s) {
				t.Errorf("%q not denied on a folding filesystem", s)
			}
		}
		if err := e.CheckWritable([]string{"PRIVATE/new.txt"}); err == nil {
			t.Fatal("create under PRIVATE allowed")
		}
	}
	if e.denied("public/new.txt") || e.denied("other.txt") {
		t.Fatal("unrelated path denied")
	}
}

func TestCanonicalRelResolvesExistingPrefix(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "real", "dir"), 0o700)
	os.Symlink("real", filepath.Join(root, "link"))
	if got := CanonicalRel(root, "link/dir/missing/new.txt"); got != "real/dir/missing/new.txt" {
		t.Fatal(got)
	}
	if got := CanonicalRel(root, "nothing/here"); got != "nothing/here" {
		t.Fatal(got)
	}
	os.Symlink(t.TempDir(), filepath.Join(root, "out"))
	if got := CanonicalRel(root, "out/x"); got != "out/x" {
		t.Fatal(got)
	}
}
