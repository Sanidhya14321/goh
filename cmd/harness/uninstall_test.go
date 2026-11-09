package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestUninstallListsThenRemovesOnlyOwnFolders(t *testing.T) {
	home := t.TempDir()
	for _, k := range []string{"HOME", "USERPROFILE"} {
		t.Setenv(k, home)
	}
	for _, k := range []string{"XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(k, filepath.Join(home, "cfg"))
	}
	cfg, _ := core.UserConfigPath()
	state := filepath.Dir(core.Defaults().SessionDir)
	neighbour := filepath.Join(filepath.Dir(filepath.Dir(cfg)), "other-tool")
	for _, d := range []string{filepath.Dir(cfg), core.Defaults().SessionDir, neighbour} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	bin := t.TempDir()
	exe := filepath.Join(bin, "goh")
	os.WriteFile(exe, []byte("x"), 0o755)
	if runtime.GOOS != "windows" {
		os.Symlink(exe, filepath.Join(bin, "harness"))
	}
	got := uninstallTargets(exe, false)
	want := []string{exe, filepath.Dir(cfg), state}
	if runtime.GOOS != "windows" {
		want = []string{exe, filepath.Join(bin, "harness"), filepath.Dir(cfg), state}
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("targets\n%v\nwant\n%v", got, want)
	}
	if k := uninstallTargets(exe, true); len(k) != len(want)-2 {
		t.Fatal("--keep-data still lists data", k)
	}
	if t2 := uninstallTargets(filepath.Join(bin, "python3"), true); len(t2) != 0 {
		t.Fatal("non-goh executable listed", t2)
	}
	var out bytes.Buffer
	if err := runUninstall(nil, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(state); err != nil || !strings.Contains(out.String(), "--yes") {
		t.Fatal("listing removed data or gave no hint", err, out.String())
	}
	if err := runUninstall([]string{"--yes"}, &out); err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{filepath.Dir(cfg), state} {
		if _, err := os.Stat(gone); !os.IsNotExist(err) {
			t.Fatal("not removed", gone)
		}
	}
	if _, err := os.Stat(neighbour); err != nil {
		t.Fatal("removed another tool's folder")
	}
}
