package tui

import (
	"strings"
	"unicode"
)

func (m *model) insertInput(text string) {
	value := []rune(m.input)
	m.inputCursor = min(m.inputCursor, len(value))
	m.input = string(value[:m.inputCursor]) + text + string(value[m.inputCursor:])
	m.inputCursor += len([]rune(text))
}

func (m *model) deleteInput(backward bool) {
	value := []rune(m.input)
	m.inputCursor = min(m.inputCursor, len(value))
	if backward {
		if m.inputCursor == 0 {
			return
		}
		m.inputCursor--
	}
	if m.inputCursor < len(value) {
		m.input = string(value[:m.inputCursor]) + string(value[m.inputCursor+1:])
	}
}

func (m *model) inputLineEnd(end bool) {
	value := []rune(m.input)
	m.inputCursor = min(m.inputCursor, len(value))
	if end {
		for m.inputCursor < len(value) && value[m.inputCursor] != '\n' {
			m.inputCursor++
		}
	} else {
		for m.inputCursor > 0 && value[m.inputCursor-1] != '\n' {
			m.inputCursor--
		}
	}
}

func (m *model) inputUpDown(down bool) {
	value := []rune(m.input)
	m.inputCursor = min(m.inputCursor, len(value))
	start := m.inputCursor
	for start > 0 && value[start-1] != '\n' {
		start--
	}
	column := m.inputCursor - start
	if down {
		end := m.inputCursor
		for end < len(value) && value[end] != '\n' {
			end++
		}
		if end == len(value) {
			m.moveHistory(1)
			return
		}
		nextEnd := end + 1
		for nextEnd < len(value) && value[nextEnd] != '\n' {
			nextEnd++
		}
		m.inputCursor = min(end+1+column, nextEnd)
		return
	}
	if start == 0 {
		m.moveHistory(-1)
		return
	}
	previous := start - 1
	for previous > 0 && value[previous-1] != '\n' {
		previous--
	}
	m.inputCursor = min(previous+column, start-1)
}

func (m *model) moveHistory(step int) {
	if len(m.promptHistory) == 0 {
		return
	}
	if m.historyPosition == 0 {
		if step > 0 {
			return
		}
		m.historyDraft = m.input
	}
	m.historyPosition = min(len(m.promptHistory), max(0, m.historyPosition-step))
	m.input = m.historyDraft
	if m.historyPosition > 0 {
		m.input = m.promptHistory[len(m.promptHistory)-m.historyPosition]
	}
	m.inputCursor = len([]rune(m.input))
}

func (m *model) rememberPrompt(prompt string) {
	if len(m.promptHistory) == 0 || m.promptHistory[len(m.promptHistory)-1] != prompt {
		m.promptHistory = append(m.promptHistory, prompt)
		if len(m.promptHistory) > 200 {
			m.promptHistory = m.promptHistory[len(m.promptHistory)-200:]
		}
	}
	m.historyPosition, m.historyDraft = 0, ""
}

func (m *model) deleteWord() {
	value := []rune(m.input)
	end := min(m.inputCursor, len(value))
	start := end
	for start > 0 && unicode.IsSpace(value[start-1]) {
		start--
	}
	for start > 0 && !unicode.IsSpace(value[start-1]) {
		start--
	}
	m.input, m.inputCursor = string(value[:start])+string(value[end:]), start
}

func (m *model) inputRows(width int) []string {
	value := []rune(m.input)
	cursor := min(m.inputCursor, len(value))
	// Render a bounded window around the cursor while retaining the entire
	// draft. Editing near the beginning never requires painting its full tail.
	start, end := max(0, cursor-2048), min(len(value), cursor+2048)
	before := string(value[start:cursor])
	after := string(value[cursor:end])
	rows := terminalRows(before+"▏"+after, width)
	cursorRow := len(terminalRows(before+"▏", width)) - 1
	first := max(0, cursorRow-2)
	return rows[first:min(len(rows), first+3)]
}

// slashQuery is what the suggestions match against: the typed prefix, or,
// while Tab is cycling through completions, the prefix the cycle began from.
func (m *model) slashQuery() string {
	if m.slashPrefix != "" && strings.HasPrefix(m.input, m.slashPrefix) {
		for _, entry := range m.allPaletteEntries() {
			if entry.command == m.input {
				return m.slashPrefix
			}
		}
	}
	return m.input
}

func (m *model) slashMatches() []paletteEntry {
	query := m.slashQuery()
	if !strings.HasPrefix(query, "/") || strings.ContainsAny(query, " \t\n") {
		return nil
	}
	var matches []paletteEntry
	for _, entry := range m.allPaletteEntries() {
		if strings.HasPrefix(entry.command, query) {
			matches = append(matches, entry)
		}
	}
	return matches
}

func (m *model) hasSlashSuggestions() bool {
	matches := m.slashMatches()
	return len(matches) > 1 || (len(matches) == 1 && m.input != matches[0].command)
}

// completeSlash fills in the first suggestion; pressing Tab again moves to
// the next one, so every listed command is reachable from the keyboard.
func (m *model) completeSlash() bool {
	matches := m.slashMatches()
	if len(matches) == 0 {
		return false
	}
	query := m.slashQuery()
	next := 0
	for i, entry := range matches {
		if entry.command == m.input {
			next = (i + 1) % len(matches)
		}
	}
	m.slashPrefix = query
	m.input = matches[next].command
	m.inputCursor = len([]rune(m.input))
	return true
}
