package main

import (
	"bytes"
	"os/exec"
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
	script := b.String() + `COMP_WORDS=(goh se); COMP_CWORD=1; _goh; echo "${COMPREPLY[*]}"
COMP_WORDS=(goh completion z); COMP_CWORD=2; _goh; echo "${COMPREPLY[*]}"`
	out, err := exec.Command(bash, "--norc", "-c", script).CombinedOutput()
	if err != nil {
		t.Fatal(err, string(out))
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 || lines[0] != "sessions setup settings serve" || lines[1] != "zsh" {
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
