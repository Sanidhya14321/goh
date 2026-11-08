package tui

import (
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
	agent "example.com/independent-harness/internal/runtime"
)

func TestContextMeter(t *testing.T) {
	cfg := core.Defaults()
	cfg.ContextWindow = 200000
	m := &model{a: &agent.Agent{Config: cfg}}
	if m.contextLine() != "" {
		t.Fatal("no usage yet should show nothing")
	}
	m.Update(core.Event{Type: "assistant_completed", Data: map[string]any{"usage": core.Usage{Prompt: 45000, Completion: 5000, Total: 50000}}})
	m.Update(core.Event{Type: "assistant_completed", Data: map[string]any{"usage": core.Usage{Prompt: 90000, Completion: 10000, Total: 100000}}})
	if got := m.contextLine(); got != "Context: 50% of 200k" {
		t.Fatal(got)
	}
	if m.usage != 150000 {
		t.Fatal("total usage still accumulates", m.usage)
	}
	if !strings.Contains(strings.Join(m.sidebar("s", "model"), "\n"), "Context: 50%") {
		t.Fatal("sidebar missing context meter")
	}
	m.contextTokens = 999999
	if got := m.contextLine(); got != "Context: 100% of 200k" {
		t.Fatal("percent must cap at 100", got)
	}
}
