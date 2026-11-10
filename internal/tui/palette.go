package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type paletteEntry struct {
	command, label string
}

var paletteEntries = []paletteEntry{
	{"/provider", "Configure an API key or compatible provider"},
	{"/connect", "Continue with ChatGPT or select API key setup"},
	{"/settings", "Settings and advanced configuration"},
	{"/sessions", "Open a saved session"},
	{"/workspace", "Choose a workspace"},
	{"/compact", "Compact the conversation; /compact focus keeps what you name"},
	{"/todos", "View session tasks"},
	{"/skills", "View available skills"},
	{"/sidebar", "Toggle session sidebar"},
	{"/commands", "List custom commands"},
	{"/models", "Select a configured model profile"},
	{"/agents", "Select Build, Plan or a custom agent"},
	{"/export", "Save the conversation as a Markdown transcript"},
	{"/fork", "Branch the full session or start before an earlier turn"},
	{"/editor", "Edit the prompt in $VISUAL or $EDITOR (Ctrl+X E)"},
	{"/theme", "Choose a color theme (Ctrl+X T)"},
	{"/copy", "Copy any recent message to the clipboard"},
	{"/jobs", "Background jobs: view output or stop"},
	{"/snapshots", "Whole-project changes per run, with restore commands"},
	{"/changes", "Files changed in this session, with diffs"},
	{"/timestamps", "Show or hide when each prompt was sent"},
	{"/effort", "Reasoning effort for this session: /effort low|medium|high|off"},
	{"/thinking", "Show or hide the model's reasoning"},
	{"/details", "Show or hide full tool output"},
	{"/find", "Search the conversation: /find text"},
	{"/rename", "Rename this session: /rename title (empty resets)"},
	{"/undo", "Undo the last turn and restore its checkpointed files"},
	{"/redo", "Return to the turn removed by /undo"},
	{"/new", "Start a new conversation"},
	{"/timeline", "Search recent prompts and recall into the composer"},
	{"/exit", "Quit GoHarness (also /quit, Ctrl+D)"},
	{"/status", "Inspect provider, context, permissions and integrations"},
}

func (m *model) allPaletteEntries() []paletteEntry {
	entries := append([]paletteEntry(nil), paletteEntries...)
	if m.a == nil {
		return entries
	}
	names := make([]string, 0, len(m.a.CommandSet()))
	for name := range m.a.CommandSet() {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		command := "/" + name
		reserved := false
		for _, entry := range paletteEntries {
			if entry.command == command {
				reserved = true
				break
			}
		}
		if !reserved {
			entries = append(entries, paletteEntry{command, m.a.CommandSet()[name].Description})
		}
	}
	return entries
}

func (m *model) paletteEntries() []paletteEntry {
	all := m.allPaletteEntries()
	texts := make([]string, len(all))
	for i, entry := range all {
		texts[i] = entry.command + " " + entry.label
	}
	var entries []paletteEntry
	for _, i := range rankFuzzy(m.paletteQuery, texts) {
		entries = append(entries, all[i])
	}
	return entries
}

