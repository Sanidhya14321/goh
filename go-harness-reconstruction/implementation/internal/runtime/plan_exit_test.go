package runtime

import (
	"context"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func planAgent(t *testing.T, allow bool) (*Agent, *int) {
	a, _ := testAgent(t)
	a.Config.Agent, a.Config.AgentMode = "plan", "plan"
	asked := 0
	a.Approve = func(_ context.Context, p core.Permission) (core.Decision, error) {
		if p.Tool == "plan_exit" {
			asked++
			if !strings.Contains(p.Preview, "1. change the parser") {
				t.Fatal("approval must show the plan", p.Preview)
			}
		}
		return core.Decision{Allow: allow}, nil
	}
	turn := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		turn++
		if turn == 1 {
			return answer(toolCall("p1", "plan_exit", map[string]any{"plan": "1. change the parser\n2. add tests"})), nil
		}
		return answer(), nil
	})
	return a, &asked
}

func TestPlanExitHandsOffOnApproval(t *testing.T) {
	a, asked := planAgent(t, true)
	if status, err := a.Run(context.Background(), "plan it"); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	if *asked != 1 || !strings.Contains(a.HandoffPlan, "add tests") {
		t.Fatal("approved plan not handed off", *asked, a.HandoffPlan)
	}
	b, _ := planAgent(t, false)
	b.Run(context.Background(), "plan it")
	if b.HandoffPlan != "" {
		t.Fatal("declined plan_exit must not hand off")
	}
}

func TestPlanExitOnlyForPlanAgent(t *testing.T) {
	a, _ := testAgent(t)
	for _, d := range a.localDefinitions() {
		if d.Function.Name == "plan_exit" {
			t.Fatal("build mode offers plan_exit")
		}
	}
	if _, _, err := a.prepare("plan_exit", []byte(`{"plan":"x"}`)); err == nil {
		t.Fatal("build mode accepted plan_exit")
	}
	a.Config.Agent, a.Config.AgentMode = "plan", "plan"
	found := false
	for _, d := range a.localDefinitions() {
		found = found || d.Function.Name == "plan_exit"
	}
	if !found {
		t.Fatal("plan mode lacks plan_exit")
	}
	a.depth = 1
	if _, _, err := a.prepare("plan_exit", []byte(`{"plan":"x"}`)); err == nil {
		t.Fatal("subagent accepted plan_exit")
	}
	headless, _ := planAgent(t, true)
	headless.Approve = nil
	if status, _ := headless.Run(context.Background(), "plan"); status != "needs_permission" {
		t.Fatal("headless plan_exit should stop for approval", status)
	}
}
