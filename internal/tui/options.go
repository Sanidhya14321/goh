package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type Option struct {
	Value, Label, Description string
}

type optionPicker struct {
	title, current, query string
	options               []Option
	index, width, height  int
	selected              string
	chosen                bool
	backdrop              string
	footer                string
	notice                string // shown above the footer, e.g. why the last step failed
}

func (m *optionPicker) matches() []Option {
	texts := make([]string, len(m.options))
	for i, option := range m.options {
		texts[i] = option.Label + " " + option.Description
	}
	var matches []Option
	for _, i := range rankFuzzy(m.query, texts) {
		matches = append(matches, m.options[i])
	}
	return matches
}

func (m *optionPicker) Init() tea.Cmd { return nil }
func (m *optionPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok, handled := wheelAsKey(msg); handled {
		if !ok {
			return m, nil
		}
		msg = key
	}
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case tea.PasteMsg:
		if len(m.query)+len(v.Content) <= 256 {
			m.query += oneLineQuery(v.Content)
			m.index = 0
		}
	case tea.KeyPressMsg:
		switch v.String() {
		case "esc", "ctrl+c", "ctrl+d":
			return m, tea.Quit
		case "up":
			m.index = max(0, m.index-1)
		case "down":
			m.index = min(max(0, len(m.matches())-1), m.index+1)
		case "pgup":
			m.index = max(0, m.index-10)
		case "pgdown":
			m.index = min(max(0, len(m.matches())-1), m.index+10)
		case "home":
			m.index = 0
		case "end":
			m.index = max(0, len(m.matches())-1)
		case "enter":
			options := m.matches()
			if len(options) > 0 {
				m.selected, m.chosen = options[min(m.index, len(options)-1)].Value, true
				return m, tea.Quit
			}
		case "ctrl+u":
			m.query, m.index = "", 0
		case "backspace":
			value := []rune(m.query)
			if len(value) > 0 {
				m.query = string(value[:len(value)-1])
				m.index = 0
			}
		default:
			if v.Text != "" && len(m.query)+len(v.Text) <= 256 {
				m.query += Safe(v.Text)
				m.index = 0
			}
		}
	}
	return m, nil
}

func (m *optionPicker) View() tea.View {
	width, height := m.width, m.height
	if width < 1 || height < 1 {
		return blankView()
	}
	options := m.matches()
	if height < 7 {
		// Too short for the box: show the highlighted option itself (and the
		// title when it fits), so Enter never applies something unseen.
		var rows []string
		if height >= 2 {
			rows = append(rows, ansi.Truncate(Safe(m.title)+" · ↑/↓ Enter Esc", width, "…"))
		}
		if len(options) == 0 {
			rows = append(rows, ansi.Truncate("No matches", width, "…"))
		} else {
			o := options[min(m.index, len(options)-1)]
			rows = append(rows, cell(ansi.Truncate("› "+Safe(o.Label), width, "…"), width, onAccent, accent))
		}
		return canvas(rows, width, height)
	}
	// Wide enough for the longest option with its description (so they
	// are not cut mid-word), within the terminal and a readable maximum.
	labelWidth, want := 0, 68
	for _, o := range m.options {
		labelWidth = max(labelWidth, ansi.StringWidth(o.Label))
	}
	for _, o := range m.options {
		want = max(want, labelWidth+ansi.StringWidth(o.Description)+16)
	}
	boxWidth := min(min(want, 100), max(1, width-4))
	left := max(0, (width-boxWidth)/2)
	padding := strings.Repeat(" ", left)
	visible := max(1, height-6)
	top := max(0, (height-min(len(options), visible)-5)/2)
	rows := make([]string, top)
	rows = append(rows,
		padding+cell("  GoHarness  /  "+m.title, boxWidth, foreground, panel),
		padding+cell("  Search  "+m.query+" ▏", boxWidth, accent, panel),
		padding+cell("", boxWidth, foreground, panel))
	visible = max(1, height-len(rows)-3)
	start := max(0, m.index-visible+1)
	for i := start; i < min(len(options), start+visible); i++ {
		option := options[i]
		mark, color, bg := "  ", muted, panel
		if i == m.index {
			mark, color, bg = "› ", onAccent, accent
		}
		label := option.Label
		if option.Description != "" {
			// Descriptions line up in one column after the labels.
			label += strings.Repeat(" ", max(0, labelWidth-ansi.StringWidth(option.Label))) + "  " + option.Description
		}
		if option.Value == m.current {
			label += "  ● current"
		}
		rows = append(rows, padding+cell(mark+label, boxWidth, color, bg))
	}
	if len(options) == 0 {
		rows = append(rows, padding+cell("  No matches", boxWidth, muted, panel))
	}
	footer := m.footer
	if footer == "" {
		footer = "  ↑/↓ select  Enter apply  Esc back to chat"
	}
	rows = append(rows, padding+cell("", boxWidth, foreground, panel))
	if m.notice != "" {
		for _, line := range wordWrap(m.notice, boxWidth-4) {
			rows = append(rows, padding+cell("  "+line, boxWidth, red, panel))
		}
	}
	rows = append(rows, padding+cell("  "+fitHints(strings.TrimLeft(footer, " "), boxWidth-2), boxWidth, muted, panel))
	return overlayView(m.backdrop, rows, left, top, width, height)
}

func overlayView(backdrop string, rows []string, left, top, width, height int) tea.View {
	if backdrop == "" {
		return canvas(rows, width, height)
	}
	boxWidth := ansi.StringWidth(rows[top]) - left
	under := strings.Split(ansi.Strip(backdrop), "\n")
	composed := make([]string, height)
	for y := range composed {
		line := ""
		if y < len(under) {
			line = under[y]
		}
		composed[y] = cell(line, width, muted, background)
		if y >= top && y < len(rows) {
			composed[y] = cell(ansi.Cut(line, 0, left), left, muted, background) +
				ansi.Cut(rows[y], left, left+boxWidth) +
				cell(ansi.Cut(line, left+boxWidth, width), width-left-boxWidth, muted, background)
		}
	}
	return canvas(composed, width, height)
}

func SelectOption(title, current string, options []Option, backdrop string) (string, bool, error) {
	return SelectOptionWith(title, current, options, backdrop, "", "")
}

// SelectOptionWith is SelectOption with a notice (shown in red above the
// footer) and a custom footer; empty values use the defaults.
func SelectOptionWith(title, current string, options []Option, backdrop, notice, footer string) (string, bool, error) {
	return selectOption(title, current, current, options, backdrop, notice, footer)
}

// SelectOptionFrom opens the picker on start without marking it current.
func SelectOptionFrom(title, start string, options []Option, backdrop, notice, footer string) (string, bool, error) {
	return selectOption(title, "", start, options, backdrop, notice, footer)
}

func selectOption(title, current, start string, options []Option, backdrop, notice, footer string) (string, bool, error) {
	m := &optionPicker{title: title, current: current, options: options, backdrop: backdrop, width: 80, height: 24, notice: notice, footer: footer}
	for i, option := range options {
		if option.Value == start {
			m.index = i
			break
		}
	}
	err := runProgram(m)
	return m.selected, m.chosen, err
}

// oneLineQuery makes pasted text usable as a single-line search: newlines and
// tabs become spaces and terminal controls are removed.
func oneLineQuery(text string) string {
	return strings.Join(strings.Fields(Safe(text)), " ")
}
