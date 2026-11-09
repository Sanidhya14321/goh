package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestTimelineSearchPreviewRecallAndDraft(t *testing.T) {
	m := &model{width: 120, height: 36, input: "unfinished 界", inputCursor: 12, blocked: "configure provider"}
	m.rememberPrompt("older\n多行 prompt")
	m.rememberPrompt("newest prompt")
	_, cmd := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	if cmd != nil || m.timeline == nil || m.active {
		t.Fatal("opening timeline started work")
	}
	if m.timeline.picker.options[0].Label != "newest prompt" {
		t.Fatal("timeline not newest first")
	}
	m.Update(tea.PasteMsg{Content: "多行"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.timeline.selected != 0 || !strings.Contains(ansi.Strip(m.View().Content), "多行 prompt") {
		t.Fatal("search did not preview original multiline prompt")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.timeline != nil || m.input != "unfinished 界" || m.inputCursor != 12 {
		t.Fatal("cancel changed draft")
	}
	m.openTimeline()
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if cmd != nil || m.timeline != nil || m.active || m.input != "newest prompt" || m.inputCursor != 13 {
		t.Fatal("recall submitted or damaged prompt")
	}
	if len(m.promptHistory) != 2 {
		t.Fatal("browsing changed history")
	}
}

func TestTimelineBoundsEmptyResultsAndControlSafety(t *testing.T) {
	m := &model{width: 80, height: 24, input: "/timeline"}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.timeline.selected != -1 || !strings.Contains(ansi.Strip(m.View().Content), "No matches") {
		t.Fatal("empty history selected a prompt")
	}
	m.timeline = nil
	for i := 0; i < 205; i++ {
		m.rememberPrompt(strings.Repeat("界\t", i+1) + "\x1b[2J\nlast line")
	}
	m.openTimeline()
	if len(m.timeline.prompts) != 200 {
		t.Fatal("history bound lost")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	for _, size := range [][2]int{{8, 4}, {24, 10}, {60, 18}, {120, 36}} {
		m.width, m.height = size[0], size[1]
		m.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
		text := m.View().Content
		if strings.Contains(text, "\x1b[2J") {
			t.Fatal("untrusted terminal control escaped")
		}
		lines := strings.Split(text, "\n")
		if len(lines) > m.height {
			t.Fatal("height overflow")
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > m.width {
				t.Fatal("width overflow")
			}
		}
	}
	m.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if m.timeline != nil {
		t.Fatal("Ctrl+D did not dismiss")
	}
}
