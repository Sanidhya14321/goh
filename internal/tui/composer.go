package tui

import (
	"strings"
	"unicode"

	"github.com/rivo/uniseg"
)

// graphemeStarts returns the rune offsets where each grapheme cluster of s
// begins, plus len(runes) at the end. Editing moves by these, so one
// Backspace removes 👍🏽 or e + combining accent whole.
func graphemeStarts(s string) []int {
	starts := []int{}
	offset := 0
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		starts = append(starts, offset)
		offset += len(g.Runes())
	}
	return append(starts, offset)
}

// dropLastGrapheme removes the last grapheme cluster of s.
func dropLastGrapheme(s string) string {
	starts := graphemeStarts(s)
	if len(starts) < 2 {
		return s
	}
	return string([]rune(s)[:starts[len(starts)-2]])
}

func (m *model) insertInput(text string) {
	value := []rune(m.input)
	m.inputCursor = min(m.inputCursor, len(value))
	m.input = string(value[:m.inputCursor]) + text + string(value[m.inputCursor:])
	m.inputCursor += len([]rune(text))
}

func (m *model) deleteInput(backward bool) {
	value := []rune(m.input)
	m.inputCursor = min(m.inputCursor, len(value))
	starts := graphemeStarts(m.input)
	from, to := m.inputCursor, m.inputCursor
	if backward {
		if m.inputCursor == 0 {
			return
		}
		for i := len(starts) - 1; i >= 0; i-- {
			if starts[i] < m.inputCursor {
				from = starts[i]
				break
			}
		}
	} else {
		for _, st := range starts {
			if st > m.inputCursor {
				to = st
				break
			}
		}
	}
	if from < to {
		m.input = string(value[:from]) + string(value[to:])
		m.inputCursor = from
	}
}

// moveCursor moves the cursor one grapheme cluster left or right.
func (m *model) moveCursor(right bool) {
	starts := graphemeStarts(m.input)
	cur := min(m.inputCursor, starts[len(starts)-1])
	if right {
		for _, st := range starts {
			if st > cur {
				m.inputCursor = st
				return
			}
		}
		return
	}
	for i := len(starts) - 1; i >= 0; i-- {
		if starts[i] < cur {
			m.inputCursor = starts[i]
			return
		}
	}
}

// visualRows splits the draft into the rows the composer shows at width:
// each logical line wrapped by display cells. Each row is [start, end) in
// runes; end excludes the newline.
func visualRows(value []rune, width int) [][2]int {
	if width < 1 {
		width = 1
	}
	var rows [][2]int
	start, cells := 0, 0
	g := uniseg.NewGraphemes(string(value))
	offset := 0
	for g.Next() {
		n := len(g.Runes())
		if g.Str() == "\n" || g.Str() == "\r\n" {
			rows = append(rows, [2]int{start, offset})
			offset += n
			start, cells = offset, 0
			continue
		}
		w := g.Width()
		if cells+w > width && offset > start {
			rows = append(rows, [2]int{start, offset})
			start, cells = offset, 0
		}
		cells += w
		offset += n
	}
	return append(rows, [2]int{start, offset})
}

// cellsBetween is the display width of value[from:to].
func cellsBetween(value []rune, from, to int) int {
	return uniseg.StringWidth(string(value[from:to]))
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
	width := m.inputWidth
	if width <= 0 {
		width = 1 << 30 // not rendered yet: logical lines only
	}
	rows := visualRows(value, width)
	row := 0
	for i, r := range rows {
		if m.inputCursor >= r[0] && m.inputCursor <= r[1] {
			row = i
			if m.inputCursor < r[1] || i == len(rows)-1 || rows[i+1][0] > r[1] {
				break // at a wrap point the cursor belongs to the next row
			}
		}
	}
	// Up/Down move through displayed rows; history only past the first or
	// last row, so a long wrapped draft is never replaced by mistake.
	if (!down && row == 0) || (down && row == len(rows)-1) {
		if down {
			m.moveHistory(1)
		} else {
			m.moveHistory(-1)
		}
		return
	}
	column := cellsBetween(value, rows[row][0], m.inputCursor)
	var target [2]int
	if down {
		target = rows[row+1]
	} else {
		target = rows[row-1]
	}
	cursor := target[0]
	for _, st := range graphemeStarts(string(value[target[0]:target[1]])) {
		at := target[0] + st
		if at > target[1] || cellsBetween(value, target[0], at) > column {
			break
		}
		cursor = at
	}
	m.inputCursor = cursor
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
	m.inputWidth = width
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
