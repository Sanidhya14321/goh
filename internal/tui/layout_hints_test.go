package tui

import "testing"

func TestFitHintsDropsWholeHints(t *testing.T) {
	row := "Enter send  Alt+Enter newline  Ctrl+P commands  Ctrl+D exit  Tab agents"
	if got := fitHints(row, 50); got != "Enter send  Alt+Enter newline  Ctrl+P commands" {
		t.Fatalf("%q", got)
	}
	if got := fitHints("Ctrl+C cancel  ·  Ctrl+D exits after cleanup", 20); got != "Ctrl+C cancel" {
		t.Fatalf("%q", got)
	}
	if got := fitHints(row, 200); got != row {
		t.Fatalf("%q", got)
	}
}
