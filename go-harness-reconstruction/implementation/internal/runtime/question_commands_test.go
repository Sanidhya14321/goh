package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func questionArguments() map[string]any {
	return map[string]any{"questions": []any{map[string]any{"header": "Checks", "question": "Which checks should run?", "multiple": true, "options": []any{map[string]any{"label": "unit", "description": "Focused checks"}, map[string]any{"label": "race", "description": "Concurrency checks"}}}, map[string]any{"header": "Notes", "question": "Any extra context?", "options": []any{}}}}
}
func TestQuestionInteractiveHeadlessAndDismissed(t *testing.T) {
	for _, mode := range []string{"answered", "headless", "dismissed"} {
		t.Run(mode, func(t *testing.T) {
			a, _ := testAgent(t)
			turn := 0
			asked := 0
			if mode != "headless" {
				a.Ask = func(ctx context.Context, q core.QuestionRequest) (core.QuestionReply, error) {
					asked++
					if len(q.Questions) != 2 {
						t.Fatal(q)
					}
					if mode == "dismissed" {
						return core.QuestionReply{}, ErrQuestionDismissed
					}
					return core.QuestionReply{Answers: [][]string{{"unit", "race"}, {"custom task context"}}}, nil
				}
			}
			a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
				turn++
				if turn == 1 {
					return answer(toolCall("ask", "question", questionArguments())), nil
				}
				if mode == "answered" {
					r := toolResult(t, h[len(h)-1])
					if !strings.Contains(r["answers"].([]any)[1].([]any)[0].(string), "custom task context") {
						t.Fatal(r)
					}
				} else {
					if !strings.Contains(h[len(h)-1].Content, "dismissed") {
						t.Fatal("dismissal not paired")
					}
				}
				return answer(), nil
			})
			var events []core.Event
			a.Emit = func(e core.Event) { events = append(events, e) }
			status, e := a.Run(context.Background(), "investigate checks")
			if mode == "headless" {
				if status != "needs_input" || !errors.Is(e, ErrNeedsInput) || turn != 1 {
					t.Fatal(status, e, turn)
				}
			} else if status != "completed" || e != nil || asked != 1 {
				t.Fatal(status, e, asked)
			}
			terminal(t, events, status)
			if len(a.Store.Pending()) != 0 {
				t.Fatal("questions created uncertain external effects")
			}
			for _, r := range a.Store.Records() {
				if r.Type == "permission" {
					t.Fatal("answer interpreted as permission")
				}
			}
		})
	}
}
func TestQuestionValidationAndCancellation(t *testing.T) {
	a, _ := testAgent(t)
	for _, bad := range []string{`null`, `{"questions":[]}`, `{"questions":[{"header":"h","question":"q","options":[{"label":"a"}]}]}`, `{"questions":[{"header":"h","question":"q","multiple":null,"options":[]}]}`} {
		if _, e := parseQuestion(json.RawMessage(bad)); e == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.Ask = func(ctx context.Context, _ core.QuestionRequest) (core.QuestionReply, error) {
		cancel()
		return core.QuestionReply{}, ctx.Err()
	}
	b, _ := json.Marshal(questionArguments())
	if _, e := a.executeQuestion(ctx, b); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}
func TestCustomCommandsEnterSharedRuntimeAsUserData(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.Commands = map[string]core.CommandConfig{"review": {Template: "Review $1. Details: $2. Raw: $ARGUMENTS"}, "literal": {Template: "Keep !`printf never-execute` and @missing literal: $ARGUMENTS"}}
	a.Config.Agent = "plan"
	a.Config.AgentMode = "plan"
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		prompt := h[len(h)-1].Content
		if !strings.Contains(prompt, "Review parser.go. Details: edge cases streaming.") {
			t.Fatal(prompt)
		}
		return answer(), nil
	})
	status, e := a.Run(context.Background(), `/review parser.go "edge cases" streaming`)
	if e != nil || status != "completed" {
		t.Fatal(status, e)
	}
	if len(a.Store.Pending()) > 0 {
		t.Fatal("command expansion started effects")
	}
	found := false
	for _, r := range a.Store.Records() {
		found = found || r.Type == "command_invoked"
		if r.Type == "tool_start" {
			t.Fatal("pure command executed tool")
		}
	}
	if !found {
		t.Fatal("command provenance missing")
	}
	called := false
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		called = true
		return answer(), nil
	})
	if _, e = a.Run(context.Background(), "/unknown"); e == nil || called {
		t.Fatal("unknown command reached provider")
	}
}

func TestQuestionCancellationCanContinueWithoutReplay(t *testing.T) {
	a, _ := testAgent(t)
	ctx, cancel := context.WithCancel(context.Background())
	a.Ask = func(ctx context.Context, _ core.QuestionRequest) (core.QuestionReply, error) {
		cancel()
		return core.QuestionReply{}, ctx.Err()
	}
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return answer(toolCall("ask", "question", questionArguments())), nil
	})
	if status, e := a.Run(ctx, "ask me"); status != "cancelled" || e == nil {
		t.Fatal(status, e)
	}
	if len(a.Store.Pending()) != 0 {
		t.Fatal("user input created uncertain effect")
	}
	calls := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		calls++
		found := 0
		for _, m := range h {
			if m.Role == "tool" && m.ToolCallID == "ask" {
				found++
				if !strings.Contains(m.Content, "User input was interrupted") {
					t.Fatal(m)
				}
			}
		}
		if found != 1 {
			t.Fatal("interrupted call not repaired exactly once", found)
		}
		return answer(), nil
	})
	if status, e := a.Run(context.Background(), "continue without the answer"); status != "completed" || e != nil || calls != 1 {
		t.Fatal(status, e, calls)
	}
}

func TestRepairReusedToolIDProducesOneResult(t *testing.T) {
	a, _ := testAgent(t)
	old := toolCall("reused", "question", questionArguments())
	for _, m := range []core.Message{{Role: "assistant", ToolCalls: []core.Call{old}}, {Role: "tool", ToolCallID: "reused", Content: `{"error":false,"content":"answered"}`}, {Role: "assistant", ToolCalls: []core.Call{old}}} {
		if e := a.commit(m); e != nil {
			t.Fatal(e)
		}
	}
	before := len(a.History)
	if e := a.RepairInterruptedHistory(); e != nil {
		t.Fatal(e)
	}
	if len(a.History) != before+1 {
		t.Fatal("reused ID repaired multiple times")
	}
	if e := a.RepairInterruptedHistory(); e != nil {
		t.Fatal(e)
	}
	if len(a.History) != before+1 {
		t.Fatal("repair was not idempotent")
	}
}

func TestCommandAgentDelegatesLikeAMention(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.Commands = map[string]core.CommandConfig{
		"scan": {Template: "Scan $ARGUMENTS for dead code", Agent: "explore"},
		"bad":  {Template: "x", Agent: "nobody"},
	}
	var seen string
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		seen = h[len(h)-1].Content
		return answer(), nil
	})
	if status, e := a.Run(context.Background(), "/scan internal"); e != nil || status != "completed" {
		t.Fatal(status, e)
	}
	if !strings.HasPrefix(seen, "@explore Scan internal for dead code") || !strings.Contains(seen, "mentioned the explore subagent") {
		t.Fatal(seen)
	}
	seen = ""
	if _, e := a.Run(context.Background(), "/bad"); e == nil || seen != "" {
		t.Fatal("unknown command agent reached the provider", e)
	}
}
