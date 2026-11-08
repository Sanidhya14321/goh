package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// findState searches the whole transcript (not only the rendered window), so
// older turns are found too. While active, ↑/↓ (or Enter) move between
// matches and Esc returns to the composer; the draft is never changed.
type findState struct {
	query   string
	matches []int // line indexes in the searched text, oldest first
	current int
}

func (m *model) startFind(query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		m.status = "Usage: /find text"
		return
	}
	lines := strings.Split(m.displayLog(), "\n")
	needle := strings.ToLower(query)
	f := &findState{query: query}
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), needle) {
			f.matches = append(f.matches, i)
		}
	}
	if len(f.matches) == 0 {
		m.status = fmt.Sprintf("No matches for %q", oneLine(query, 40))
		return
	}
	f.current = len(f.matches) - 1 // newest first, like scrolling up
	m.find = f
	m.showMatch()
}

// showMatch scrolls so the current match sits in the upper third of the
// transcript area, measuring wrapped rows from the match to the end.
func (m *model) showMatch() {
	f := m.find
	lines := strings.Split(m.displayLog(), "\n")
	line := f.matches[f.current]
	width := m.wrappedWidth
	if width <= 0 {
		width = max(1, m.width-4)
	}
	after := len(transcriptRows(strings.Join(lines[line:], "\n"), width, "", false))
	h := m.transcriptHeight()
	m.scrollOffset = max(0, after-h+h/3)
	m.status = fmt.Sprintf("Match %d/%d: %s · ↑/↓ next · Esc done", f.current+1, len(f.matches), oneLine(lines[line], 50))
}

func (m *model) findKey(key string) (tea.Model, tea.Cmd) {
	f := m.find
	switch key {
	case "esc", "ctrl+c":
		m.find = nil
		m.scrollOffset = 0
		m.status = "ready"
	case "up", "shift+enter":
		f.current = (f.current - 1 + len(f.matches)) % len(f.matches)
		m.showMatch()
	case "down", "enter":
		f.current = (f.current + 1) % len(f.matches)
		m.showMatch()
	}
	return m, nil
}
