package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestWheelScrollsTranscriptAndClicksAreIgnored(t *testing.T) {
	m := &model{width: 100, height: 30, log: strings.Repeat("line\n", 200)}
	m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	if m.scrollOffset != 3 {
		t.Fatalf("wheel up: offset %d", m.scrollOffset)
	}
	m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.scrollOffset != 0 {
		t.Fatalf("wheel down clamps at bottom: %d", m.scrollOffset)
	}
	before := m.input
	for _, msg := range []tea.Msg{
		tea.MouseClickMsg{Button: tea.MouseLeft}, tea.MouseClickMsg{Button: tea.MouseRight},
		tea.MouseReleaseMsg{}, tea.MouseMotionMsg{}, tea.FocusMsg{}, tea.BlurMsg{},
	} {
		if _, cmd := m.Update(msg); cmd != nil || m.input != before || m.active {
			t.Fatalf("%T changed state", msg)
		}
	}
}

func TestWheelMovesPaletteSelection(t *testing.T) {
	m := &model{width: 100, height: 30, palette: true}
	m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.paletteIndex != 1 {
		t.Fatalf("palette index %d", m.paletteIndex)
	}
}

func TestFirstFrameBeforeSizeIsBlank(t *testing.T) {
	m := &model{}
	if v := m.View(); v.Content != "" || !v.AltScreen {
		t.Fatalf("unsized first frame: %q alt=%v", v.Content, v.AltScreen)
	}
}

func TestIdleCtrlCClearsDraftThenNeedsDoublePress(t *testing.T) {
	m := &model{width: 100, height: 30, input: "draft", inputCursor: 5}
	ctrlC := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	if _, cmd := m.Update(ctrlC); cmd != nil || m.input != "" {
		t.Fatal("first Ctrl+C should only clear the draft")
	}
	_, cmd := m.Update(ctrlC)
	if !strings.Contains(m.View().Content, "Press Ctrl+C again to exit") {
		t.Fatal("missing exit hint")
	}
	_ = cmd // the hint timer; running it would wait out the window
	if _, cmd = m.Update(ctrlC); cmd == nil {
		t.Fatal("second Ctrl+C should exit")
	} else if _, quit := cmd().(tea.QuitMsg); !quit {
		t.Fatal("second Ctrl+C should quit")
	}
}
