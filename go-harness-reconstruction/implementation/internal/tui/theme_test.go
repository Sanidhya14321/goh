package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"example.com/independent-harness/internal/core"
)

func TestThemePickerAppliesTrustedPalette(t *testing.T) {
	t.Cleanup(func() { applyTheme("dark") })
	m := &model{width: 60, height: 12, status: "ready", log: "\nYou: hi\n"}
	before := m.View().Content
	leader(m, 't')
	if m.themePicker == nil {
		t.Fatal("leader t did not open the theme picker")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.themePicker != nil || currentTheme != "dark" {
		t.Fatal("Esc must close without changing the theme")
	}
	m.openThemePicker()
	for _, r := range "light" {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if currentTheme != "light" || m.status != "Theme: light" {
		t.Fatal("theme not applied", currentTheme, m.status)
	}
	// Tests render without color, so check the palette and cache directly.
	if background != themes["light"].background || foreground != themes["light"].foreground {
		t.Fatal("palette not switched", background, foreground)
	}
	if m.wrappedRows != nil {
		t.Fatal("cached rows must be dropped so the next render repaints")
	}
	_ = before
}

func TestThemeConfigValidation(t *testing.T) {
	c := core.Defaults()
	c.Model, c.Endpoint, c.ContextWindow = "m", "http://127.0.0.1:1/v1/chat/completions", 1<<20
	c.Theme = "high-contrast"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Theme = "neon"
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "unknown theme") {
		t.Fatal("unknown theme accepted:", err)
	}
	if applyTheme("neon") || currentTheme != "dark" {
		t.Fatal("unknown theme must not change the palette")
	}
}
