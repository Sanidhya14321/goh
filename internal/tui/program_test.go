package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func press(text string) tea.KeyPressMsg {
	if text == "enter" {
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	}
	r := []rune(text)[0]
	return tea.KeyPressMsg{Code: r, Text: text}
}

// Keys that arrive after a screen asks to close are replayed, in order, into
// the next screen, and Enter keeps its meaning there.
func TestKeysTypedWhileAScreenClosesReachTheNext(t *testing.T) {
	takeTypeAhead()
	picker := &optionPicker{options: []Option{{Value: "build", Label: "build"}, {Value: "plan", Label: "plan"}}, width: 80, height: 24}
	first := &settler{inner: picker}
	first.Update(press("enter"))
	if !first.closing || !picker.chosen || picker.selected != "build" {
		t.Fatalf("picker did not close on Enter: %+v", picker)
	}
	for _, k := range []string{"p", "l", "enter", "h"} {
		first.Update(press(k))
	}
	if picker.query != "" {
		t.Fatalf("closing picker still took keys: %q", picker.query)
	}
	next := &optionPicker{options: picker.options, width: 80, height: 24}
	second := &settler{inner: next, replay: takeTypeAhead()}
	second.Update(replayKeys{})
	if !next.chosen || next.selected != "plan" {
		t.Fatalf("replayed keys did not choose plan: %+v", next)
	}
	left := takeTypeAhead()
	if len(left) != 1 || left[0].(tea.KeyPressMsg).Text != "h" {
		t.Fatalf("key after the second Enter should carry on: %v", left)
	}
}

// While chat waits for setup, Enter on an empty composer opens setup instead
// of doing nothing.
func TestEnterOnBlockedChatOpensSetup(t *testing.T) {
	m := &model{width: 80, height: 24, blocked: "Chat is waiting for setup.\nNo model provider is connected yet.", keys: keymapFor(nil)}
	if _, cmd := m.Update(press("enter")); !isQuit(cmd) || m.navigation != "connect" {
		t.Fatalf("navigation %q", m.navigation)
	}
}
