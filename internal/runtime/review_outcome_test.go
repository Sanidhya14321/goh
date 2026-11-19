package runtime

import (
	"context"
	"encoding/json"
	"fmt"
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
	shellEdit := toolCall("s", "shell", map[string]any{"command": "echo x > made.txt"})
	checkShell := toolCall("u", "shell", map[string]any{"command": "test -f made.txt"})
	cacheOnly := toolCall("p", "shell", map[string]any{"command": "mkdir -p __pycache__ && echo x > __pycache__/a.pyc"})
	for name, tc := range map[string]struct {
		calls [][]core.Call
		want  string
	}{
		"no changes": {nil, OutcomeNoChanges},
		"unverified": {[][]core.Call{{create}}, OutcomeUnverified},
		"verified":   {[][]core.Call{{create}, {check}}, OutcomeVerified},
		// Edits made through the shell count as changes too.
		"shell edit":         {[][]core.Call{{shellEdit}}, OutcomeUnverified},
		"shell edit checked": {[][]core.Call{{shellEdit}, {checkShell}}, OutcomeVerified},
		// Test caches are not changes.
		"cache only": {[][]core.Call{{cacheOnly}}, OutcomeNoChanges},
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
	for _, m := range a.History {
		if m.Role == "assistant" && strings.TrimSpace(m.Content) == "" && len(m.ToolCalls) == 0 {
			t.Fatal("empty assistant reply was committed to history")
		}
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
		"role stand-in":  {"tests/test_pay.py", "class IdempotentGateway:\n    def charge(self): pass\n", 1},
		"mock library":   {"tests/test_pay.py", "from unittest.mock import Mock\n", 1},
		"go stand-in":    {"pay/pay_test.go", "package pay\n\ntype recordingClient struct{}\n", 1},
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
					os.MkdirAll(filepath.Join(root, filepath.Dir(tc.path)), 0o755)
					return answer(toolCall("c", "create_file", map[string]any{"path": tc.path, "content": tc.content})), nil
				case 2:
					return answer(toolCall("t", "shell", map[string]any{"command": "test -f " + tc.path})), nil
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

func TestTestScope(t *testing.T) {
	for command, want := range map[string]string{
		"go test ./...":                           "full",
		"go test -count=1 ./internal/...":         "full",
		"cd pkg && go test -run TestAdd .":        "selected",
		"go test -run=TestParse ./parser":         "selected",
		"go test -v -run 'TestA|TestB' ./x":       "selected",
		"pytest":                                  "full",
		"pytest -k add tests/":                    "selected",
		"pytest tests/test_math.py::test_add":     "selected",
		"python -m pytest -q":                     "full",
		"python3 -m unittest discover":            "full",
		"npx jest -t 'adds numbers'":              "selected",
		"npm test":                                "full",
		"npm run test -- --testNamePattern add":   "selected",
		"cargo test":                              "full",
		"cargo test parse_duration":               "selected",
		"cargo test -- --nocapture":               "full",
		"go build ./... && go vet ./...":          "",
		"ls -la":                                  "",
		"grep -rn 'go test -run' docs":            "",
		"mvn test -Dtest=AddTest":                 "selected",
		"./gradlew test --tests '*AddTest'":       "selected",
		"go test ./... -run TestOnlyOne -count=1": "selected",
	} {
		if got := testScope(command); got != want {
			t.Errorf("%q: %q, want %q", command, got, want)
		}
	}
}

// A run whose only passing tests after a change were picked by name gets
// one reminder to run the full tests; a full run afterwards, or from the
// start, needs none.
func TestCompletionCheckSelectedTests(t *testing.T) {
	testutil.RequireUnixShell(t)
	for name, tc := range map[string]struct {
		checks []string
		want   int
	}{
		"selected only":        {[]string{stubGo + "go test -run TestAdd ./..."}, 1},
		"selected then full":   {[]string{stubGo + "go test -run TestAdd ./...", stubGo + "go test ./..."}, 0},
		"full only":            {[]string{stubGo + "go test ./..."}, 0},
		"build after selected": {[]string{stubGo + "go test -run TestAdd ./...", stubGo + "go build ./..."}, 1},
		"non-test check":       {[]string{stubGo + "go build ./..."}, 0},
	} {
		t.Run(name, func(t *testing.T) {
			a, root := testAgent(t)
			writeStubGo(t, root)
			a.Config.CompletionCheck = nil
			a.Grants["create_file"], a.Grants["shell"] = true, true
			reminders, turn := 0, 0
			a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
				turn++
				if last := h[len(h)-1]; last.Role == "user" && strings.Contains(last.Content, "picked by name") {
					reminders++
					return answer(), nil
				}
				if turn == 1 {
					return answer(toolCall("c", "create_file", map[string]any{"path": "fix.go", "content": "package fix\n"})), nil
				}
				if i := turn - 2; i < len(tc.checks) {
					// A stub go on PATH passes without running a real toolchain.
					return answer(toolCall(fmt.Sprint("t", i), "shell", map[string]any{"command": tc.checks[i]})), nil
				}
				return answer(), nil
			})
			if status, err := a.Run(context.Background(), "go"); status != "completed" {
				t.Fatal(status, err)
			}
			if reminders != tc.want {
				t.Fatalf("reminders %d want %d", reminders, tc.want)
			}
			if lastOutcome(t, a) != OutcomeVerified {
				t.Fatal(lastOutcome(t, a))
			}
		})
	}
}

// stubGo puts the project's stub/go first on PATH: a fake runner that passes
// and reports a package that ran tests.
const stubGo = "PATH=./stub:$PATH "

func writeStubGo(t *testing.T, root string) {
	t.Helper()
	os.MkdirAll(filepath.Join(root, "stub"), 0o755)
	if err := os.WriteFile(filepath.Join(root, "stub", "go"), []byte("#!/bin/sh\necho 'ok  \tfix\t0.01s'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
