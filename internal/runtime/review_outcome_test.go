package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

func lastOutcome(t *testing.T, a *Agent) string {
	t.Helper()
	out := ""
	for _, r := range a.Store.Records() {
		if r.Type == "run_finished" {
			var d struct{ Outcome string }
			json.Unmarshal(r.Data, &d)
			out = d.Outcome
		}
	}
	return out
}

// Review R11: a completed run says whether its changes were checked, and an
// empty reply is not success.
func TestRunOutcomes(t *testing.T) {
	testutil.RequireUnixShell(t)
	script := func(calls ...[]core.Call) Model {
		turn := 0
		return modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
			if turn < len(calls) {
				turn++
				return answer(calls[turn-1]...), nil
			}
			return answer(), nil
		})
	}
	create := toolCall("c", "create_file", map[string]any{"path": "new.txt", "content": "x\n"})
	check := toolCall("t", "shell", map[string]any{"command": "test -f new.txt"})
	for name, tc := range map[string]struct {
		calls [][]core.Call
		want  string
	}{
		"no changes": {nil, OutcomeNoChanges},
		"unverified": {[][]core.Call{{create}}, OutcomeUnverified},
		"verified":   {[][]core.Call{{create}, {check}}, OutcomeVerified},
	} {
		t.Run(name, func(t *testing.T) {
			a, _ := testAgent(t)
			a.Grants["create_file"], a.Grants["shell"] = true, true
			a.Provider = script(tc.calls...)
			if status, err := a.Run(context.Background(), "go"); status != "completed" {
				t.Fatal(status, err)
			}
			if got := lastOutcome(t, a); got != tc.want {
				t.Fatalf("outcome %q want %q", got, tc.want)
			}
		})
	}
	a, root := testAgent(t)
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return core.Reply{Message: core.Message{Role: "assistant"}, Finish: "stop"}, nil
	})
	if status, _ := a.Run(context.Background(), "fix it"); status == "completed" {
		t.Fatal("empty reply counted as success")
	}
	if _, err := os.Stat(filepath.Join(root, "new.txt")); err == nil {
		t.Fatal("unexpected file")
	}
}
