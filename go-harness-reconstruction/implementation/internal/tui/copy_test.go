package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/core"
	agent "example.com/independent-harness/internal/runtime"
)

func TestCopyPickerCopiesChosenMessage(t *testing.T) {
	a := &agent.Agent{History: []core.Message{
		{Role: "system", Content: "SYSTEM PROMPT"},
		{Role: "user", Content: "first question"},
		{Role: "assistant", Content: "first **answer**"},
		{Role: "tool", Content: "TOOL OUTPUT"},
		{Role: "user", Content: "second question"},
		{Role: "assistant", Content: ""},
	}}
	m := &model{a: a, width: 80, height: 24, input: "draft", inputCursor: 5}
	m.openCopyPicker()
	if m.copyPicker == nil || len(m.copyPicker.options) != 3 {
		t.Fatal("expected three copyable messages", m.copyPicker)
	}
	if !strings.HasPrefix(m.copyPicker.options[0].Label, "You: second") {
		t.Fatal("newest first", m.copyPicker.options[0].Label)
	}
	for _, o := range m.copyPicker.options {
		if strings.Contains(o.Label, "SYSTEM") || strings.Contains(o.Label, "TOOL") {
			t.Fatal("system or tool content offered", o.Label)
		}
	}
	for _, r := range "answer" {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil || m.copyPicker != nil || !strings.Contains(m.status, "Copied message (16") {
		t.Fatal("copy did not happen", m.status)
	}
	if m.input != "draft" || m.inputCursor != 5 {
		t.Fatal("copy changed the draft")
	}
	m.openCopyPicker()
	m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.copyPicker != nil {
		t.Fatal("esc did not close")
	}
}

func TestCopyPickerWithoutMessages(t *testing.T) {
	m := &model{width: 80, height: 24}
	m.openCopyPicker()
	if m.copyPicker != nil || m.status == "" {
		t.Fatal("expected an explanation, not a picker")
	}
}
