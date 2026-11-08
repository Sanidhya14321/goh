package runtime

import (
	"context"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestUserShellRecordsCallAndFeedsNextTurn(t *testing.T) {
	a, _ := testAgent(t)
	calls := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		calls++
		return answer(), nil
	})
	status, err := a.RunShell(context.Background(), "echo from-user-shell")
	if status != "completed" || err != nil || calls != 0 {
		t.Fatal(status, err, calls)
	}
	n := len(a.History)
	if a.History[n-3].Content != "!echo from-user-shell" || len(a.History[n-2].ToolCalls) != 1 ||
		a.History[n-1].Role != "tool" || !strings.Contains(a.History[n-1].Content, "from-user-shell") {
		t.Fatalf("%+v", a.History[n-3:])
	}
	var sawApproval, sawStart, sawResult bool
	for _, r := range a.Store.Records() {
		switch r.Type {
		case "permission":
			sawApproval = sawApproval || strings.Contains(string(r.Data), "user_typed")
		case "tool_start":
			sawStart = true
		case "tool_result":
			sawResult = true
		}
	}
	if !sawApproval || !sawStart || !sawResult || len(a.Store.Pending()) != 0 {
		t.Fatal("journal incomplete", sawApproval, sawStart, sawResult)
	}
	var seen []core.Message
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		seen = h
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "what did it print?"); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	joined := ""
	for _, m := range seen {
		joined += m.Content + "\n"
	}
	if !strings.Contains(joined, "from-user-shell") {
		t.Fatal("model did not see the shell output")
	}
	if a.Grants["shell"] {
		t.Fatal("a typed command granted shell to the model")
	}
}

func TestUserShellRefusals(t *testing.T) {
	a, _ := testAgent(t)
	if _, err := a.RunShell(context.Background(), "  "); err == nil {
		t.Fatal("empty command accepted")
	}
	a.Config.Agent, a.Config.AgentMode = "plan", "plan"
	before := len(a.History)
	if _, err := a.RunShell(context.Background(), "echo x"); err == nil || len(a.History) != before {
		t.Fatal("plan mode allowed a typed shell command", err)
	}
}
