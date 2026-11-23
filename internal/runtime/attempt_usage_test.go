package runtime

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
	"github.com/Sanidhya14321/goh/internal/session"
)

// journalUsage sums usage records: total tokens, records, discarded records.
func journalUsage(t *testing.T, a *Agent) (total, records, discarded int) {
	t.Helper()
	for _, r := range a.Store.Records() {
		if r.Type != "usage" {
			continue
		}
		var u struct {
			Total     int  `json:"total_tokens"`
			Discarded bool `json:"discarded"`
		}
		json.Unmarshal(r.Data, &u)
		total += u.Total
		records++
		if u.Discarded {
			discarded++
		}
	}
	return
}

func lastRunFinished(t *testing.T, a *Agent) map[string]any {
	t.Helper()
	var d map[string]any
	for _, r := range a.Store.Records() {
		if r.Type == "run_finished" {
			d = nil
			json.Unmarshal(r.Data, &d)
		}
	}
	return d
}

// Handoff P1: an interrupted attempt that reported usage is charged before
// the retry is decided; once it exhausts max_run_tokens no retry is sent,
// and no tool from the discarded reply runs.
func TestInterruptedAttemptUsageCountsBeforeRetry(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.RetryInterruptedStreams = 2
	a.Config.MaxRunTokens = 50
	calls := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		calls++
		r := answer(toolCall("x", "list_files", map[string]any{}))
		r.Usage = &core.Usage{Prompt: 50, Completion: 10, Total: 60}
		return r, provider.ErrInterrupted
	})
	status, err := a.Run(context.Background(), "hello")
	if status != "budget_exhausted" || err == nil {
		t.Fatal(status, err)
	}
	if calls != 1 {
		t.Fatalf("retried after the budget was spent: %d requests", calls)
	}
	if total, records, discarded := journalUsage(t, a); total != 60 || records != 1 || discarded != 1 {
		t.Fatalf("usage total %d records %d discarded %d", total, records, discarded)
	}
	if a.toolCalls != 0 {
		t.Fatal("tool from a discarded reply ran")
	}
}

// Within budget, both the discarded and the accepted attempt are recorded;
// an interrupted attempt without usage is reported as missing usage.
func TestInterruptedAttemptsRecordedWithAndWithoutUsage(t *testing.T) {
	for _, withUsage := range []bool{true, false} {
		a, _ := testAgent(t)
		a.Config.RetryInterruptedStreams = 1
		calls := 0
		a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
			calls++
			r := answer()
			r.Usage = &core.Usage{Prompt: 30, Completion: 5, Total: 35}
			if calls == 1 {
				if !withUsage {
					r.Usage = nil
				}
				return r, provider.ErrInterrupted
			}
			return r, nil
		})
		if status, err := a.Run(context.Background(), "hello"); status != "completed" {
			t.Fatal(status, err)
		}
		total, records, discarded := journalUsage(t, a)
		missing, _ := lastRunFinished(t, a)["usage_missing_calls"].(float64)
		switch {
		case calls != 2:
			t.Fatalf("requests %d", calls)
		case withUsage && (total != 70 || records != 2 || discarded != 1 || missing != 0):
			t.Fatalf("with usage: total %d records %d discarded %d missing %v", total, records, discarded, missing)
		case !withUsage && (total != 35 || records != 1 || discarded != 0 || missing != 1):
			t.Fatalf("without usage: total %d records %d discarded %d missing %v", total, records, discarded, missing)
		}
	}
}

// A provider error before any response, then a fallback: only the fallback
// reply is charged, and a pre-response failure is not "missing usage".
func TestPreResponseErrorThenFallbackUsage(t *testing.T) {
	a := failoverAgent(t, "backup")
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return core.Reply{}, provider.ErrConnection
	})
	a.RoleModelFactory = func(core.Config, string) Model {
		return modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
			r := answer()
			r.Usage = &core.Usage{Prompt: 10, Completion: 2, Total: 12}
			return r, nil
		})
	}
	if status, err := a.Run(context.Background(), "hello"); status != "completed" {
		t.Fatal(status, err)
	}
	if total, records, _ := journalUsage(t, a); total != 12 || records != 1 {
		t.Fatalf("usage %d in %d records", total, records)
	}
	if _, ok := lastRunFinished(t, a)["usage_missing_calls"]; ok {
		t.Fatal("pre-response failure counted as missing usage")
	}
}

// A child whose provider reports no usage keeps that signal in the parent,
// and the child's tokens are counted once.
func TestChildUsageAggregatedOnce(t *testing.T) {
	a, _ := testAgent(t)
	a.Grants["task"] = true
	a.ChildModelFactory = func(core.Config, *session.Store) Model {
		return modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
			return core.Reply{Message: core.Message{Role: "assistant", Content: "child done"}, Finish: "stop"}, nil // no usage
		})
	}
	turn := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		turn++
		r := answer()
		if turn == 1 {
			r = answer(toolCall("t", "task", map[string]any{"description": "explore", "prompt": "look", "subagent_type": "general"}))
		}
		r.Usage = &core.Usage{Prompt: 100, Completion: 10, Total: 110}
		return r, nil
	})
	if status, err := a.Run(context.Background(), "delegate"); status != "completed" {
		t.Fatal(status, err)
	}
	if a.runTokens != 220 {
		t.Fatalf("run tokens %d", a.runTokens)
	}
	if missing, _ := lastRunFinished(t, a)["usage_missing_calls"].(float64); missing != 1 {
		t.Fatalf("child missing usage not reported: %v", missing)
	}
}

// A local shell operation after a model run starts with its own counters.
func TestShellOperationStartsWithOwnCounters(t *testing.T) {
	a, _ := testAgent(t)
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return answer(), nil // no usage reported
	})
	if status, err := a.Run(context.Background(), "hello"); status != "completed" {
		t.Fatal(status, err)
	}
	if _, ok := lastRunFinished(t, a)["usage_missing_calls"]; !ok {
		t.Fatal("fixture: model run should report missing usage")
	}
	a.Grants["shell"] = true
	if status, err := a.RunShell(context.Background(), "true"); status != "completed" {
		t.Fatal(status, err)
	}
	if _, ok := lastRunFinished(t, a)["usage_missing_calls"]; ok {
		t.Fatal("shell operation inherited the model run's missing usage")
	}
	if a.runTokens != 0 {
		t.Fatal("shell operation inherited run tokens")
	}
}
