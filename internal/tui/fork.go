package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type forkOptionsMsg struct {
	picker  *optionPicker
	options []Option
}

func (m *model) openFork() tea.Cmd {
	if m.a == nil {
		m.status = "Configure a provider to fork a session"
		return nil
	}
	picker := &optionPicker{
		title:    "Fork session",
		footer:   "  Loading recent turns…  Esc back to chat",
		backdrop: m.View().Content,
	}
	m.fork = picker
	store := m.a.Store
	return func() tea.Msg {
		options := []Option{{Value: "0", Label: "Full session", Description: "Copy all conversation state"}}
		for _, turn := range store.RecentUserTurns(200) {
			prompt := []rune(strings.Join(strings.Fields(Safe(turn.Prompt)), " "))
			label := fmt.Sprintf("Before turn %d · %s", turn.Number, string(prompt[:min(240, len(prompt))]))
			options = append(options, Option{Value: strconv.Itoa(turn.Number), Label: label})
		}
		return forkOptionsMsg{picker: picker, options: options}
	}
}

func (m *model) forkUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c", "ctrl+d":
			m.fork = nil
			return m, nil
		}
	}
	m.fork.Update(msg)
	if m.fork.chosen {
		m.forkBefore, _ = strconv.Atoi(m.fork.selected)
		m.fork = nil
		m.navigation = "fork"
		return m, tea.Quit
	}
	return m, nil
}
