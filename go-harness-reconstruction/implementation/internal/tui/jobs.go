package tui

import (
	"context"

	"encoding/json"
	agent "example.com/independent-harness/internal/runtime"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type jobStoppedMsg struct{ id, status string }

// openJobs lists this chat's background jobs. Enter shows a job's recent
// output; Ctrl+K stops the selected job (and its process group).
func (m *model) openJobs() {
	if m.a == nil || m.a.Tools == nil {
		m.status = "No background jobs"
		return
	}
	jobs := m.a.Tools.Jobs()
	if len(jobs) == 0 {
		m.status = "No background jobs. The agent can start one with job_start."
		return
	}
	var options []Option
	for i := len(jobs) - 1; i >= 0; i-- {
		j := jobs[i]
		id, _ := j["id"].(string)
		command, _ := j["command"].(string)
		status, _ := j["status"].(string)
		options = append(options, Option{Value: id, Label: id + "  " + status, Description: oneLine(command, 60)})
	}
	m.jobsPicker = &optionPicker{title: "Background jobs", options: options, footer: "  Enter output  Ctrl+K stop  Esc back · jobs stop when GoHarness exits"}
}

func (m *model) jobsUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c", "ctrl+d":
			m.jobsPicker = nil
			return m, nil
		case "ctrl+k":
			matches := m.jobsPicker.matches()
			if len(matches) == 0 {
				return m, nil
			}
			id := matches[min(m.jobsPicker.index, len(matches)-1)].Value
			m.jobsPicker = nil
			m.status = "Stopping " + id + "…"
			tools := m.a.Tools
			return m, func() tea.Msg {
				b, _ := json.Marshal(map[string]string{"id": id})
				r := tools.Execute(contextBackground(), "job_stop", b)
				status := "stopped"
				if r.Error {
					status = "could not be stopped: " + oneLine(r.Content, 80)
				}
				return jobStoppedMsg{id, status}
			}
		}
	}
	m.jobsPicker.Update(msg)
	if !m.jobsPicker.chosen {
		return m, nil
	}
	id := m.jobsPicker.selected
	m.jobsPicker = nil
	b, _ := json.Marshal(map[string]any{"id": id, "limit": 16384})
	r := m.a.Tools.Execute(contextBackground(), "job_output", b)
	var v struct {
		Command  string `json:"command"`
		Status   string `json:"status"`
		Output   string `json:"output"`
		ExitCode *int   `json:"exit_code"`
	}
	json.Unmarshal([]byte(r.Content), &v)
	header := fmt.Sprintf("%s · %s", id, v.Status)
	if v.ExitCode != nil {
		header += fmt.Sprintf(" · exit %d", *v.ExitCode)
	}
	lines := []string{"# " + header, "$ " + v.Command, ""}
	lines = append(lines, strings.Split(strings.TrimRight(Safe(v.Output), "\n"), "\n")...)
	m.infoPanel = &statusPanel{title: "Job output", lines: lines, backdrop: m.View().Content, offset: 1 << 30}
	return m, nil
}

func contextBackground() context.Context { return context.Background() }

func runningJobs(a *agent.Agent) int {
	if a == nil || a.Tools == nil {
		return 0
	}
	n := 0
	for _, j := range a.Tools.Jobs() {
		if j["status"] == "running" {
			n++
		}
	}
	return n
}

// jobStatusLines summarizes jobs for /status.
func jobStatusLines(a *agent.Agent) []string {
	if a == nil || a.Tools == nil {
		return nil
	}
	jobs := a.Tools.Jobs()
	if len(jobs) == 0 {
		return []string{"No background jobs"}
	}
	var lines []string
	for _, j := range jobs {
		id, _ := j["id"].(string)
		status, _ := j["status"].(string)
		command, _ := j["command"].(string)
		lines = append(lines, id+" · "+status+" · "+oneLine(Safe(command), 60))
	}
	return lines
}
