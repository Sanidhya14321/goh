package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestShellRulesDecideBeforeAsking(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.ShellRules = []core.ShellRule{{Match: "echo safe*", Action: "allow"}, {Match: "echo bad*", Action: "deny"}, {Match: "echo careful*", Action: "ask"}}
	asked := []string{}
	a.Approve = func(_ context.Context, p core.Permission) (core.Decision, error) {
		asked = append(asked, string(p.Arguments))
		return core.Decision{Allow: true}, nil
	}
	perm := func(tool, cmd string) core.Decision {
		b, _ := json.Marshal(map[string]string{"command": cmd})
		d, err := a.Permission(context.Background(), core.Permission{Tool: tool, Arguments: b})
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	if !perm("shell", "echo safe").Allow || perm("shell", "echo bad thing").Allow || perm("child:shell", "echo bad").Allow || perm("job_start", "echo bad").Allow {
		t.Fatal("rule decisions wrong")
	}
	if len(asked) != 0 {
		t.Fatal("rules should decide without asking", asked)
	}
	a.Grants["shell"] = true
	perm("shell", "echo careful")
	perm("shell", "echo other")
	if len(asked) != 1 || !strings.Contains(asked[0], "careful") {
		t.Fatal("ask rule must prompt even when shell is granted; other granted commands must not", asked)
	}
	perm("shell", "echo safe; echo more")
	if len(asked) != 1 {
		t.Fatal("granted shell should still run compound commands without a rule prompt", asked)
	}
	a.Grants["shell"] = false
	perm("shell", "echo safe && echo more")
	if len(asked) != 2 {
		t.Fatal("allow must not cover a compound command", asked)
	}
	var fromRules int
	for _, r := range a.Store.Records() {
		if r.Type == "permission" && strings.Contains(string(r.Data), "shell_rules") {
			fromRules++
		}
	}
	if fromRules != 4 {
		t.Fatal("rule decisions should be journaled", fromRules)
	}
}
