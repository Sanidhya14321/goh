package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/Sanidhya14321/goh/internal/core"
)

// The fast width path must agree with ansi for every rune it claims.
func TestTextWidthMatchesAnsi(t *testing.T) {
	for r := rune(0x20); r < 0x2800; r++ {
		if !(r < 0x7f || narrowRune(r)) {
			continue
		}
		s := "a" + string(r) + "b"
		if got, want := textWidth(s), ansi.StringWidth(s); got != want {
			t.Fatalf("%U: textWidth %d, ansi %d", r, got, want)
		}
	}
	for _, s := range []string{"✓ note.txt · 1 line", "▸ Read  a.go", "界界", "e\u0301x", "👍🏽 ok", "\x1b[1mbold\x1b[22m", "a\u2060b\u2061c", "\x1b[38;2;1;2;3m✓ ok\x1b[0m  ", "\x1b]8;;x\x07link", "\x1b[38;2;1;2;3m界\x1b[0m"} {
		if got, want := textWidth(s), ansi.StringWidth(s); got != want {
			t.Fatalf("%q: textWidth %d, ansi %d", s, got, want)
		}
	}
}

func TestStyleMarksAndInlineStyling(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if hasStyleMarks(verbatimMark+"plain") || !hasStyleMarks("x"+codeOn+"y") || !hasStyleMarks(verbatimMark+boldOff) {
		t.Fatal("style mark detection")
	}
	rows := transcriptRows("use `go test` and **care**, then *more*", 80, "", false)
	if rows[0].text != "use go test and care, then more" || !strings.Contains(rows[0].styled, "\x1b[1mcare") {
		t.Fatalf("%+v", rows[0])
	}
	// Model text cannot forge a mark: it is removed before rendering.
	forged := transcriptRows("x"+codeOn+"y", 80, "", false)
	if forged[0].text != "xy" || forged[0].styled != "" {
		t.Fatalf("%+v", forged[0])
	}
	// A code span wrapped across rows stays styled on the next row.
	wrapped := transcriptRows("`aaaa bbbb cccc dddd`", 10, "", false)
	if len(wrapped) < 2 || !strings.HasPrefix(wrapped[1].styled, "\x1b[38;2;"+accent+"m") {
		t.Fatalf("%+v", wrapped)
	}
}

func TestEditDiffLines(t *testing.T) {
	args, _ := json.Marshal(map[string]any{"path": "a.go", "old_text": "keep\nold\nkeep2", "new_text": "keep\nnew\nnewer\nkeep2"})
	diff := editDiffLines("edit_file", args)
	if diff != diffMark+"-old\n"+diffMark+"+new\n"+diffMark+"+newer\n" || diffCounts(diff) != "+2 −1" {
		t.Fatalf("%q", diff)
	}
	long, _ := json.Marshal(map[string]any{"path": "b.go", "content": strings.Repeat("line\n", 30)})
	if d := editDiffLines("write_file", long); strings.Count(d, "\n") != diffRowLimit+1 || !strings.Contains(d, "18 more changed lines") {
		t.Fatalf("%q", d)
	}
	if editDiffLines("read_file", args) != "" {
		t.Fatal("only file changes have a diff")
	}
	// A finished edit shows its lines live and when resumed.
	m := &model{width: 80, height: 24}
	m.Update(core.Event{Type: "tool_started", Data: map[string]any{"id": "c1", "name": "edit_file", "arguments": json.RawMessage(args)}})
	m.Update(core.Event{Type: "tool_finished", Data: map[string]any{"id": "c1", "result": map[string]any{"content": `{"path":"a.go","bytes":20}`}}})
	shown := ansi.Strip(strings.Join(rowTexts(transcriptRows(m.displayLog(), 80, "", false)), "\n"))
	if !strings.Contains(shown, "▸ Edit  a.go") || !strings.Contains(shown, "✓ a.go · +2 −1") || !strings.Contains(shown, "- old") || !strings.Contains(shown, "+ newer") {
		t.Fatalf("%s", shown)
	}
}

func rowTexts(rows []transcriptRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.gutter + " " + r.text
	}
	return out
}
