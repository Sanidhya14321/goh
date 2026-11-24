package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// python is missing on macOS and many Linux systems where python3 exists.
func TestMissingPythonPointsToPython3(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell messages")
	}
	bin := t.TempDir()
	os.WriteFile(filepath.Join(bin, "python3"), []byte("#!/bin/sh\n"), 0o755)
	t.Setenv("PATH", bin)
	if got := missingProgramHint("/bin/sh: python: command not found\n"); got != "python is not installed here; use python3" {
		t.Fatalf("hint %q", got)
	}
	if got := missingProgramHint("/bin/sh: pip: command not found\n"); got != "" {
		t.Fatalf("pip3 is not installed, yet hinted %q", got)
	}
	os.WriteFile(filepath.Join(bin, "python"), []byte("#!/bin/sh\n"), 0o755)
	if got := missingProgramHint("/bin/sh: python: command not found\n"); got != "" {
		t.Fatalf("python exists, yet hinted %q", got)
	}
	if got := missingProgramHint("FAIL: test_python_path"); strings.Contains(got, "python3") {
		t.Fatal("hinted without a missing command")
	}
}

func TestShellResultCarriesTheHint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell messages")
	}
	bin := t.TempDir()
	os.WriteFile(filepath.Join(bin, "python3"), []byte("#!/bin/sh\n"), 0o755)
	t.Setenv("PATH", bin+":/usr/bin:/bin")
	if _, err := exec.LookPath("python"); err == nil {
		t.Skip("python is installed here")
	}
	e, _ := engine(t)
	r := e.Execute(context.Background(), "shell", json.RawMessage(`{"command":"python -c 1"}`))
	if !strings.Contains(r.Content, "use python3") {
		t.Fatal(r.Content)
	}
}
