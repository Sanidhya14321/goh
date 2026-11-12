package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func TestProviderManagerKeys(t *testing.T) {
	rows := []ProviderRow{{Name: "a", Title: "A"}, {Name: "b", Title: "B", Active: true}}
	for key, want := range map[rune]string{'m': ProviderModel, 'k': ProviderKey, 't': ProviderTest, 'e': ProviderEdit, 'a': ProviderAdd} {
		m := &providerManager{rows: rows, index: 1}
		m.Update(runeKey(key))
		if m.action != want || (want != ProviderAdd && m.name != "b") {
			t.Fatalf("%c: %q %q", key, m.action, m.name)
		}
	}
	m := &providerManager{rows: rows, index: 1}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.action != ProviderUse || m.name != "b" {
		t.Fatal("Enter did not use the provider")
	}
	// Remove needs a second press, and anything else cancels it.
	m = &providerManager{rows: rows}
	m.Update(runeKey('d'))
	if m.action != "" || !strings.Contains(m.notice, "Press d again") {
		t.Fatal("removed without confirmation")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m.Update(runeKey('d'))
	if m.action != "" {
		t.Fatal("confirmation carried over to another provider")
	}
	m.Update(runeKey('d'))
	if m.action != ProviderRemove || m.name != "b" {
		t.Fatal(m.action, m.name)
	}
	// The Add row only adds; provider actions do nothing there.
	m = &providerManager{rows: rows}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	m.Update(runeKey('m'))
	m.Update(runeKey('d'))
	if m.action != "" {
		t.Fatal("provider action on the Add row")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.action != ProviderAdd {
		t.Fatal("Enter on Add row")
	}
	// With no providers the Add row is selected and the screen says so.
	m = &providerManager{width: 80, height: 24}
	if v := m.View(); !strings.Contains(ansiText(v), "No provider yet") {
		t.Fatal("empty state missing")
	}
}

func ansiText(v tea.View) string { return v.Content }

func TestSettingsChoiceField(t *testing.T) {
	f := &settingsForm{fields: []SettingsField{{Label: "Protocol", Value: "b", Choices: []string{"a", "b", "c"},
		ChoiceHelp: map[string]string{"c": "about c"}}}, width: 80, height: 24}
	f.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	f.Update(runeKey('x'))
	f.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if f.fields[0].Value != "c" || f.fields[0].help() != "about c" {
		t.Fatal(f.fields[0].Value)
	}
	f.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	f.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	f.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if f.fields[0].Value != "b" {
		t.Fatal(f.fields[0].Value)
	}
}

func TestWordWrap(t *testing.T) {
	lines := wordWrap("alpha beta gamma delta", 11)
	if strings.Join(lines, "|") != "alpha beta|gamma delta" {
		t.Fatal(lines)
	}
}
