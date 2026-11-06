package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/core"
	agent "example.com/independent-harness/internal/runtime"
	"example.com/independent-harness/internal/session"
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
	if m.inputCursor != 5 {
		t.Fatal("down did not preserve the column on the next line")
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
