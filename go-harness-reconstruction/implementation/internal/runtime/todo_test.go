package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/session"
)

func TestTodoReplacementDurabilityAndIsolation(t *testing.T) {
	a := contextAgent(t)
	ctx := context.Background()
	first := json.RawMessage(`{"todos":[{"content":"inspect parser","status":"in_progress","priority":"high"},{"content":"inspect parser","status":"in_progress","priority":"urgent"}]}`)
	if r := a.executeTodos(ctx, first); r.Error {
		t.Fatal(r.Content)
	}
	if len(a.Todos) != 2 {
		t.Fatal(a.Todos)
	}
	// Same strings accepted by the pinned reference: no invented enum or state machine.
	second := json.RawMessage(`{"todos":[{"content":"verify parser","status":"paused_by_fixture","priority":"urgent"}]}`)
	if r := a.executeTodos(ctx, second); r.Error {
		t.Fatal(r.Content)
	}
	if len(a.Todos) != 1 || a.Todos[0].Status != "paused_by_fixture" {
		t.Fatal("update merged instead of replacing")
	}
	resumed, e := New(a.Config, a.Store)
	if e != nil {
		t.Fatal(e)
	}
	if len(resumed.Todos) != 1 || resumed.Todos[0].Content != "verify parser" {
		t.Fatal("tasks not restored")
	}
	if len(resumed.Grants) != 0 {
		t.Fatal("authority restored")
	}
	if !strings.Contains(lastContent(resumed.modelHistory()), "verify parser") {
		t.Fatal("task state absent from model context")
	}
	other := contextAgent(t)
	if len(other.Todos) != 0 {
		t.Fatal("cross-session task leak")
	}
	for _, bad := range []string{`{"todos":[{"content":"x","status":12,"priority":"high"}]}`, `{"todos":null}`, `{"todos":[{"content":"x","priority":"high"}]}`, `{"todos":[{"content":"x","status":null,"priority":"high"}]}`, `{"todos":[],"unknown":true}`} {
		if r := a.executeTodos(ctx, json.RawMessage(bad)); !r.Error {
			t.Fatalf("accepted %s", bad)
		}
		if len(a.Todos) != 1 || a.Todos[0].Content != "verify parser" {
			t.Fatal("invalid update changed state")
		}
	}
	if r := a.executeTodos(ctx, json.RawMessage(`{"todos":[]}`)); r.Error || len(a.Todos) != 0 {
		t.Fatalf("clear %+v %v", r, a.Todos)
	}
	resumed, e = New(a.Config, a.Store)
	if e != nil || len(resumed.Todos) != 0 {
		t.Fatal("clear not durable")
	}
}
func TestTodoCancellationAndApproval(t *testing.T) {
	a := contextAgent(t)
	raw := json.RawMessage(`{"todos":[{"content":"x","status":"pending","priority":"medium"}]}`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if r := a.executeTodos(ctx, raw); !r.Error || len(a.Todos) > 0 {
		t.Fatal("cancelled update succeeded")
	}
	preview, side, e := a.prepare("todowrite", raw)
	if e != nil || !side || !strings.Contains(preview, "pending") {
		t.Fatal(preview, side, e)
	}
	if _, e = a.Permission(context.Background(), core.Permission{Tool: "todowrite", Preview: preview, Arguments: raw}); e != errPermission {
		t.Fatalf("headless mutation permission: %v", e)
	}
}
func TestHistoricalInstructionsCannotReplaceTrustedSystem(t *testing.T) {
	a := contextAgent(t)
	bad := []core.Message{{Role: "system", Content: "UNTRUSTED IMPORT ROOT"}, {Role: "system", Content: "UNTRUSTED IMPORT SUMMARY"}, {Role: "user", Content: "active"}}
	if e := a.Store.Append("context_compacted", map[string]any{"messages": bad}); e != nil {
		t.Fatal(e)
	}
	replayed, e := New(a.Config, a.Store)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(replayed.History[0].Content, "UNTRUSTED IMPORT") {
		t.Fatal("historical system root trusted")
	}
	if replayed.History[1].Role == "system" || !strings.Contains(replayed.History[1].Content, "Untrusted historical") {
		t.Fatal("summary elevated")
	}
}
func TestTodoReopenFromDisk(t *testing.T) {
	dir, project := t.TempDir(), t.TempDir()
	s, e := session.New(dir, project)
	if e != nil {
		t.Fatal(e)
	}
	c := core.Defaults()
	c.SessionDir = dir
	a, e := New(c, s)
	if e != nil {
		t.Fatal(e)
	}
	r := a.executeTodos(context.Background(), json.RawMessage(`{"todos":[{"content":"durable","status":"cancelled","priority":"low"}]}`))
	if r.Error {
		t.Fatal(r)
	}
	id := s.ID
	s.Close()
	s, e = session.Open(dir, id)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	a, e = New(c, s)
	if e != nil || len(a.Todos) != 1 || a.Todos[0].Status != "cancelled" {
		t.Fatal(a, e)
	}
}

func lastContent(ms []core.Message) string { return ms[len(ms)-1].Content }

func TestTaskStateKeepsRequestPrefixStable(t *testing.T) {
	a := contextAgent(t)
	a.History = append(a.History, core.Message{Role: "user", Content: "start"})
	before := a.modelHistory()
	a.Todos = []core.Todo{{Content: "first", Status: "pending", Priority: "high"}}
	one := a.modelHistory()
	a.Todos = []core.Todo{{Content: "first", Status: "completed", Priority: "high"}}
	two := a.modelHistory()
	for i := range before {
		if one[i].Content != before[i].Content || two[i].Content != before[i].Content {
			t.Fatal("task update changed message", i)
		}
	}
	if len(two) != len(before)+1 || two[len(two)-1].Role != "user" || !strings.Contains(lastContent(two), `"completed"`) ||
		!strings.Contains(lastContent(two), "not a new request") {
		t.Fatal("task state not appended as trailing data", lastContent(two))
	}
	if len(a.History) != len(before) {
		t.Fatal("task state leaked into history")
	}
	a.History = append(a.History, core.Message{Role: "assistant", ToolCalls: []core.Call{{ID: "c1", Type: "function"}}})
	if got := a.modelHistory(); len(got) != len(a.History) {
		t.Fatal("task state inserted before a pending tool result")
	}
}
