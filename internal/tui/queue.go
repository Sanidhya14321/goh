package tui

import (
	"fmt"
	"strings"

	"github.com/Sanidhya14321/goh/internal/commands"

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
		m.cancelRun()
	case "shift+tab":
		// Raising access mid-run takes effect at the next request.
		m.cycleAccess()
	case "enter":
		prompt := strings.TrimSpace(m.input)
		switch {
		case prompt == "":
		case commands.IsCommand(prompt):
			m.status = "Commands run after this run finishes; the draft is kept"
		case len(m.queue) >= maxQueued:
			m.status = fmt.Sprintf("Queue full (%d prompts); the draft is kept", maxQueued)
		default:
			m.queue = append(m.queue, m.input)
			m.input, m.inputCursor = "", 0
			m.status = "running"
			m.log += "\n[queued] " + Safe(firstLine(prompt)) + "\n"
		}
	case "up":
		// Take back the newest queued prompt to edit it or delete it.
		if strings.TrimSpace(m.input) == "" && len(m.queue) > 0 {
			m.unqueueLast()
		}
	case "alt+enter", "shift+enter", "ctrl+j":
		m.insertInput("\n")
	case "left":
		m.moveCursor(false)
	case "right":
		m.moveCursor(true)
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
		if typed := strings.TrimSpace(m.input); typed != "" {
			draft += "\n\n" + typed // what the user was typing is kept
		}
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
	if reason := m.holdReason(done); reason == "" {
		m.queueHeld = ""
		return func() tea.Msg { return queuedSubmitMsg{} }
	} else {
		// Nothing is sent without the user seeing why the run stopped. The
		// prompts stay queued, each on its own, and the draft is untouched.
		m.queueHeld = reason
		m.log += fmt.Sprintf("\n[%d queued prompt(s) held because %s. Enter on an empty composer sends the next; ↑ takes the newest back to edit]\n", len(m.queue), reason)
	}
	return nil
}

// holdReason says why queued prompts must wait after a run, or "" when the
// run settled successfully and the next one may be sent. It reads the run's
// own terminal result first and the transport status second.
func (m *model) holdReason(done doneMsg) string {
	status := done.status
	if m.lastRun != nil {
		status = m.lastRun.Status
	}
	switch {
	case m.lastRun.failedVerification():
		return "the configured verification failed"
	case done.err != nil || status != "completed":
		return "the run ended with status " + Safe(status)
	}
	return ""
}

// unqueueLast moves the newest queued prompt back into the empty composer.
func (m *model) unqueueLast() {
	last := m.queue[len(m.queue)-1]
	m.queue = m.queue[:len(m.queue)-1]
	m.input, m.inputCursor = last, len([]rune(last))
	m.status = "Queued prompt returned to the composer"
	m.log += "\n[unqueued] " + Safe(firstLine(strings.TrimSpace(last))) + "\n"
	if len(m.queue) == 0 {
		m.queueHeld = ""
	}
}

// submitQueued sends the next queued prompt through the normal Enter path,
// keeping any draft the user is still writing.
func (m *model) submitQueued() (tea.Model, tea.Cmd) {
	if m.active || len(m.queue) == 0 {
		return m, nil
	}
	if m.overlayOpen() || m.palette || m.find != nil || m.fileCompletion != nil || m.leader {
		// Enter would act on the open panel instead of sending the prompt;
		// hand the queue back rather than lose a prompt into a picker.
		held := strings.Join(m.queue, "\n\n")
		if strings.TrimSpace(m.input) != "" {
			held += "\n\n" + m.input
		}
		m.queue = nil
		m.input, m.inputCursor = "", 0
		m.insertInput(held)
		m.status = "Queued prompts returned to the composer; press Enter to send"
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
