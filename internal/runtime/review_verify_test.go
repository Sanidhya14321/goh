package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

// Review R21: a bounded verify -> repair -> verify cycle with an explicit
// outcome, and a per-run tool-call budget.
func TestVerifyRepairCycle(t *testing.T) {
	testutil.RequireUnixShell(t)
	run := func(attempts int, fixes bool) (*Agent, string, int) {
		a, _ := testAgent(t)
		a.Grants["create_file"] = true
		a.Config.Verify = &core.VerifyConfig{Command: "test -f done.txt", RepairAttempts: &attempts}
		turn, sawCheck := 0, 0
		a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
			turn++
			last := h[len(h)-1]
			if last.Role == "user" && strings.Contains(last.Content, "verification command failed") {
				sawCheck++
				if fixes {
					return answer(toolCall("fix", "create_file", map[string]any{"path": "done.txt", "content": "ok\n"})), nil
				}
				return answer(), nil
			}
			if turn == 1 {
				return answer(toolCall("c", "create_file", map[string]any{"path": "work.txt", "content": "x\n"})), nil
			}
			return answer(), nil
		})
		status, err := a.Run(context.Background(), "do the work")
		if status != "completed" {
			t.Fatal(status, err)
		}
		return a, lastOutcome(t, a), sawCheck
	}
	if _, outcome, saw := run(1, true); outcome != OutcomeVerified || saw != 1 {
		t.Fatalf("repaired run: outcome %s, failures shown %d", outcome, saw)
	}
	if _, outcome, saw := run(0, true); outcome != OutcomeFailedVerification || saw != 0 {
		t.Fatalf("no-repair run: outcome %s, failures shown %d", outcome, saw)
	}
	if _, outcome, saw := run(2, false); outcome != OutcomeFailedVerification || saw != 2 {
		t.Fatalf("unfixed run: outcome %s, failures shown %d (repairs must be bounded)", outcome, saw)
	}
}

func TestMaxToolCallsBudget(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.MaxToolCalls = 3
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return answer(toolCall(ID(), "list_files", map[string]any{"path": "."}), toolCall(ID(), "glob", map[string]any{"pattern": "*"})), nil
	})
	status, err := a.Run(context.Background(), "loop")
	if status != "budget_exhausted" || err == nil || !strings.Contains(err.Error(), "max_tool_calls") || a.toolCalls != 2 {
		t.Fatal(status, err, a.toolCalls)
	}
}
