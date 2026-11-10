package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// C2: a run that changed files without a passing check is reminded once to
// check before finishing; a run that checked is not.
func TestCompletionCheck(t *testing.T) {
	testutil.RequireUnixShell(t)
	for name, checks := range map[string]bool{"unchecked": false, "checked": true} {
		t.Run(name, func(t *testing.T) {
			a, _ := testAgent(t)
			a.Config.CompletionCheck = nil // default on
			a.Grants["create_file"], a.Grants["shell"] = true, true
			reminders, turn := 0, 0
			a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
				turn++
				if last := h[len(h)-1]; last.Role == "user" && strings.Contains(last.Content, "no test, build or other check") {
					reminders++
					return answer(toolCall("t", "shell", map[string]any{"command": "test -f new.txt"})), nil
				}
				switch turn {
				case 1:
					return answer(toolCall("c", "create_file", map[string]any{"path": "new.txt", "content": "x\n"})), nil
				case 2:
					if checks {
						return answer(toolCall("t0", "shell", map[string]any{"command": "test -f new.txt"})), nil
					}
				}
				return answer(), nil
			})
			if status, err := a.Run(context.Background(), "go"); status != "completed" {
				t.Fatal(status, err)
			}
			want := 1
			if checks {
				want = 0
			}
			if reminders != want || lastOutcome(t, a) != OutcomeVerified {
				t.Fatalf("reminders %d want %d, outcome %s", reminders, want, lastOutcome(t, a))
			}
		})
	}
}

// Benchmark finding: checks that pass against a fake the model wrote itself
// trigger one reminder to confirm calls against the real definitions.
func TestCompletionCheckTestDouble(t *testing.T) {
	testutil.RequireUnixShell(t)
	for name, tc := range map[string]struct {
		path, content string
		want          int
	}{
		"own fake":       {"tests/test_pay.py", "class FakeGateway:\n    pass\n", 1},
		"plain test":     {"tests/test_pay.py", "def test_pay():\n    assert True\n", 0},
		"fake in source": {"pay.py", "class FakeGateway:\n    pass\n", 0},
	} {
		t.Run(name, func(t *testing.T) {
			a, root := testAgent(t)
			a.Config.CompletionCheck = nil
			a.Grants["create_file"], a.Grants["shell"] = true, true
			reminders, turn := 0, 0
			a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
				turn++
				if last := h[len(h)-1]; last.Role == "user" && strings.Contains(last.Content, "fake, mock or stub you wrote") {
					reminders++
					return answer(), nil
				}
				switch turn {
				case 1:
					os.MkdirAll(filepath.Join(root, "tests"), 0o755)
					return answer(toolCall("c", "create_file", map[string]any{"path": tc.path, "content": tc.content})), nil
				case 2:
					return answer(toolCall("t", "shell", map[string]any{"command": "true"})), nil
				}
				return answer(), nil
			})
			if status, err := a.Run(context.Background(), "go"); status != "completed" {
				t.Fatal(status, err)
			}
			if reminders != tc.want {
				t.Fatalf("reminders %d want %d", reminders, tc.want)
			}
		})
	}
}

func TestWritesTestDoublePatch(t *testing.T) {
	c := core.Call{}
	c.Function.Name = "apply_patch"
	b, _ := json.Marshal(map[string]any{"patchText": "*** Begin Patch\n*** Add File: pkg/pay_test.go\n+type fakeGateway struct{}\n*** End Patch"})
	c.Function.Arguments = string(b)
	if !writesTestDouble(c) {
		t.Fatal("patch fake not detected")
	}
}
