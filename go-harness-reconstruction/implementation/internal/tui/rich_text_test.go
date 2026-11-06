package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTableRenderingAcrossWidths(t *testing.T) {
	source := "| File | Result | Count |\n| :--- | :---: | ---: |\n| parser界.go | pass | 12 |\n| api.go | needs review | 1234 |"
	for _, width := range []int{12, 24, 60, 100} {
		rows := transcriptRows(source, width, "", false)
		var text strings.Builder
		for _, row := range rows {
			if ansi.StringWidth(row.text) > width {
				t.Fatalf("table overflow at width %d: %q", width, row.text)
			}
			text.WriteString(row.text)
		}
		if width == 12 {
			if !strings.Contains(text.String(), "parser") || !strings.Contains(text.String(), "1234") {
				t.Fatal("stacked table lost a value")
			}
		} else {
			columns := make([]string, 3)
			for _, row := range rows {
				if !strings.HasPrefix(row.text, "│") {
					continue
				}
				parts := strings.Split(row.text, "│")
				for i := range columns {
					columns[i] += strings.ReplaceAll(strings.TrimSpace(parts[i+1]), " ", "")
				}
			}
			for i, expected := range []string{"Fileparser界.goapi.go", "Resultpassneedsreview", "Count121234"} {
				if columns[i] != expected {
					t.Fatalf("table column %d lost text: %q", i, columns[i])
				}
			}
		}
	}
	cells, ok := tableCells("a|b\\|")
	if !ok || len(cells) != 2 || cells[1] != "b|" {
		t.Fatal("escaped final pipe discarded a cell", cells)
	}
	cells, ok = tableCells("| \x60a|b\x60 | c |")
	if !ok || len(cells) != 2 || cells[0] != "\x60a|b\x60" {
		t.Fatal("inline code pipe split a cell", cells)
	}
	preview := transcriptRows(source, 100, "", true)
	if preview[1].text != "| :--- | :---: | ---: |" {
		t.Fatal("approval text was converted into a table")
	}
}

func TestCodeHighlightingRetainsLiteralText(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	for _, language := range []string{"go", "json", "javascript", "typescript", "", "unknown"} {
		source := "func main() {\n  /* comment\n     more */ value := \"literal **text**\"\n  return 42\n}"
		highlighted := highlightCode(source, language)
		if ansi.Strip(highlighted) != source {
			t.Fatalf("%s highlighter changed code", language)
		}
		if language == "go" && (!strings.Contains(highlighted, blue) || !strings.Contains(highlighted, green) || !strings.Contains(highlighted, muted)) {
			t.Fatal("keywords, strings, or comments were not highlighted")
		}
	}
	fence := strings.Repeat("\x60", 3)
	m := &model{log: fence + "go\n/* " + strings.Repeat("comment content\n", 1000) + "*/\nreturn 42\n" + fence}
	rows := m.visibleRows(m.log, 70, 8, false)
	foundComment := false
	for _, row := range rows {
		if strings.Contains(row.text, "comment content") && strings.Contains(row.styled, muted) {
			foundComment = true
		}
	}
	if !foundComment {
		t.Fatal("scrolling lost multiline comment highlighting")
	}
	longString := fence + "go\n\"" + strings.Repeat("x", 60) + "\"\n" + fence
	wrapped := transcriptRows(longString, 12, "", false)
	if len(wrapped) < 4 || !strings.Contains(wrapped[2].styled, green) {
		t.Fatal("a wrapped string lost its highlight color")
	}
	unsafe := fence + "go\n// \x1b]52;c;payload\a\n" + fence
	for _, row := range transcriptRows(unsafe, 70, "", false) {
		if strings.Contains(row.styled, "\x1b]") {
			t.Fatal("untrusted terminal sequence reached styled output")
		}
	}
	t.Setenv("NO_COLOR", "1")
	if strings.Contains(highlightCode("return 42", "go"), "\x1b") {
		t.Fatal("NO_COLOR was ignored")
	}
}

func TestRichTranscriptBoundsWhileStreaming(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	fence := strings.Repeat("\x60", 3)
	source := "| File | Status |\n| --- | ---: |\n| 界.go | ready |\n\n" + fence + "go\n/* multiline\ncomment */\nfunc ready() bool {\n return true\n}\n" + fence
	for _, width := range []int{12, 24, 60, 90} {
		for cut := 1; cut <= len(source); cut++ {
			for _, row := range transcriptRows(strings.ToValidUTF8(source[:cut], "�"), width, "", false) {
				if ansi.StringWidth(row.text) > width || ansi.StringWidth(row.styled) > width {
					t.Fatalf("streaming frame overflowed at width %d, cut %d", width, cut)
				}
				if row.styled != "" && ansi.Strip(row.styled) != row.text {
					t.Fatal("styling changed a streamed code row")
				}
			}
		}
	}
}

func BenchmarkStreamingCode(b *testing.B) {
	source := strings.Repeat("\x60", 3) + "go\n" + strings.Repeat("func ready() bool { /* comment */ return true }\n", 2000)
	m := &model{}
	b.ReportAllocs()
	for b.Loop() {
		// Invalidate the rendered source to represent a newly arrived delta.
		m.wrappedRows = nil
		m.visibleRows(source, 82, 24, false)
	}
}
