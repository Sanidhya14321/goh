package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"

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
	needle := strings.ToLower(query)
	m.addEarlierMatches(query, needle)
	lines := strings.Split(m.displayLog(), "\n")
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
	// The transcript keeps streaming and is trimmed from the front, so a
	// remembered line can vanish; drop matches that no longer exist.
	for len(f.matches) > 0 && f.matches[len(f.matches)-1] >= len(lines) {
		f.matches = f.matches[:len(f.matches)-1]
	}
	if len(f.matches) == 0 {
		m.find = nil
		m.status = "Matches scrolled out of the transcript"
		return
	}
	f.current = min(f.current, len(f.matches)-1)
	line := f.matches[f.current]
	width := m.wrappedWidth
	if width <= 0 {
		width = max(1, m.width-4)
	}
	after := len(transcriptRows(strings.Join(lines[line:], "\n"), width, "", false))
	// On a line that wraps, aim at the row holding the match rather than
	// the start of the line, which can be several screens above it.
	if i := strings.Index(strings.ToLower(lines[line]), strings.ToLower(f.query)); i > 0 {
		end := min(len(lines[line]), i+len(f.query))
		after -= max(0, len(transcriptRows(lines[line][:end], width, "", false))-1)
	}
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
		if !m.active {
			m.status = "ready"
		}
	case "up", "shift+enter":
		if len(f.matches) > 0 {
			f.current = (f.current - 1 + len(f.matches)) % len(f.matches)
			m.showMatch()
		}
	case "down", "enter":
		if len(f.matches) > 0 {
			f.current = (f.current + 1) % len(f.matches)
			m.showMatch()
		}
	}
	return m, nil
}

// addEarlierMatches searches the session journal for user and assistant
// turns that match but are no longer in the transcript (it is capped during
// a run and rebuilt from retained history on reopen), and lists them in the
// transcript so /find can reach them. At most 20 are listed; the rest are
// counted.
func (m *model) addEarlierMatches(query, needle string) {
	if m.a == nil || m.a.Store == nil {
		return
	}
	shown := strings.ToLower(m.displayLog())
	var found []string
	extra, turn := 0, 0
	for _, r := range m.a.Store.Records() {
		if r.Type != "message" {
			continue
		}
		var msg core.Message
		if json.Unmarshal(r.Data, &msg) != nil || msg.Automatic || (msg.Role != "user" && msg.Role != "assistant") {
			continue
		}
		if msg.Role == "user" {
			turn++
		}
		lower := strings.ToLower(msg.Content)
		at := strings.Index(lower, needle)
		if at < 0 {
			continue
		}
		// The matching line, as the transcript would show it.
		start := strings.LastIndex(lower[:at], "\n") + 1
		end := strings.IndexByte(lower[at:], '\n')
		line := msg.Content[start:]
		if end >= 0 {
			line = msg.Content[start : at+end]
		}
		if strings.Contains(shown, strings.ToLower(strings.TrimSpace(Safe(line)))) {
			continue // already in the transcript
		}
		if len(found) == 20 {
			extra++
			continue
		}
		who := "You"
		if msg.Role == "assistant" {
			who = "Assistant"
		}
		found = append(found, fmt.Sprintf("  turn %d · %s: %s", turn, who, Safe(oneLine(line, 160))))
	}
	if len(found) == 0 {
		return
	}
	// The heading does not repeat the query, so it is not itself a match.
	block := "\n[find] Earlier matching turns, from the session journal (no longer in the transcript):\n" + strings.Join(found, "\n") + "\n"
	if extra > 0 {
		block += fmt.Sprintf("  … %d more earlier matches not listed; refine the search\n", extra)
	}
	m.log += block
}
