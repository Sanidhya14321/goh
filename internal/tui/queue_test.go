package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func typeText(m *model, text string) {
	for _, r := range text {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func TestQueueWhileActive(t *testing.T) {
	m := &model{width: 80, height: 20, active: true, cancel: func() {}}
	typeText(m, "next step")
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(m.queue) != 1 || m.queue[0] != "next step" || m.input != "" || !strings.Contains(m.log, "[queued] next step") {
		t.Fatal("prompt not queued", m.queue, m.input)
	}
	typeText(m, "/fork")
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(m.queue) != 1 || m.input != "/fork" {
		t.Fatal("slash commands must not be queued and the draft must stay", m.queue, m.input)
	}
	m.input, m.inputCursor = "", 0
	for i := 0; i < maxQueued; i++ {
		typeText(m, "p")
		m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	}
	if len(m.queue) != maxQueued || m.input != "p" || !strings.Contains(m.status, "Queue full") {
		t.Fatal("queue limit not enforced", len(m.queue), m.input, m.status)
	}
	// Dialog shortcuts stay unavailable during a run.
	m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	if m.palette {
		t.Fatal("palette opened during an active run")
	}
}

func TestQueuedPromptsReturnToComposerAfterFailure(t *testing.T) {
	m := &model{width: 80, height: 20, active: true, cancel: func() {}, queue: []string{"one", "two"}}
	typeText(m, "draft")
	m.Update(doneMsg{status: "cancelled"})
	if len(m.queue) != 0 || m.input != "one\n\ntwo\n\ndraft" {
		t.Fatalf("queued prompts must come back as a draft, got %q", m.input)
	}
	if m.active {
		t.Fatal("run still marked active")
	}
}

func TestCompletedRunSchedulesNextQueuedPrompt(t *testing.T) {
	m := &model{width: 80, height: 20, active: true, cancel: func() {}, queue: []string{"one"}}
	_, cmd := m.Update(doneMsg{status: "completed"})
	if cmd == nil {
		t.Fatal("completed run should schedule the queued prompt")
	}
	if _, ok := cmd().(queuedSubmitMsg); !ok {
		t.Fatal("unexpected follow-up message")
	}
}
