package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const maxCopyChoices = 50

// openCopyPicker lists recent user and assistant messages, newest first, so any
// of them can be copied (OSC 52) exactly as journaled, Markdown included.
func (m *model) openCopyPicker() {
	if m.a == nil {
		m.status = "No conversation to copy from yet"
		return
	}
	var options []Option
	for i := len(m.a.History) - 1; i >= 0 && len(options) < maxCopyChoices; i-- {
		msg := m.a.History[i]
		if (msg.Role != "user" && msg.Role != "assistant") || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		who := "You"
		if msg.Role == "assistant" {
			who = "Assistant"
		}
		options = append(options, Option{Value: strconv.Itoa(i), Label: who + ": " + oneLine(msg.Content, 60),
			Description: fmt.Sprintf("%d characters", len([]rune(msg.Content)))})
	}
	if len(options) == 0 {
		m.status = "No messages to copy yet"
		return
	}
	m.copyPicker = &optionPicker{title: "Copy message", options: options, footer: "  Enter copy  type to filter  Esc back"}
}

func (m *model) copyUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c", "ctrl+d":
			m.copyPicker = nil
			return m, nil
		}
	}
	m.copyPicker.Update(msg)
	if !m.copyPicker.chosen {
		return m, nil
	}
	selected := m.copyPicker.selected
	m.copyPicker = nil
	i, err := strconv.Atoi(selected)
	if err != nil || m.a == nil || i < 0 || i >= len(m.a.History) {
		m.status = "That message is no longer available"
		return m, nil
	}
	content := m.a.History[i].Content
	m.status = fmt.Sprintf("Copied message (%d characters)", len([]rune(content)))
	return m, tea.SetClipboard(content)
}
