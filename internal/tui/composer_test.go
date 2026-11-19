package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
)

func TestComposerEditingHistoryAndCompletion(t *testing.T) {
	m := &model{}
	m.insertInput("a界c")
	m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m.insertInput("b")
	m.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	if m.input != "a界b" || m.inputCursor != 3 {
		t.Fatalf("middle insertion/delete changed text: %q", m.input)
	}
	m.deleteInput(true)
	if m.input != "a界" {
		t.Fatal("backspace changed preceding Unicode text")
	}
	m.insertInput("\nnext")
	m.inputUpDown(false)
	if m.inputCursor != 2 {
		t.Fatal("up did not preserve the column on the preceding line")
	}
	m.inputUpDown(true)
	if m.inputCursor != 6 { // 界 is two cells wide: column 3 is after "nex"
		t.Fatal("down did not preserve the display column on the next line", m.inputCursor)
	}
	m.rememberPrompt("previous prompt")
	m.input, m.inputCursor = "unfinished draft", 0
	m.inputUpDown(false)
	if m.input != "previous prompt" {
		t.Fatal("history was not recalled")
	}
	m.inputUpDown(true)
	if m.input != "unfinished draft" {
		t.Fatal("history discarded the draft")
	}
	m.a = &agent.Agent{Config: core.Config{Commands: map[string]core.CommandConfig{"review": {Description: "Review changes"}}}, Store: &session.Store{}}
	m.input, m.inputCursor = "/rev", 4
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.input != "/review" || m.active {
		t.Fatal("Tab failed to complete a configured command")
	}
	m.width, m.height, m.input = 120, 36, "/rev"
	if !strings.Contains(m.View().Content, "Tab complete") {
		t.Fatal("slash suggestion was not displayed")
	}
	m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	m.insertInput("ordinary draft")
	if strings.Contains(m.View().Content, "Tab complete") {
		t.Fatal("slash suggestions remained visible for ordinary text")
	}
	m.input = strings.Repeat("界", 5000)
	m.inputCursor = 1
	rows := m.inputRows(30)
	if len(rows) > 3 || !strings.Contains(strings.Join(rows, ""), "▏") || len([]rune(m.input)) != 5000 {
		t.Fatal("large draft lost its cursor or changed its text")
	}
}

// Review: Backspace removes a whole grapheme cluster (emoji with skin tone,
// letter with combining accent, flags), and the cursor moves by clusters.
func TestComposerGraphemes(t *testing.T) {
	for _, tc := range []struct{ text, want string }{
		{"ok 👍🏽", "ok "},
		{"café", "caf"},
		{"🇯🇵", ""},
		{"日本", "日"},
	} {
		m := &model{}
		m.insertInput(tc.text)
		m.deleteInput(true)
		if m.input != tc.want {
			t.Errorf("%q: got %q want %q", tc.text, m.input, tc.want)
		}
	}
	m := &model{}
	m.insertInput("a👍🏽b")
	m.moveCursor(false)
	m.moveCursor(false)
	if m.inputCursor != 1 {
		t.Fatalf("cursor %d", m.inputCursor)
	}
}

// Review: Up from a visually wrapped single-line draft moves to the row
// above instead of replacing the draft with history.
func TestUpFollowsWrappedRows(t *testing.T) {
	m := &model{promptHistory: []string{"older prompt"}}
	m.insertInput(strings.Repeat("word ", 12)) // 60 cells
	m.inputWidth = 20
	m.inputUpDown(false)
	if m.input != strings.Repeat("word ", 12) {
		t.Fatal("history replaced a wrapped draft")
	}
	if m.inputCursor != 40 {
		t.Fatalf("cursor %d, want 40 (row above, same column)", m.inputCursor)
	}
	m.inputUpDown(false)
	m.inputUpDown(false)
	if m.inputCursor != 0 {
		t.Fatalf("cursor %d on the first row", m.inputCursor)
	}
	m.inputUpDown(false) // on the first row: history now
	if m.input != "older prompt" {
		t.Fatalf("history not reached from the first row: %q", m.input)
	}
	m.inputUpDown(true)
	if m.input != strings.Repeat("word ", 12) {
		t.Fatal("draft not restored")
	}
}
