package core

import (
	"os"
	"path/filepath"
	"testing"
)

// Saved levels cover a folder and its subfolders, the deepest wins, saving
// ask removes an entry, and trusting or forgetting folders keeps them.
func TestSavedFolderAccess(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("APPDATA", home)
	t.Setenv("USERPROFILE", home)
	work, _ := filepath.EvalSymlinks(t.TempDir())
	project, web := filepath.Join(work, "app"), filepath.Join(work, "app", "web")
	os.MkdirAll(web, 0o700)
	if err := SaveFolderAccess(project, AccessEdits); err != nil {
		t.Fatal(err)
	}
	if err := SaveFolderAccess(web, AccessFull); err != nil {
		t.Fatal(err)
	}
	if err := OpenWorkspace(project); err != nil { // rewrites the trusted list
		t.Fatal(err)
	}
	rules := SavedAccess()
	at := func(p string) AccessLevel {
		r, ok := AccessFor(rules, p)
		if !ok {
			return AccessAsk
		}
		return r.Level
	}
	if at(filepath.Join(web, "src", "a.js")) != AccessFull || at(filepath.Join(project, "main.go")) != AccessEdits || at(work) != AccessAsk {
		t.Fatalf("levels %v", rules)
	}
	if at(project+"-other") != AccessAsk {
		t.Fatal("a sibling with the same prefix matched")
	}
	if err := SaveFolderAccess(web, AccessAsk); err != nil {
		t.Fatal(err)
	}
	if err := ForgetWorkspace(project); err != nil {
		t.Fatal(err)
	}
	if rules = SavedAccess(); len(rules) != 1 || rules[0].Path != project {
		t.Fatalf("after reset: %v", rules)
	}
	if _, err := ParseAccessLevel("everything"); err == nil {
		t.Fatal("unknown level accepted")
	}
	if AccessFull.Next() != AccessAsk || AccessAsk.Next() != AccessEdits {
		t.Fatal("cycle order")
	}
}
