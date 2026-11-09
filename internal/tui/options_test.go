package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestOptionPickerFilterAndCancel(t *testing.T) {
	picker := &optionPicker{title: "Select agent", current: "build", width: 24, height: 10,
		options: []Option{{Value: "build", Label: "Build"}, {Value: "plan", Label: "Plan", Description: "Read-only 界"}}}
	picker.Update(tea.PasteMsg{Content: "plan"})
	for _, line := range strings.Split(picker.View().Content, "\n") {
		if ansi.StringWidth(line) > picker.width {
			t.Fatal("picker overflowed a narrow terminal")
		}
	}
	_, cmd := picker.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil || !picker.chosen || picker.selected != "plan" {
		t.Fatal("filtered option was not selected")
	}
	picker.chosen = false
	picker.query = "no match"
	picker.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if picker.chosen {
		t.Fatal("an empty result set selected an option")
	}
	_, cmd = picker.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil || picker.chosen {
		t.Fatal("cancel selected an option")
	}
}