func (m *model) paletteKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	entries := m.paletteEntries()
	switch k.String() {
	case "esc", "ctrl+p", "ctrl+k":
		m.palette = false
	case "up":
		m.paletteIndex = max(0, m.paletteIndex-1)
	case "down":
		m.paletteIndex = min(max(0, len(entries)-1), m.paletteIndex+1)
	case "pgup":
		m.paletteIndex = max(0, m.paletteIndex-10)
	case "pgdown":
		m.paletteIndex = min(max(0, len(entries)-1), m.paletteIndex+10)
	case "home":
		m.paletteIndex = 0
	case "end":
		m.paletteIndex = max(0, len(entries)-1)
	case "ctrl+u":
		m.paletteQuery, m.paletteIndex = "", 0
	case "enter":
		if len(entries) > 0 {
			entry := entries[min(m.paletteIndex, len(entries)-1)]
			m.palette = false
			switch entry.command {
			case "/connect", "/provider", "/settings", "/models", "/agents":
				m.navigation = entry.command[1:]
				return m, tea.Quit
			case "/status":
				m.openStatus()
				return m, nil
			case "/timeline":
				m.openTimeline()
				return m, nil
			}
			if entry.command == "/export" {
				m.openExport()
				return m, nil
			}
			if entry.command == "/fork" {
				return m, m.openFork()
			}
			if entry.command == "/find" || entry.command == "/rename" || entry.command == "/effort" {
				m.palette = false
				m.input = entry.command + " "
				m.inputCursor = len([]rune(m.input))
				return m, nil
			}
			if entry.command == "/thinking" {
				m.palette = false
				m.toggleThinking()
				return m, nil
			}
			if entry.command == "/details" {
				m.palette = false
				m.toggleDetails()
				return m, nil
			}
			if entry.command == "/timestamps" {
				m.palette = false
				m.toggleTimestamps()
				return m, nil
			}
			if entry.command == "/changes" {
				m.palette = false
				m.openChanges()
				return m, nil
			}
			if entry.command == "/snapshots" {
				m.palette = false
				m.showSnapshots()
				return m, nil
			}
			if entry.command == "/jobs" {
				m.palette = false
				m.openJobs()
				return m, nil
			}
			if entry.command == "/copy" {
				m.palette = false
				m.openCopyPicker()
				return m, nil
			}
			if entry.command == "/theme" {
				m.palette = false
				m.openThemePicker()
				return m, nil
			}
			if entry.command == "/editor" {
				m.palette = false
				return m, m.openEditor()
			}
			if entry.command == "/undo" || entry.command == "/redo" {
				return m, m.requestRewind(entry.command[1:])
			}
			if entry.command == "/sidebar" {
				m.hideSidebar = !m.hideSidebar
				return m, nil
			}
			m.input = entry.command
			m.inputCursor = len([]rune(m.input))
			return m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		}
	case "backspace":
		value := []rune(m.paletteQuery)
		if len(value) > 0 {
			m.paletteQuery = string(value[:len(value)-1])
			m.paletteIndex = 0
		}
	default:
		if k.Text != "" && len(m.paletteQuery) < 256 {
			m.paletteQuery += Safe(k.Text)
			m.paletteIndex = 0
		}
	}
	return m, nil
}

func (m *model) paletteView(width, height int) tea.View {
	boxWidth := min(76, max(1, width-4))
	padding := strings.Repeat(" ", max(0, (width-boxWidth)/2))
	rows := make([]string, max(0, min(3, height/6)))
	rows = append(rows,
		padding+cell("  GoHarness  /  Commands", boxWidth, foreground, panel),
		padding+cell("  Search  "+m.paletteQuery+" ▏", boxWidth, accent, panel),
		padding+cell("", boxWidth, muted, panel))
	entries := m.paletteEntries()
	visible := max(1, height-len(rows)-4)
	start := max(0, m.paletteIndex-visible+1)
	nameWidth := 0
	for _, entry := range entries {
		nameWidth = max(nameWidth, ansi.StringWidth(entry.command))
	}
	nameWidth = min(nameWidth, 18)
	for i := start; i < min(len(entries), start+visible); i++ {
		entry := entries[i]
		name := ansi.Truncate(entry.command, nameWidth, "…")
		name += strings.Repeat(" ", max(0, nameWidth-ansi.StringWidth(name)))
		if i == m.paletteIndex {
			rows = append(rows, padding+cell("› "+name+"  "+entry.label, boxWidth, background, accent))
			continue
		}
		rows = append(rows, padding+cell("  ", 2, muted, panel)+cell(name, nameWidth, foreground, panel)+cell("  "+entry.label, boxWidth-nameWidth-2, muted, panel))
	}
	if len(entries) == 0 {
		rows = append(rows, padding+cell("  No matching commands", boxWidth, muted, panel))
	}
	rows = append(rows, padding+cell("", boxWidth, foreground, panel),
		padding+cell(fmt.Sprintf("  ↑/↓ select  Enter run  Esc close  ·  %d/%d", min(m.paletteIndex+1, len(entries)), len(entries)), boxWidth, muted, panel))
	return canvas(rows, width, height)
}
