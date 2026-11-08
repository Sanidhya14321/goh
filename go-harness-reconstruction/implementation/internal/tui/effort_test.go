package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"example.com/independent-harness/internal/core"
	agent "example.com/independent-harness/internal/runtime"
	"example.com/independent-harness/internal/session"
)

func TestEffortCommand(t *testing.T) {
	cfg := core.Config{Protocol: "openai_responses", Model: "m"}
	m := &model{a: &agent.Agent{Config: cfg, Store: &session.Store{ID: "fixture"}}, width: 80, height: 24, status: "ready"}
	m.input = "/effort HIGH"
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.a.Config.ReasoningEffort != "high" {
		t.Fatal(m.a.Config.ReasoningEffort, m.log)
	}
	m.input = "/effort bogus"
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.a.Config.ReasoningEffort != "high" || !strings.Contains(m.log, "must be") {
		t.Fatal(m.log)
	}
	m.active = true
	m.input = "/effort off"
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.a.Config.ReasoningEffort != "high" {
		t.Fatal("changed during a run")
	}
	m.active = false
	m.input = "/effort off"
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.a.Config.ReasoningEffort != "" {
		t.Fatal("not turned off")
	}
}
