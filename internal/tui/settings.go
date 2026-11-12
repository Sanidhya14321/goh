package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type SettingsField struct {
	Label  string
	Value  string
	Secret bool
	// Choices, when set, makes the field a selector: ←/→ (or Space) cycle
	// through them and typing is ignored.
	Choices []string
	// Help is shown under the field while it is focused.
	Help string
	// ChoiceHelp, for a choice field, explains the selected choice.
	ChoiceHelp map[string]string
}

// help is the text shown under a focused field.
func (f SettingsField) help() string {
	if h := f.ChoiceHelp[f.Value]; h != "" {
		return h
	}
	return f.Help
}

// wordWrap breaks text at spaces so no line exceeds width, where possible.
func wordWrap(text string, width int) []string {
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		line := ""
		for _, word := range strings.Fields(para) {
			if line != "" && len([]rune(line))+1+len([]rune(word)) > width {
				lines = append(lines, line)
				line = ""
			}
			if line != "" {
				line += " "
			}
			line += word
		}
		lines = append(lines, line)
	}
	return lines
}

// cycle moves a choice field by step.
func (f *SettingsField) cycle(step int) {
	if len(f.Choices) == 0 {
		return
	}
	i := 0
	for j, c := range f.Choices {
		if c == f.Value {
			i = j
		}
	}
	f.Value = f.Choices[(i+step+len(f.Choices))%len(f.Choices)]
}

type settingsForm struct {
	title, hint, problem         string
	fields                       []SettingsField
	save                         func([]SettingsField) error
	index, cursor, height, width int
	multiline, saved             bool
	dirty, confirmDiscard        bool
}

func (m *settingsForm) Init() tea.Cmd { return nil }

func (m *settingsForm) insert(text string) {
	f := &m.fields[m.index]
	if len(f.Choices) > 0 {
		return
	}
	if !m.multiline {
		// Keys copied from a browser often carry surrounding whitespace or a
		// trailing newline; a value with an inner line break is refused
		// visibly instead of being dropped without a word.
		text = strings.Trim(text, "\r\n")
		if strings.ContainsAny(text, "\r\n") {
			m.problem = "This field is a single line; the pasted text had line breaks and was not inserted"
			return
		}
		text = Safe(text)
	}
	m.dirty, m.confirmDiscard = true, false
	if len(f.Value)+len(text) > 1<<20 {
		return
	}
	value := []rune(f.Value)
	f.Value = string(value[:m.cursor]) + text + string(value[m.cursor:])
	m.cursor += len([]rune(text))
}

