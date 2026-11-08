package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// maxQueued bounds prompts typed while a run is active.
const maxQueued = 5

type queuedSubmitMsg struct{}

// activeKey edits the composer while a run is active. Enter queues the draft
// (as in OpenCode's queued prompts); shortcuts that would open dialogs stay
// unavailable until the run ends. Esc cancels the run.
func (m *model) activeKey(v tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch v.String() {
	case "esc":
		m.cancel()
	case "enter":
		prompt := strings.TrimSpace(m.input)
		switch {
		case prompt == "":
		case strings.HasPrefix(prompt, "/"):
			m.status = "Commands run after this run finishes; the draft is kept"
		case len(m.queue) >= maxQueued:
			m.status = fmt.Sprintf("Queue full (%d prompts); the draft is kept", maxQueued)
		default:
			m.queue = append(m.queue, m.input)
			m.input, m.inputCursor = "", 0
			m.log += "\n[queued] " + Safe(firstLine(prompt)) + "\n"
		}
	case "alt+enter", "shift+enter", "ctrl+j":
		m.insertInput("\n")
	case "left":
		m.inputCursor = max(0, min(m.inputCursor, len([]rune(m.input)))-1)
	case "right":
		m.inputCursor = min(len([]rune(m.input)), m.inputCursor+1)
	case "home", "ctrl+a":
		m.inputLineEnd(false)
	case "end", "ctrl+e":
		m.inputLineEnd(true)
	case "ctrl+u":
		m.input, m.inputCursor = "", 0
	case "ctrl+w":
		m.deleteWord()
	case "backspace":
		m.deleteInput(true)
	case "delete":
		m.deleteInput(false)
	default:
		if v.Text != "" {
			m.insertInput(Safe(v.Text))
		}
	}
	return m, nil
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	if r := []rune(line); len(r) > 80 {
		line = string(r[:80]) + "…"
	}
	return line
}

// afterRun sends the next queued prompt only after a completed run. Any other
// outcome (failure, cancellation, approval or input needed) returns queued
// prompts to the composer so nothing is sent without the user seeing it.
func (m *model) afterRun(done doneMsg) tea.Cmd {
	if m.a != nil && m.a.HandoffPlan != "" && done.err == nil {
		// The user approved plan_exit: continue this session in build mode
		// with the go-ahead ready in the composer (queued prompts kept too).
		m.a.HandoffPlan = ""
		draft := "Carry out the approved plan."
		if len(m.queue) > 0 {
			draft += "\n\n" + strings.Join(m.queue, "\n\n")
			m.queue = nil
		}
		m.input, m.inputCursor = "", 0
		m.insertInput(draft)
		m.navigation = "plan_exit"
		return tea.Quit
	}
	if len(m.queue) == 0 {
		return nil
	}
	if done.status == "completed" && done.err == nil {
		return func() tea.Msg { return queuedSubmitMsg{} }
	}
	held := strings.Join(m.queue, "\n\n")
	if strings.TrimSpace(m.input) != "" {
		held += "\n\n" + m.input
	}
	m.queue = nil
	m.input, m.inputCursor = "", 0
	m.insertInput(held)
	m.log += "\n[queued prompts returned to the composer because the run ended with status " + Safe(done.status) + "]\n"
	return nil
}

// submitQueued sends the next queued prompt through the normal Enter path,
// keeping any draft the user is still writing.
func (m *model) submitQueued() (tea.Model, tea.Cmd) {
	if m.active || len(m.queue) == 0 {
		return m, nil
	}
	draft, cursor := m.input, m.inputCursor
	next := m.queue[0]
	m.queue = m.queue[1:]
	m.input, m.inputCursor = next, len([]rune(next))
	model, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.input, m.inputCursor = draft, cursor
	return model, cmd
}
