package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompletionScripts(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		var b bytes.Buffer
		if err := writeCompletion(&b, shell); err != nil {
			t.Fatal(shell, err)
		}
		for _, c := range commandHelp {
			if !strings.Contains(b.String(), c[0]) {
				t.Errorf("%s script misses %s", shell, c[0])
			}
		}
	}
	if writeCompletion(&bytes.Buffer{}, "powershell") == nil {
		t.Fatal("unknown shell accepted")
	}
	if runtime.GOOS == "windows" {
		return
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not installed")
	}
	var b bytes.Buffer
	writeCompletion(&b, "bash")
	fake := t.TempDir()
	os.WriteFile(filepath.Join(fake, "goh"), []byte("#!/bin/sh\nprintf '20261101T1-aaa\\n20261102T1-bbb\\n'\n"), 0o755)
	script := b.String() + `COMP_WORDS=(goh se); COMP_CWORD=1; _goh; echo "${COMPREPLY[*]}"
COMP_WORDS=(goh completion z); COMP_CWORD=2; _goh; echo "${COMPREPLY[*]}"
COMP_WORDS=(goh auth l); COMP_CWORD=2; _goh; echo "${COMPREPLY[*]}"
COMP_WORDS=(goh resume 2026110); COMP_CWORD=2; _goh; echo "${COMPREPLY[*]}"
COMP_WORDS=(goh sessions show 20261102); COMP_CWORD=3; _goh; echo "${COMPREPLY[*]}"`
	cmd := exec.Command(bash, "--norc", "-c", script)
	cmd.Env = append(os.Environ(), "PATH="+fake+":/usr/bin:/bin")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err, string(out))
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 5 || lines[0] != "sessions setup settings serve" || lines[1] != "zsh" || lines[2] != "login logout list" || lines[3] != "20261101T1-aaa 20261102T1-bbb" || lines[4] != "20261102T1-bbb" {
		t.Fatalf("bash completion %q", out)
	}
	if zsh, err := exec.LookPath("zsh"); err == nil {
		var z bytes.Buffer
		writeCompletion(&z, "zsh")
		if out, err := exec.Command(zsh, "-n", "-c", z.String()).CombinedOutput(); err != nil {
			t.Fatal("zsh syntax", err, string(out))
		}
	}
}
