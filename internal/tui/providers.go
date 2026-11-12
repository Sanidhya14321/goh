package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// ProviderRow is one connected provider in the provider manager.
type ProviderRow struct {
	Name, Title string
	// Detail is the second line: model, host and key state.
	Detail string
	Active bool
}

// Provider manager actions, returned with the provider's name.
const (
	ProviderUse    = "use"
	ProviderAdd    = "add"
	ProviderModel  = "model"
	ProviderKey    = "key"
	ProviderTest   = "test"
	ProviderEdit   = "edit"
	ProviderRemove = "remove"
)

type providerManager struct {
	rows          []ProviderRow
	index         int
	notice        string
	confirmRemove string
	action, name  string
	width, height int
	backdrop      string
}

var providerKeys = map[string]string{"a": ProviderAdd, "m": ProviderModel, "k": ProviderKey, "t": ProviderTest, "e": ProviderEdit}

func (m *providerManager) Init() tea.Cmd { return nil }

// onAdd reports whether the highlighted row is "Add a provider".
func (m *providerManager) onAdd() bool { return m.index >= len(m.rows) }

func (m *providerManager) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok, handled := wheelAsKey(msg); handled {
		if !ok {
			return m, nil
		}
		msg = key
	}
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case tea.KeyPressMsg:
		key := v.String()
		if key != "d" {
			m.confirmRemove = ""
		}
		switch key {
		case "esc", "ctrl+c", "ctrl+d", "q":
			return m, tea.Quit
		case "up":
			m.index = max(0, m.index-1)
		case "down":
			m.index = min(len(m.rows), m.index+1)
		case "home":
			m.index = 0
		case "end":
			m.index = len(m.rows)
		case "enter":
			if m.onAdd() {
				m.action = ProviderAdd
			} else {
				m.action, m.name = ProviderUse, m.rows[m.index].Name
			}
			return m, tea.Quit
		case "d", "delete", "backspace":
			if m.onAdd() {
				return m, nil
			}
			name := m.rows[m.index].Name
			if m.confirmRemove == name {
				m.action, m.name = ProviderRemove, name
				return m, tea.Quit
			}
			m.confirmRemove = name
			m.notice = "Press d again to remove " + name + " and its saved key."
		default:
			action, ok := providerKeys[key]
			if !ok {
				return m, nil
			}
			if action == ProviderAdd {
				m.action = ProviderAdd
				return m, tea.Quit
			}
			if m.onAdd() {
				return m, nil
			}
			m.action, m.name = action, m.rows[m.index].Name
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *providerManager) View() tea.View {
	width, height := m.width, m.height
	if width < 1 || height < 1 {
		return blankView()
	}
	boxWidth := min(78, max(1, width-4))
	left := max(0, (width-boxWidth)/2)
	pad := strings.Repeat(" ", left)
	var body []string
	add := func(text string, fg, bg string) { body = append(body, pad+cell(text, boxWidth, fg, bg)) }
	header := []string{pad + cell("  GoHarness  /  Providers", boxWidth, foreground, panel)}
	if len(m.rows) == 0 {
		header = append(header, pad+cell("  No provider yet. Add one to start chatting.", boxWidth, muted, panel))
	} else {
		header = append(header, pad+cell("  ● marks the provider new chats use.", boxWidth, muted, panel))
	}
	header = append(header, pad+cell("", boxWidth, foreground, panel))
	selectedLine := 0
	for i, r := range m.rows {
		mark, fg, bg, detailFg := "  ", foreground, panel, muted
		if i == m.index {
			mark, fg, bg, detailFg = "› ", background, accent, background
			selectedLine = len(body)
		}
		dot := "  "
		if r.Active {
			dot = "● "
		}
		add(mark+dot+Safe(r.Name)+"  ·  "+Safe(r.Title), fg, bg)
		add("      "+Safe(r.Detail), detailFg, bg)
	}
	mark, fg, bg := "  ", accent, panel
	if m.onAdd() {
		mark, fg, bg = "› ", background, accent
		selectedLine = len(body)
	}
	add(mark+"+ Add a provider", fg, bg)

	footer := []string{pad + cell("", boxWidth, foreground, panel)}
	if m.notice != "" {
		for _, line := range terminalRows(Safe(m.notice), max(1, boxWidth-4)) {
			footer = append(footer, pad+cell("  "+line, boxWidth, accent, panel))
		}
	}
	keys := []string{"  Enter use  ·  m model  ·  k key  ·  t test", "  a add  ·  e edit  ·  d remove  ·  Esc close"}
	if m.onAdd() {
		keys = []string{"  Enter add a provider  ·  Esc close"}
	}
	for _, line := range keys {
		footer = append(footer, pad+cell(line, boxWidth, muted, panel))
	}
	room := max(1, height-len(header)-len(footer))
	start := 0
	if len(body) > room {
		start = min(max(0, selectedLine-room/2), len(body)-room)
		if selectedLine+1 >= start+room {
			start = min(len(body)-room, selectedLine+2-room)
		}
	}
	body = body[start:min(len(body), start+room)]
	rows := append(append(append([]string{}, header...), body...), footer...)
	top := max(0, (height-len(rows))/2)
	rows = append(make([]string, top), rows...)
	return overlayView(m.backdrop, rows, left, top, width, height)
}

// ManageProviders shows the connected providers. It returns the chosen
// action and provider name ("" for add), or action "" when closed.
// selected highlights a provider; notice reports the last action's outcome.
func ManageProviders(rows []ProviderRow, selected, notice, backdrop string) (action, name string, err error) {
	m := &providerManager{rows: rows, notice: notice, backdrop: backdrop, width: 80, height: 24}
	for i, r := range rows {
		if r.Name == selected || selected == "" && r.Active {
			m.index = i
		}
	}
	if len(rows) == 0 {
		m.index = 0
	}
	_, err = tea.NewProgram(m).Run()
	return m.action, m.name, err
}
