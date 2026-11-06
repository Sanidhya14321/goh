package tui

import (
	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/session"
	"strings"
)

type chooser struct {
	sessions      []session.Info
	index         int
	chosen        bool
	width, height int
}

func (m *chooser) Init() tea.Cmd { return nil }
func (m *chooser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
	}
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "up", "k":
			if m.index > 0 {
				m.index--
			}
		case "down", "j":
			if m.index < len(m.sessions) {
				m.index++
			}
		case "enter":
			m.chosen = true
			return m, tea.Quit
		case "esc", "ctrl+c", "ctrl+d":
			return m, tea.Quit
		}
	}
	return m, nil
}
func (m *chooser) View() tea.View {
	width, height := m.width, m.height
	if width < 1 {
		width = 80
	}
	if height < 1 {
		height = 24
	}
	boxWidth := min(88, max(1, width-4))
	padding := strings.Repeat(" ", max(0, (width-boxWidth)/2))
	rows := []string{"", padding + cell("  GoHarness  /  Sessions", boxWidth, foreground, panel),
		padding + cell("  ↑/↓ select · Enter open · Esc cancel", boxWidth, muted, panel),
		padding + cell("", boxWidth, foreground, panel)}
	items := []string{"Start a new session"}
	for _, s := range m.sessions {
		items = append(items, s.ID+"  "+Safe(s.Project))
	}
	start := 0
	visible := max(1, height-5)
	if m.index >= visible {
		start = m.index - visible + 1
	}
	for i := start; i < len(items) && i < start+visible; i++ {
		mark := "  "
		fg := muted
		if i == m.index {
			mark, fg = "› ", accent
		}
		rows = append(rows, padding+cell(mark+items[i], boxWidth, fg, panel))
	}
	return canvas(rows, width, height)
}
func SelectSession(infos []session.Info) (string, bool, error) {
	m := &chooser{sessions: infos}
	_, e := tea.NewProgram(m).Run()
	if e != nil {
		return "", false, e
	}
	if !m.chosen {
		return "", false, nil
	}
	if m.index == 0 {
		return "", true, nil
	}
	return m.sessions[m.index-1].ID, true, nil
}
