package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/Sanidhya14321/goh/internal/core"
)

// palette holds trusted RGB triples ("r;g;b"). Only these values reach the
// terminal as color sequences; theme names select from this fixed table.
type palette struct {
	background, panel, foreground, muted, accent, blue, green, red string
}

var themes = map[string]palette{
	"dark":          {"10;10;10", "24;24;24", "230;230;230", "132;132;132", "250;178;131", "92;156;245", "127;216;143", "224;108;117"},
	"light":         {"250;250;250", "235;235;235", "28;28;28", "110;110;110", "176;88;24", "28;94;196", "26;127;55", "190;40;50"},
	"high-contrast": {"0;0;0", "28;28;28", "255;255;255", "200;200;200", "255;200;0", "110;180;255", "90;255;120", "255;90;90"},
}

var themeDescriptions = map[string]string{
	"dark":          "Default dark palette",
	"light":         "Dark text on a light background",
	"high-contrast": "Maximum contrast on black",
}

// currentTheme is the applied theme name.
var currentTheme = "dark"

// applyTheme sets the palette. Unknown names keep the current palette and
// report false; configuration validation rejects them earlier.
func applyTheme(name string) bool {
	if name == "" {
		name = "dark"
	}
	p, ok := themes[name]
	if !ok {
		return false
	}
	background, panel, foreground, muted = p.background, p.panel, p.foreground, p.muted
	accent, blue, green, red = p.accent, p.blue, p.green, p.red
	currentTheme = name
	return true
}

func init() {
	core.ThemeNames = func() []string {
		names := make([]string, 0, len(themes))
		for n := range themes {
			names = append(names, n)
		}
		return names
	}
}

// openThemePicker lists themes with the current one marked; moving the
// selection previews nothing until Enter, so Esc always restores the view.
func (m *model) openThemePicker() {
	options := []Option{}
	for _, name := range []string{"dark", "light", "high-contrast"} {
		options = append(options, Option{Value: name, Label: name, Description: themeDescriptions[name]})
	}
	m.themePicker = &optionPicker{title: "Theme", current: currentTheme, options: options, footer: "  Enter apply for this session  Esc back · set \"theme\" in config to keep it"}
}

func (m *model) themeUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c", "ctrl+d":
			m.themePicker = nil
			return m, nil
		}
	}
	m.themePicker.Update(msg)
	if m.themePicker.chosen {
		if applyTheme(m.themePicker.selected) {
			m.status = "Theme: " + currentTheme
			m.wrappedRows = nil // re-render cached rows in the new colors
		}
		m.themePicker = nil
	}
	return m, nil
}
