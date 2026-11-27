package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestReplayArguments(t *testing.T) {
	for _, bad := range [][]string{
		{},
		{"--models", "a"},
		{"ID"},
		{"ID", "--models", "a", "extra"},
		{"ID", "--models", "a", "--jobs", "9"},
		{"ID", "--models", "a", "--turns", "-1"},
		{"ID", "--models", "a", "--name", "../x"},
		{"ID", "--models", "a", "--", "--model", "b"},
		{"ID", "--models", "a", "--", "--session=x"},
		{"ID", "--models", "a", "--", "--prompt", "x"},
	} {
		if err := runReplay(bad, &bytes.Buffer{}); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
	if err := runReplay([]string{"ID", "--models", "a", "--session-dir", t.TempDir()}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "session ID") {
		t.Fatal(err)
	}
	if replayLabel([]string{"fix it\nplease"}) != "fix it" || replayLabel([]string{"a", "b", "c"}) != "a (+2 more prompts)" {
		t.Fatal(replayLabel([]string{"a", "b", "c"}))
	}
}
