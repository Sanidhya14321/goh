package tui

import (
	"strings"
	"testing"
)

func TestTranscriptCodeAndReviewPreserveContents(t *testing.T) {
	fence := strings.Repeat("\x60", 3)
	code := "value := \"**literal** \x60text\x60\""
	rows := transcriptRows("# Heading\n"+fence+"go\n"+code+"\n"+fence+"\n- **item**", 100, "", false)
	if rows[0].text != "Heading" || rows[0].fg != accent || rows[2].text != code || rows[2].bg != panel || rows[4].text != "• item" {
		t.Fatalf("unexpected Markdown rendering: %+v", rows)
	}
	preview := fence + "go\n- **literal**\n+ \x60replacement\x60\n" + fence
	var text []string
	for _, row := range transcriptRows(preview, 100, "", true) {
		text = append(text, row.text)
	}
	if strings.Join(text, "\n") != preview {
		t.Fatal("approval diff was interpreted as Markdown")
	}
	m := &model{log: fence + "go\n" + strings.Repeat("literal code\n", 1000), width: 80, height: 24}
	rows = m.visibleRows(m.log, 70, 8, false)
	for _, row := range rows[:len(rows)-1] {
		if row.bg != panel {
			t.Fatal("scrolled code fence lost its panel styling")
		}
	}
	// A final logical line stays visible when the tail window starts inside it.
	m.log = strings.Repeat("x", 10000)
	rows = m.visibleRows(m.log, 70, 8, false)
	if len(rows) != 8 || !strings.Contains(rows[0].text, "x") {
		t.Fatal("long line vanished from the viewport")
	}
}