func (m *settingsForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok, handled := wheelAsKey(msg); handled {
		if !ok {
			return m, nil
		}
		msg = key
	}
	if len(m.fields) == 0 {
		if k, ok := msg.(tea.KeyPressMsg); ok && (k.String() == "esc" || k.String() == "ctrl+c") {
			return m, tea.Quit
		}
		return m, nil
	}
	f := &m.fields[m.index]
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() != "esc" {
		m.confirmDiscard = false
	}
	switch k := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = k.Height
		m.width = k.Width
	case tea.PasteMsg:
		content := k.Content
		if !m.multiline {
			content = strings.TrimSpace(content) // pasted keys often carry stray spaces
		}
		m.insert(content)
	case tea.KeyPressMsg:
		switch k.String() {
		case "esc", "ctrl+c":
			if m.dirty && !m.confirmDiscard {
				m.confirmDiscard = true
				m.problem = "Unsaved changes. Press Esc again to discard them, or Ctrl+S to save."
				return m, nil
			}
			return m, tea.Quit
		case "up", "down":
			if !m.multiline {
				step := 1
				if k.String() == "up" {
					step = len(m.fields) - 1
				}
				m.index = (m.index + step) % len(m.fields)
				m.cursor = len([]rune(m.fields[m.index].Value))
			}
		case "ctrl+s":
			if err := m.save(m.fields); err != nil {
				m.problem = err.Error()
				return m, nil
			}
			m.saved = true
			return m, tea.Quit
		case "tab", "shift+tab":
			if m.multiline {
				m.insert("  ")
			} else {
				step := 1
				if k.String() == "shift+tab" {
					step = len(m.fields) - 1
				}
				m.index = (m.index + step) % len(m.fields)
				m.cursor = len([]rune(m.fields[m.index].Value))
			}
		case "enter":
			if m.multiline {
				m.insert("\n")
			} else {
				m.index = (m.index + 1) % len(m.fields)
				m.cursor = len([]rune(m.fields[m.index].Value))
			}
		case "left", "right", "space":
			if len(f.Choices) > 0 {
				step := 1
				if k.String() == "left" {
					step = -1
				}
				f.cycle(step)
				m.dirty = true
				return m, nil
			}
			if k.String() == "space" {
				m.insert(" ")
			} else if k.String() == "left" {
				m.cursor = max(0, m.cursor-1)
			} else {
				m.cursor = min(len([]rune(f.Value)), m.cursor+1)
			}
		case "home":
			m.cursor = 0
		case "end":
			m.cursor = len([]rune(f.Value))
		case "ctrl+u":
			if len(f.Choices) > 0 {
				return m, nil
			}
			f.Value, m.cursor, m.dirty = "", 0, true
		case "backspace", "delete":
			if len(f.Choices) > 0 {
				return m, nil
			}
			m.dirty = true
			value := []rune(f.Value)
			if k.String() == "backspace" && m.cursor > 0 {
				m.cursor--
			} else if k.String() == "backspace" {
				return m, nil
			}
			if m.cursor < len(value) {
				f.Value = string(value[:m.cursor]) + string(value[m.cursor+1:])
			}
		default:
			if k.Text != "" {
				m.insert(k.Text)
			}
		}
	}
	return m, nil
}

func (m *settingsForm) View() tea.View {
	var b strings.Builder
	fmt.Fprintf(&b, "GoHarness • %s\n%s\n\n", m.title, strings.Join(wordWrap(m.hint, m.textWidth()), "\n"))
	for i, f := range m.fields {
		value := []rune(f.Value)
		if f.Secret {
			value = []rune(strings.Repeat("•", len(value)))
		}
		shown := string(value)
		mark := "  "
		if len(f.Choices) > 0 {
			shown = "‹ " + f.Value + " ›"
			if i == m.index {
				shown += "   ←/→ change"
			}
			fmt.Fprintf(&b, "%s%s\n%s\n", map[bool]string{true: "> ", false: "  "}[i == m.index], f.Label, Safe(shown))
			if i == m.index && f.help() != "" {
				for _, line := range wordWrap(Safe(f.help()), m.textWidth()-2) {
					fmt.Fprintf(&b, "  %s\n", line)
				}
			}
			continue
		}
		if i == m.index {
			mark = "> "
			shown = string(value[:m.cursor]) + "▏" + string(value[m.cursor:])
			if m.multiline {
				lines := strings.Split(shown, "\n")
				current := strings.Count(string(value[:m.cursor]), "\n")
				visible := max(2, m.height-8)
				start := max(0, current-visible+1)
				shown = strings.Join(lines[start:min(len(lines), start+visible)], "\n")
			}
		}
		fmt.Fprintf(&b, "%s%s\n%s\n", mark, f.Label, Safe(shown))
		if i == m.index && f.help() != "" {
			for _, line := range wordWrap(Safe(f.help()), m.textWidth()-2) {
				fmt.Fprintf(&b, "  %s\n", line)
			}
		}
	}
	var tail strings.Builder
	if m.problem != "" {
		fmt.Fprintln(&tail, "\n"+Safe(m.problem))
	}
	fmt.Fprint(&tail, "\nCtrl+S save • Esc back • Ctrl+U clear field")
	if !m.multiline {
		fmt.Fprint(&tail, " • Tab next field")
	}
	width, height := m.width, m.height
	if width < 1 || height < 1 {
		return blankView()
	}
	boxWidth := min(82, max(1, width-4))
	padding := strings.Repeat(" ", max(0, (width-boxWidth)/2))
	lines := terminalRows(b.String(), max(1, boxWidth-4))
	footer := terminalRows(tail.String(), max(1, boxWidth-4))
	// On short terminals the error and controls always show, and the fields
	// scroll so the focused one stays visible.
	if room := max(1, height-1-len(footer)); len(lines) > room {
		focus := 0
		for i, line := range lines {
			if strings.HasPrefix(line, "> ") {
				focus = i
				break
			}
		}
		start := min(max(0, focus-room/2), len(lines)-room)
		lines = lines[start : start+room]
	}
	lines = append(lines, footer...)
	rows := []string{""}
	for i, line := range lines {
		fg := foreground
		if i == 0 || strings.HasPrefix(line, "> ") {
			fg = accent
		}
		rows = append(rows, padding+cell("  "+line, boxWidth, fg, panel))
	}
	return canvas(rows, width, height)
}

