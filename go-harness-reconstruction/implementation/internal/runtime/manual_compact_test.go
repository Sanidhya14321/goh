package runtime

import (
	"context"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestManualCompactDurableWithoutToolStartup(t *testing.T) {
	a := contextAgent(t)
	a.Config.ContextWindow = 131072
	for _, m := range []core.Message{{Role: "user", Content: "fix parser"}, {Role: "assistant", Content: strings.Repeat("older investigation. ", 100)}, {Role: "user", Content: "verify parser"}, {Role: "assistant", Content: "recent group one"}, {Role: "assistant", Content: "recent group two"}} {
		if e := a.commit(m); e != nil {
			t.Fatal(e)
		}
	}
	original := len(a.Store.Records())
	a.Provider = &summaryModel{summary: "Parser work state retained; run tests next."}
	a.Todos = []core.Todo{{Content: "verify parser", Status: "pending", Priority: "high"}}
	calls := 0
	a.ExternalFactory = func() External { calls++; return nil }
	var events []core.Event
	a.Emit = func(e core.Event) { events = append(events, e) }
	status, e := a.Compact(context.Background())
	if e != nil || status != "completed" {
		t.Fatal(status, e)
	}
	if calls != 0 {
		t.Fatal("manual compact launched integrations")
	}
	if len(a.Store.Records()) <= original {
		t.Fatal("summary not durable")
	}
	if a.History[1].Role == "system" {
		t.Fatal("summary elevated to trusted instructions")
	}
	if !strings.Contains(a.modelHistory()[0].Content, "verify parser") {
		t.Fatal("task state lost")
	}
	terminal, compacted := 0, 0
	for _, event := range events {
		if event.Type == "run_finished" {
			terminal++
		}
		if event.Type == "context_compacted" {
			compacted++
		}
	}
	if terminal != 1 || compacted != 1 || events[len(events)-1].Type != "run_finished" {
		t.Fatal("manual lifecycle incorrect")
	}
	resumed, e := New(a.Config, a.Store)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(resumed.History[1].Content, "Parser work state") {
		t.Fatal("manual summary not resumed")
	}
}
func TestManualCompactFailureAndPendingProtection(t *testing.T) {
	a := contextAgent(t)
	a.Provider = &summaryModel{summary: "ok"}
	if status, e := a.Compact(context.Background()); e == nil || status != "failed" {
		t.Fatal("empty session compacted")
	}
	if e := a.commit(core.Message{Role: "user", Content: "task"}); e != nil {
		t.Fatal(e)
	}
	if e := a.Store.Append("tool_start", map[string]any{"id": "uncertain", "name": "shell"}); e != nil {
		t.Fatal(e)
	}
	if status, e := a.Compact(context.Background()); e == nil || status != "failed" {
		t.Fatal("compacted uncertain effects")
	}
}
