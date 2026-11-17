package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// A project reached through a symlink still counts as holding the saved
// keys when the credentials folder has not been created yet.
func TestCredentialBoundaryResolvesSymlinkedHome(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	home := filepath.Join(link, "home") // config path is given through the link
	if err := os.Mkdir(home, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("APPDATA", home)
	path, err := UserConfigPath()
	if err != nil {
		t.Skip(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "credentials")); err == nil {
		t.Fatal("credentials folder should not exist yet")
	}
	// The project is the resolved folder above the home, as projectRoot
	// passes it after EvalSymlinks.
	if err := CheckProjectCredentialBoundary(real); !errors.Is(err, ErrProjectHoldsKeys) {
		t.Fatalf("resolved project holding the home: %v", err)
	}
	if err := CheckProjectCredentialBoundary(t.TempDir()); err != nil {
		t.Fatalf("unrelated project refused: %v", err)
	}
}
