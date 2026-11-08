package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/core"
	agent "example.com/independent-harness/internal/runtime"
	"example.com/independent-harness/internal/session"
)

func findModel() *model {
	cfg := core.Defaults()
	cfg.Model = "fixture"
	m := &model{a: &agent.Agent{Config: cfg, Store: &session.Store{ID: "fixture", Project: "/tmp/p"}}, width: 80, height: 24, status: "ready"}
	m.log = "\nYou: where is NeedleOld defined\n"
	for i := 0; i < 400; i++ {
		m.log += fmt.Sprintf("filler line %d\n", i)
	}
	m.log += "\nYou: and needleNew?\n" + strings.Repeat("tail\n", 60)
	return m
}

func screenHas(m *model, s string) bool { return strings.Contains(m.View().Content, s) }

func TestFindScrollsOldMatchesIntoView(t *testing.T) {
	m := findModel()
	m.View()
	if screenHas(m, "NeedleOld") || screenHas(m, "needleNew") {
		t.Fatal("fixture should start with both matches off screen")
	}
	m.input = "/find needle"
	m.inputCursor = len(m.input)
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.find == nil || len(m.find.matches) != 2 || m.input != "" {
		t.Fatal("find did not start", m.status)
	}
	if !screenHas(m, "needleNew") || !strings.Contains(m.status, "Match 2/2") {
		t.Fatal("newest match not shown", m.status)
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !screenHas(m, "NeedleOld") || !strings.Contains(m.status, "Match 1/2") {
		t.Fatal("older match not scrolled into view", m.status, m.scrollOffset)
	}
	if !screenHas(m, "▸ You: where is NeedleOld") || screenHas(m, "▸ filler") {
		t.Fatal("match not marked, or a non-match marked")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if !strings.Contains(m.status, "Match 2/2") {
		t.Fatal("down should wrap to the newest match", m.status)
	}
	m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if m.input != "" {
		t.Fatal("typing during find edited the draft")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.find != nil || m.scrollOffset != 0 {
		t.Fatal("esc should end find at the bottom")
	}
	if screenHas(m, "▸ ") {
		t.Fatal("highlight remained after find ended")
	}
}

func TestFindWithoutMatchesOrQuery(t *testing.T) {
	m := findModel()
	m.startFind("absent")
	if m.find != nil || !strings.Contains(m.status, "No matches") {
		t.Fatal(m.status)
	}
	m.startFind("  ")
	if m.find != nil || !strings.Contains(m.status, "Usage") {
		t.Fatal(m.status)
	}
}
