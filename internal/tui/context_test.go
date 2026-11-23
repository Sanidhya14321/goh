package tui

import (
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
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

func TestSidebarCostFromConfiguredPrice(t *testing.T) {
	cfg := core.Defaults()
	cfg.Model = "m"
	cfg.Prices = map[string]core.ModelPrice{"m": {Input: 2, Output: 8}}
	m := &model{a: &agent.Agent{Config: cfg, Store: &session.Store{ID: "fixture"}}, width: 120, height: 40}
	m.Update(core.Event{Type: "assistant_completed", Data: map[string]any{"usage": core.Usage{Prompt: 500_000, Completion: 125_000}}})
	if side := strings.Join(m.sidebar("s", "m"), "\n"); !strings.Contains(side, "~$2.0000 estimated") {
		t.Fatal(side)
	}
	cfg.Prices = nil
	m = &model{a: &agent.Agent{Config: cfg, Store: &session.Store{ID: "fixture"}}, width: 120, height: 40}
	m.Update(core.Event{Type: "assistant_completed", Data: map[string]any{"usage": core.Usage{Prompt: 10}}})
	if strings.Contains(strings.Join(m.sidebar("s", "m"), "\n"), "estimated") {
		t.Fatal("cost shown without a price")
	}
}

// Handoff P1: usage from a fallback route is not divided by the primary's
// window; the meter omits the denominator until the primary answers again.
func TestContextMeterOmitsDenominatorForFallback(t *testing.T) {
	cfg := core.Defaults()
	cfg.ContextWindow = 1048576
	m := &model{a: &agent.Agent{Config: cfg}}
	m.Update(core.Event{Type: "assistant_completed", Data: map[string]any{"usage": core.Usage{Prompt: 40000, Completion: 1000}, "model": "m", "fallback": "backup"}})
	if got := m.contextLine(); got != "Context: 41k (fallback backup)" {
		t.Fatal(got)
	}
	m.Update(core.Event{Type: "assistant_completed", Data: map[string]any{"usage": core.Usage{Prompt: 104858, Completion: 0}, "model": "m"}})
	if got := m.contextLine(); got != "Context: 10% of 1048k" {
		t.Fatal(got)
	}
}

// Each reply ends with a dim line naming the model, the run's time and the
// tokens sent and received, so a short prompt's cost is explained by its
// input (instructions and tool definitions), not hidden in one total.
func TestReplyMetaLine(t *testing.T) {
	cfg := core.Defaults()
	cfg.Model, cfg.ContextWindow = "gpt-6.1-sol", 272000
	m := &model{a: &agent.Agent{Config: cfg}}
	m.Update(core.Event{Type: "run_started"})
	m.log += "Hi! What can I help you with?"
	m.Update(core.Event{Type: "assistant_completed", Data: map[string]any{"usage": core.Usage{Prompt: 1551, Completion: 13, Total: 1564}, "model": "gpt-6.1-sol"}})
	m.Update(core.Event{Type: "run_finished", Data: map[string]any{"status": "completed", "duration_ms": 5745}})
	if want := "gpt-6.1-sol · 5.7s · 1,551 in · 13 out"; !strings.Contains(m.log, verbatimMark+want+"\n") {
		t.Fatalf("meta line missing: %q", m.log)
	}
	side := strings.Join(m.sidebar("s", "gpt-6.1-sol"), "\n")
	if !strings.Contains(side, "1.6k in · 13 out") || !strings.Contains(side, "▰") {
		t.Fatal(side)
	}
	// Cached input and several requests are named; a run without usage
	// (a local shell command) adds no line.
	m.Update(core.Event{Type: "run_started"})
	for i := 0; i < 2; i++ {
		m.Update(core.Event{Type: "assistant_completed", Data: map[string]any{"usage": core.Usage{Prompt: 2000, Completion: 50, CacheRead: 1500}}})
	}
	m.Update(core.Event{Type: "run_finished", Data: map[string]any{"status": "completed", "duration_ms": 1200}})
	if want := "gpt-6.1-sol · 1.2s · 4,000 in (3,000 cached) · 100 out · 2 requests"; !strings.Contains(m.log, want) {
		t.Fatalf("multi-request line: %q", m.log)
	}
	before := m.log
	m.Update(core.Event{Type: "run_started"})
	m.Update(core.Event{Type: "run_finished", Data: map[string]any{"status": "completed", "duration_ms": 30}})
	if strings.Count(m.log, verbatimMark) != strings.Count(before, verbatimMark) {
		t.Fatal("local run added a usage line")
	}
}
