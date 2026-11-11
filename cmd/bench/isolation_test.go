package main

import (
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/testutil"
)

func TestHostGradingDropsSecrets(t *testing.T) {
	testutil.RequireUnixShell(t) // the benchmark grades with /bin/sh
	t.Setenv("GOH_BENCH_FAKE_TOKEN", "leak")
	t.Setenv("SSH_AUTH_SOCK", "/tmp/agent.sock")
	code, out := csh("", t.TempDir(), "env", 30)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if strings.Contains(out, "GOH_BENCH_FAKE_TOKEN") || strings.Contains(out, "SSH_AUTH_SOCK") {
		t.Fatalf("grading saw host secrets:\n%s", out)
	}
	if !strings.Contains(out, "PATH=") {
		t.Fatal("PATH missing")
	}
}

func TestShortCommitIsAnError(t *testing.T) {
	tk := task{ID: "x", dir: t.TempDir()}
	tk.Git = &struct {
		URL    string `json:"url"`
		Commit string `json:"commit"`
	}{URL: "https://example.invalid/r", Commit: "abc"}
	if _, err := tk.sourceTree(); err == nil || !strings.Contains(err.Error(), "40-character") {
		t.Fatalf("err = %v", err)
	}
}
