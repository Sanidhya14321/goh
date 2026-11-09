package tui

import (
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
)

func TestReasoningBlockAndToggle(t *testing.T) {
	m := &model{a: &agent.Agent{Config: core.Defaults(), Store: &session.Store{ID: "fixture"}}, width: 80, height: 24, status: "ready"}
	m.Update(core.Event{Type: "reasoning_updated", Data: map[string]any{"text": "Check the "}})
	m.Update(core.Event{Type: "reasoning_updated", Data: map[string]any{"text": "tests\x00H first"}})
	if strings.Contains(m.log, "Check") {
		t.Fatal("reasoning written before the turn moved on")
	}
	m.Update(core.Event{Type: "assistant_updated", Data: map[string]any{"text": "Done."}})
	shown := m.displayLog()
	if !strings.Contains(shown, "┊ Check the tests") || strings.Index(shown, "Check") > strings.Index(shown, "Done.") {
		t.Fatalf("shown:\n%q", shown)
	}
	if strings.Contains(m.log, "\x00H first") {
		t.Fatal("marker forged through reasoning text")
	}
	m.toggleThinking()
	hidden := m.displayLog()
	if strings.Contains(hidden, "Check") || !strings.Contains(hidden, "Done.") {
		t.Fatalf("hidden:\n%q", hidden)
	}
}

func TestRetryNoticeInStatus(t *testing.T) {
	m := &model{a: &agent.Agent{Config: core.Defaults(), Store: &session.Store{ID: "fixture"}}, width: 80, height: 24}
	m.Update(core.Event{Type: "provider_retry", Data: core.RetryNotice{Attempt: 2, Max: 3, DelayMS: 1500, Reason: "provider unavailable"}})
	if !strings.Contains(m.status, "retrying in 1.5s, attempt 2 of 3") {
		t.Fatal(m.status)
	}
}
