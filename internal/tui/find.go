package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

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
		if at, _ := foldIndex(line, query); at >= 0 {
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
	if i, end := foldIndex(lines[line], f.query); i > 0 {
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
	shown := m.displayLog()
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
		// Line by line on the original text: case folding never shifts
		// offsets into another string, so multibyte text cannot be cut.
		for _, line := range strings.Split(msg.Content, "\n") {
			at, end := foldIndex(line, query)
			if at < 0 {
				continue
			}
			excerpt := matchExcerpt(line, at, end, 160)
			if _, e := foldIndex(shown, strings.TrimSpace(Safe(line))); e >= 0 && strings.TrimSpace(line) != "" {
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
			found = append(found, fmt.Sprintf("  turn %d · %s: %s", turn, who, Safe(excerpt)))
		}
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

// foldIndex finds query in s by Unicode case folding and returns the byte
// range of the match in s itself (-1, -1 when absent). Lowercasing s and
// searching the copy would give offsets into a string whose byte length
// can differ.
func foldIndex(s, query string) (int, int) {
	n := utf8.RuneCountInString(query)
	if n == 0 {
		return -1, -1
	}
	for i := 0; i < len(s); {
		j, k := i, 0
		for k < n && j < len(s) {
			_, size := utf8.DecodeRuneInString(s[j:])
			j += size
			k++
		}
		if k == n && strings.EqualFold(s[i:j], query) {
			return i, j
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}
	return -1, -1
}

// matchExcerpt is at most limit runes of line, centred on the match
// [at, end) (byte offsets in line), so a match far into a long line stays
// in the excerpt and /find finds it again in the transcript.
func matchExcerpt(line string, at, end, limit int) string {
	r := []rune(line)
	if len(r) <= limit {
		return line
	}
	from := utf8.RuneCountInString(line[:at])
	to := from + utf8.RuneCountInString(line[at:end])
	start := max(0, from-(limit-(to-from))/2)
	stop := min(len(r), start+limit)
	start = max(0, stop-limit)
	out := string(r[start:stop])
	if start > 0 {
		out = "…" + out
	}
	if stop < len(r) {
		out += "…"
	}
	return out
}
