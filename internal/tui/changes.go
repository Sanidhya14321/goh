package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// openChanges lists this session's file checkpoints; Enter shows the diff.
// Reverting stays with /undo, /timeline revert and `goh changes`, which keep
// conversation and files consistent.
func (m *model) openChanges() {
	if m.a == nil {
		m.status = "No session yet"
		return
	}
	if m.active || m.status == "running" {
		m.status = "Wait for the run to finish before viewing changes"
		return
	}
	checkpoints, err := m.a.Checkpoints()
	if err != nil {
		m.status = oneLine(err.Error(), 120)
		return
	}
	var options []Option
	for _, c := range checkpoints {
		files := strings.Join(c.Files, ", ")
		options = append(options, Option{Value: c.ID, Label: oneLine(Safe(files), 60), Description: c.Status + " · " + c.ID})
	}
	m.changesPicker = &optionPicker{title: "Changed files", options: options, footer: "  Enter diff  Esc back · /undo reverts a turn"}
}

func (m *model) changesUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c", "ctrl+d":
			m.changesPicker = nil
			return m, nil
		}
	}
	m.changesPicker.Update(msg)
	if !m.changesPicker.chosen {
		return m, nil
	}
	id := m.changesPicker.selected
	m.changesPicker = nil
	changes, err := m.a.CheckpointDiff(id)
	if err != nil {
		m.status = oneLine(err.Error(), 120)
		return m, nil
	}
	lines := []string{"# Checkpoint " + id}
	if len(changes) == 0 {
		lines = append(lines, "No differences recorded (the edit may have been rolled back).")
	}
	for _, c := range changes {
		lines = append(lines, "", fmt.Sprintf("# %s · %s", c.Path, c.Status))
		switch {
		case c.Binary:
			lines = append(lines, "(binary file)")
		default:
			lines = append(lines, strings.Split(strings.TrimRight(c.Preview, "\n"), "\n")...)
		}
		if c.Truncated {
			lines = append(lines, "… diff truncated; see `goh changes diff`")
		}
	}
	for i := range lines {
		lines[i] = Safe(lines[i])
	}
	m.infoPanel = &statusPanel{title: "Changes", lines: lines, backdrop: m.View().Content}
	return m, nil
}
