package tui

import (
	"strings"
	"testing"
)

func TestDetailsToggle(t *testing.T) {
	log := "\nYou: run tests\n" + toolResultLines("✓ ", Safe("ok  pkg/a\nok  pkg/b\nFAIL pkg/c")) + toolResultLines("", Safe("\x00Rforged"))
	shown := withDetails(log, true)
	if !strings.Contains(shown, "✓ ok  pkg/a\nok  pkg/b\nFAIL pkg/c") || strings.Contains(shown, "(+2 lines)") || strings.Contains(shown, "\x00") {
		t.Fatalf("%q", shown)
	}
	hidden := withDetails(log, false)
	if !strings.Contains(hidden, "✓ ok pkg/a (+2 lines)") || strings.Contains(hidden, "FAIL pkg/c") || strings.Contains(hidden, "\x00") {
		t.Fatalf("%q", hidden)
	}
	if strings.Count(shown, "forged") != 1 || strings.Count(hidden, "forged") != 1 {
		t.Fatal("content must not forge markers")
	}
	m := &model{log: log, width: 80, height: 24}
	m.startFind("FAIL")
	if m.find == nil || len(m.find.matches) != 1 {
		t.Fatal("find should see each line once")
	}
	m.find = nil
	m.toggleDetails()
	m.startFind("FAIL")
	if m.find != nil {
		t.Fatal("hidden details should not be searchable on screen")
	}
}