func EditSettings(title, hint string, fields []SettingsField, multiline bool, save func([]SettingsField) error) (bool, error) {
	if len(fields) == 0 {
		return false, nil
	}
	m := &settingsForm{title: title, hint: hint, fields: fields, cursor: len([]rune(fields[0].Value)), height: 24, width: 80, multiline: multiline, save: save}
	_, err := tea.NewProgram(m).Run()
	return m.saved, err
}

type launcher struct {
	status, action string
	index          int
	width, height  int
}

var launchItems = []struct {
	action, label, shortcut string
}{
	{"chat", "Open chat or resume a session", "c"},
	{"provider", "Providers: add, switch, test, keys and models", "p"},
	{"config", "Edit advanced settings", "s"},
	{"project", "Choose a workspace", "w"},
	{"command", "Run a GoHarness command", "r"},
}

func (m *launcher) Init() tea.Cmd { return nil }

func (m *launcher) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok, handled := wheelAsKey(msg); handled {
		if !ok {
			return m, nil
		}
		msg = key
	}
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
	}
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "up":
			m.index = max(0, m.index-1)
		case "down":
			m.index = min(len(launchItems)-1, m.index+1)
		case "enter":
			m.action = launchItems[m.index].action
			return m, tea.Quit
		case "esc", "ctrl+c", "ctrl+d", "q":
			return m, tea.Quit
		default:
			for _, item := range launchItems {
				if k.String() == item.shortcut {
					m.action = item.action
					return m, tea.Quit
				}
			}
		}
	}
	return m, nil
}

func (m *launcher) View() tea.View {
	width, height := m.width, m.height
	if width < 1 || height < 1 {
		return blankView()
	}
	boxWidth := min(76, max(1, width-4))
	padding := strings.Repeat(" ", max(0, (width-boxWidth)/2))
	rows := []string{"", padding + cell("  GoHarness  /  Settings", boxWidth, foreground, panel),
		padding + cell("", boxWidth, foreground, panel)}
	for i, item := range launchItems {
		mark := "  "
		fg := muted
		if i == m.index {
			mark, fg = "› ", accent
		}
		rows = append(rows, padding+cell(mark+item.label+"  ["+item.shortcut+"]", boxWidth, fg, panel))
	}
	rows = append(rows, padding+cell("", boxWidth, foreground, panel),
		padding+cell("  ↑/↓ select · Enter open · Esc back to chat", boxWidth, muted, panel))
	if m.status != "" {
		rows = append(rows, padding+cell("  "+m.status, boxWidth, accent, panel))
	}
	return canvas(rows, width, height)
}

func LaunchMenu(status string) (string, error) {
	m := &launcher{status: status}
	_, err := tea.NewProgram(m).Run()
	return m.action, err
}

// textWidth is the usable text width inside the form box.
func (m *settingsForm) textWidth() int {
	return max(20, min(82, max(1, m.width-4))-4)
}
