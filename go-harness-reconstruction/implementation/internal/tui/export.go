package tui

import (
	"io"
	"path/filepath"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/session"
)

type exportPanel struct {
	editor                  settingsForm
	backdrop, problem       string
	toolsFocused, withTools bool
	saving, saved           bool
}

type exportDoneMsg struct{ err error }

func (m *model) openExport() {
	if m.a == nil {
		m.status = "Open a session to export its transcript"
		return
	}
	name := "session-" + m.a.Store.ID + ".md"
	m.export = &exportPanel{
		editor: settingsForm{
			fields: []SettingsField{{Label: "Filename", Value: name}},
			cursor: len(name),
		},
		backdrop: m.View().Content,
	}
}

func (m *model) exportUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	p := m.export
	if p.saving {
		return m, nil
	}
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c", "ctrl+d":
			m.export = nil
			return m, nil
		case "enter", "ctrl+s":
			if p.saved {
				m.export = nil
				return m, nil
			}
			name := strings.TrimSpace(p.editor.fields[0].Value)
			if name == "" || strings.ContainsFunc(name, unicode.IsControl) {
				p.problem = "Enter a filename without control characters."
				return m, nil
			}
			store := m.a.Store
			output := name
			if !filepath.IsAbs(output) {
				output = filepath.Join(store.Project, output)
			}
			includeTools := p.withTools
			p.problem, p.saving = "", true
			return m, func() tea.Msg {
				return exportDoneMsg{err: session.WriteExportFile(output, func(w io.Writer) error {
					if exportsHTML(output) {
						return store.WriteHTMLTranscript(w, includeTools)
					}
					return store.WriteTranscript(w, includeTools)
				})}
			}
		case "tab", "shift+tab":
			if !p.saved {
				p.toolsFocused = !p.toolsFocused
			}
			return m, nil
		case "space":
			if p.toolsFocused && !p.saved {
				p.withTools = !p.withTools
				return m, nil
			}
		}
	}
	if !p.toolsFocused && !p.saved {
		p.editor.Update(msg)
	}
	return m, nil
}

func (m *model) exportView(width, height int) tea.View {
	p := m.export
	box := min(76, max(1, width-4))
	left := (width - box) / 2
	padding := strings.Repeat(" ", left)
	top := max(0, (height-9)/2)
	rows := make([]string, top)
	line := func(s, color string) {
		rows = append(rows, padding+cell("  "+s, box, color, panel))
	}
	line("GoHarness  /  Export transcript", foreground)
	line("Markdown, or an HTML page for .html names · Relative paths use the workspace", muted)
	name := []rune(p.editor.fields[0].Value)
	shown := string(name)
	if !p.toolsFocused && !p.saved {
		cursor := p.editor.cursor
		shown = string(name[:cursor]) + "▏" + string(name[cursor:])
	}
	line("Filename", muted)
	line(shown, foreground)
	check, color := "[ ]", muted
	if p.withTools {
		check = "[x]"
	}
	if p.toolsFocused {
		color = accent
	}
	line(check+" Include tool details", color)
	switch {
	case p.saving:
		line("Saving transcript…", accent)
	case p.saved:
		line("Transcript saved", green)
	case p.problem != "":
		line(p.problem, red)
	default:
		line("Exported text may contain private content.", muted)
	}
	if p.saved {
		line("Enter or Esc closes", muted)
	} else if !p.saving {
		line("Tab option · Space toggle · Enter save · Esc cancel", muted)
	}
	return overlayView(p.backdrop, rows, left, top, width, height)
}

// exportsHTML picks the HTML page for .html/.htm names; others get Markdown.
func exportsHTML(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".html" || ext == ".htm"
}
