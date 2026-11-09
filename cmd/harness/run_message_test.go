package main

import (
	"strings"
	"testing"
)

func TestRunMessage(t *testing.T) {
	got, err := runMessage([]string{"fix", "the", "tests"}, strings.NewReader("unused"))
	if err != nil || got != "fix the tests" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = runMessage([]string{"-", "review", "this"}, strings.NewReader("diff --git a b\n+x\n"))
	if err != nil || got != "diff --git a b\n+x\n\nreview this" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = runMessage([]string{"explain:", "-"}, strings.NewReader("panic: boom\n"))
	if err != nil || got != "explain:\n\npanic: boom" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := runMessage([]string{"-", "-"}, strings.NewReader("x")); err == nil {
		t.Fatal("two stdin markers accepted")
	}
	if _, err := runMessage([]string{"-"}, strings.NewReader("   \n")); err == nil {
		t.Fatal("empty message accepted")
	}
	if _, err := runMessage([]string{"-"}, strings.NewReader(strings.Repeat("x", 1<<20+1))); err == nil {
		t.Fatal("oversized stdin accepted")
	}
}
