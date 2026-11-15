package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspacesTrustAndRecency(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("APPDATA", home)
	work := t.TempDir()
	a, b := filepath.Join(work, "a"), filepath.Join(work, "b")
	os.MkdirAll(filepath.Join(a, "sub"), 0700)
	os.MkdirAll(b, 0700)
	if IsTrustedWorkspace(a) {
		t.Fatal("trusted before asking")
	}
	if err := OpenWorkspace(a); err != nil {
		t.Fatal(err)
	}
	if !IsTrustedWorkspace(a) || !IsTrustedWorkspace(filepath.Join(a, "sub")) || IsTrustedWorkspace(b) {
		t.Fatal("trust must cover the folder and its subfolders only")
	}
	if err := OpenWorkspace(b); err != nil {
		t.Fatal(err)
	}
	if list := Workspaces(); len(list) != 2 {
		t.Fatal(list)
	}
	if err := ForgetWorkspace(b); err != nil || IsTrustedWorkspace(b) {
		t.Fatal("forget failed", err)
	}
	// The home folder holds the keys and is never recorded.
	if err := OpenWorkspace(home); !errors.Is(err, ErrProjectHoldsKeys) {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(a, "go.mod"), []byte("module x\n"), 0600)
	os.Mkdir(filepath.Join(a, ".git"), 0700)
	if got := ProjectKind(a); got != "git · Go" {
		t.Fatal(got)
	}
}
