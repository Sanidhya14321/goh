package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/session"
)

type summaryModel struct {
	err     error
	summary string
	seen    []core.Message
}

func (m *summaryModel) Complete(_ context.Context, ms []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
	m.seen = append([]core.Message(nil), ms...)
	return core.Reply{Message: core.Message{Role: "assistant", Content: m.summary}, Finish: "stop"}, m.err
}
func contextAgent(t *testing.T) *Agent {
	t.Helper()
	c := core.Defaults()
	c.SessionDir = t.TempDir()
	c.ContextWindow = 40000
	c.OutputBudget = 2000
	s, e := session.New(c.SessionDir, t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	a, e := New(c, s)
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func TestCompactionPreservesRequestAndRecentGroups(t *testing.T) {
	a := contextAgent(t)
	a.History = append(a.History, core.Message{Role: "user", Content: "active task: fix the parser"})
	for i := 0; i < 6; i++ {
		call := core.Call{ID: ID(), Type: "function"}
		call.Function.Name = "read_file"
		call.Function.Arguments = `{"path":"x"}`
		a.History = append(a.History, core.Message{Role: "assistant", ToolCalls: []core.Call{call}}, core.Message{Role: "tool", ToolCallID: call.ID, Content: strings.Repeat("a", 6000)})
	}
	old := append([]core.Message(nil), a.History...)
	m := &summaryModel{summary: "Parser investigation; x hash abc. No changes yet."}
	a.Provider = m
	if e := a.compact(context.Background(), nil); e != nil {
		t.Fatal(e)
	}
	if len(a.History) != 7 {
		t.Fatalf("history length %d", len(a.History))
	}
	if a.History[2].Content != "active task: fix the parser" {
		t.Fatal("active request lost")
	}
	for i := 0; i < 4; i++ {
		if a.History[3+i].ToolCallID != old[len(old)-4+i].ToolCallID || a.History[3+i].Content != old[len(old)-4+i].Content {
			t.Fatal("recent groups changed")
		}
	}
	if len(m.seen) != 2 {
		t.Fatal("summary not requested")
	}
	if !strings.Contains(string(a.Store.Records()[len(a.Store.Records())-1].Data), "before_estimate") {
		t.Fatal("summary not journaled")
	}
}
func TestCompactionFailureKeepsHistory(t *testing.T) {
	for _, tc := range []struct {
		name  string
		model *summaryModel
	}{{"provider_failure", &summaryModel{err: errors.New("offline")}}, {"empty", &summaryModel{}}, {"ineffective", &summaryModel{summary: strings.Repeat("z", 45000)}}} {
		t.Run(tc.name, func(t *testing.T) {
			a := contextAgent(t)
			a.History = append(a.History, core.Message{Role: "user", Content: "older"}, core.Message{Role: "assistant", Content: strings.Repeat("x", 19000)}, core.Message{Role: "user", Content: "active"}, core.Message{Role: "assistant", Content: "recent1"}, core.Message{Role: "assistant", Content: strings.Repeat("y", 6000)})
			a.Provider = tc.model
			before := estimate(a.History, nil)
			if e := a.compact(context.Background(), nil); e == nil {
				t.Fatal("accepted failed summary")
			}
			if estimate(a.History, nil) != before {
				t.Fatal("modified failed context")
			}
		})
	}
}
func TestCompactionUnresolvedNeverRemoved(t *testing.T) {
	a := contextAgent(t)
	c := core.Call{ID: "pending"}
	c.Function.Name = "shell"
	c.Function.Arguments = `{"command":"true"}`
	a.History = append(a.History, core.Message{Role: "user", Content: "active"}, core.Message{Role: "assistant", ToolCalls: []core.Call{c}}, core.Message{Role: "assistant", Content: strings.Repeat("a", 15000)}, core.Message{Role: "assistant", Content: strings.Repeat("b", 15000)})
	a.Provider = &summaryModel{summary: "small"}
	if e := a.compact(context.Background(), nil); e == nil {
		t.Fatal("unresolved group should exhaust preserved context")
	}
	found := false
	for _, m := range a.History {
		for _, v := range m.ToolCalls {
			found = found || v.ID == "pending"
		}
	}
	if !found {
		t.Fatal("unresolved call lost")
	}
}
