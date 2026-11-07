package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// The timeline shares the composer's bounded, session-local prompt history.
// Opening it performs no journal reads, provider calls, or runtime mutations.
type timelinePanel struct {
	picker       optionPicker
	prompts      []string
	selected     int
	offset       int
	wrapped      []string
	wrappedWidth int
}

func (m *model) openTimeline() {
	p := &timelinePanel{selected: -1}
	p.picker = optionPicker{title: "Timeline · recent prompts", footer: "  ↑/↓ select  Enter preview  Esc chat · up to 200", backdrop: m.View().Content}
	p.prompts = append([]string(nil), m.promptHistory...)
	for i := len(p.prompts) - 1; i >= 0; i-- {
		p.picker.options = append(p.picker.options, Option{Value: strconv.Itoa(i), Label: strings.Join(strings.Fields(Safe(p.prompts[i])), " ")})
	}
	m.timeline = p
}

func (m *model) timelineUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	p := m.timeline
	if v, ok := msg.(tea.KeyPressMsg); ok {
		key := v.String()
		if key == "esc" || key == "ctrl+c" || key == "ctrl+d" {
			if p.selected >= 0 && key == "esc" {
				p.selected = -1
				p.offset = 0
				p.wrapped = nil
			} else {
				m.timeline = nil
			}
			return m, nil
		}
		if p.selected >= 0 {
			switch key {
			case "ctrl+r":
				m.input = Safe(p.prompts[p.selected])
				m.inputCursor = len([]rune(m.input))
				m.historyPosition, m.historyDraft = 0, ""
				m.timeline = nil
			case "up":
				p.offset = max(0, p.offset-1)
			case "down":
				p.offset++
			case "pgup":
				p.offset = max(0, p.offset-8)
			case "pgdown":
				p.offset += 8
			case "home":
				p.offset = 0
			case "end":
				p.offset = len([]rune(p.prompts[p.selected]))
			}
			return m, nil
		}
		if key == "enter" {
			options := p.picker.matches()
			if len(options) > 0 {
				p.selected, _ = strconv.Atoi(options[min(p.picker.index, len(options)-1)].Value)
				p.offset = 0
			}
			return m, nil
		}
	}
	if p.selected < 0 {
		p.picker.Update(msg)
	}
	// Picker quit commands are intentionally not forwarded to the chat program.
	return m, nil
}

func (m *model) timelineView(width, height int) tea.View {
	p := m.timeline
	if p.selected < 0 {
		p.picker.width, p.picker.height = width, height
		return p.picker.View()
	}
	boxWidth := min(82, max(1, width-4))
	left := max(0, (width-boxWidth)/2)
	pad := strings.Repeat(" ", left)
	wrapWidth := max(1, boxWidth-4)
	if p.wrapped == nil || p.wrappedWidth != wrapWidth {
		p.wrapped = terminalRows(Safe(p.prompts[p.selected]), wrapWidth)
		p.wrappedWidth = wrapWidth
	}
	content := p.wrapped
	visible := max(1, min(len(content), height-7))
	p.offset = min(p.offset, max(0, len(content)-visible))
	top := max(0, (height-visible-5)/2)
	rows := make([]string, top)
	rows = append(rows, pad+cell("  GoHarness  /  Prompt preview", boxWidth, foreground, panel),
		pad+cell(fmt.Sprintf("  Prompt %d · %d recent retained · Esc list", p.selected+1, len(p.prompts)), boxWidth, muted, panel))
	for _, line := range content[p.offset : p.offset+visible] {
		rows = append(rows, pad+cell("  "+line, boxWidth, foreground, panel))
	}
	rows = append(rows, pad+cell("  ↑/↓ scroll · PgUp/PgDn · Home/End", boxWidth, muted, panel),
		pad+cell("  Ctrl+R recall to composer · edit before sending", boxWidth, accent, panel))
	return overlayView(p.picker.backdrop, rows, left, top, width, height)
}
