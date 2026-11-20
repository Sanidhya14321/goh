package runtime

import (
	"fmt"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

func lastFinished(t *testing.T, a *Agent) map[string]any {
	t.Helper()
	var d map[string]any
	for _, r := range a.Store.Records() {
		if r.Type == "run_finished" {
			d = map[string]any{}
			json.Unmarshal(r.Data, &d)
		}
	}
	return d
}

func scripted(calls ...[]core.Call) Model {
	turn := 0
	return modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		if turn < len(calls) {
			turn++
			return answer(calls[turn-1]...), nil
		}
		return answer(), nil
	})
}

// Review P1: an unrelated successful command (pwd) after an edit is not a
// check, so the run stays unverified and the completion reminder fires.
func TestInspectionIsNotVerification(t *testing.T) {
	testutil.RequireUnixShell(t)
	a, _ := testAgent(t)
	a.Grants["create_file"], a.Grants["shell"] = true, true
	a.Provider = scripted(
		[]core.Call{toolCall("c", "create_file", map[string]any{"path": "new.txt", "content": "x\n"})},
		[]core.Call{toolCall("p", "shell", map[string]any{"command": "pwd"})},
	)
	if status, err := a.Run(context.Background(), "go"); status != "completed" {
		t.Fatal(status, err)
	}
	d := lastFinished(t, a)
	if d["outcome"] != OutcomeUnverified {
		t.Fatalf("pwd certified a change: %v", d)
	}
	if detail, _ := d["verification_detail"].(map[string]any); detail["evidence"] != nil || detail["change_tracking"] != TrackingKnown {
		t.Fatalf("detail %v", detail)
	}

	// The reminder fires when no recognized check ran.
	a2, _ := testAgent(t)
	a2.Config.CompletionCheck = nil
	a2.Grants["create_file"], a2.Grants["shell"] = true, true
	reminded := false
	turn := 0
	a2.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		if last := h[len(h)-1]; last.Automatic && strings.Contains(last.Content, "no test, build or other check") {
			reminded = true
			return answer(), nil
		}
		switch turn {
		case 1:
			return answer(toolCall("c", "create_file", map[string]any{"path": "new.txt", "content": "x\n"})), nil
		case 2:
			return answer(toolCall("p", "shell", map[string]any{"command": "pwd && ls"})), nil
		}
		return answer(), nil
	})
	a2.Run(context.Background(), "go")
	if !reminded {
		t.Fatal("completion reminder skipped after an inspection-only command")
	}
}

// A recognized check records evidence naming the command, kind and revision.
func TestShellCheckEvidence(t *testing.T) {
	testutil.RequireUnixShell(t)
	a, _ := testAgent(t)
	a.Grants["create_file"], a.Grants["shell"] = true, true
	a.Provider = scripted(
		[]core.Call{toolCall("c", "create_file", map[string]any{"path": "new.txt", "content": "x\n"})},
		[]core.Call{toolCall("t", "shell", map[string]any{"command": "test -f new.txt"})},
	)
	a.Run(context.Background(), "go")
	d := lastFinished(t, a)
	detail, _ := d["verification_detail"].(map[string]any)
	ev, _ := detail["evidence"].(map[string]any)
	if d["outcome"] != OutcomeVerified || ev["source"] != "shell" || ev["kind"] != "assert" || ev["command"] != "test -f new.txt" || ev["revision"] == "" {
		t.Fatalf("%v", d)
	}
}

func TestCheckKind(t *testing.T) {
	for cmd, want := range map[string]string{
		"pwd": "", "ls -la": "", "git status": "", "echo done": "", "cat a.go": "", "mkdir x": "",
		"go test ./...": "test", "cd x && pytest -k a": "test",
		"go build ./...": "build", "go vet ./...": "build", "npm run build": "build", "make": "build", "tsc --noEmit": "build",
		"test -f a.txt": "assert", "[ -f a ]": "assert", "diff a b": "assert", "grep -q foo a.txt": "assert", "grep foo a": "",
		"python3 main.py": "run", "go run .": "run", "./bin/tool --check": "run", "node --version": "", "FOO=1 python x.py": "run",
	} {
		if got := checkKind(cmd); got != want {
			t.Errorf("checkKind(%q) = %q want %q", cmd, got, want)
		}
	}
}

// The configured verifier journals its start before running and its
// settlement after, with evidence in run_finished.
func TestConfiguredVerifierJournal(t *testing.T) {
	testutil.RequireUnixShell(t)
	zero := 0
	a, _ := testAgent(t)
	a.Grants["create_file"] = true
	a.Config.Verify = &core.VerifyConfig{Command: "exit 3", RepairAttempts: &zero}
	a.Provider = scripted([]core.Call{toolCall("c", "create_file", map[string]any{"path": "w.txt", "content": "x\n"})})
	if status, _ := a.Run(context.Background(), "go"); status != "completed" {
		t.Fatal(status)
	}
	var order []string
	for _, r := range a.Store.Records() {
		if r.Type == "verification_started" || r.Type == "verification" || r.Type == "run_finished" {
			order = append(order, r.Type)
		}
	}
	if strings.Join(order, ",") != "verification_started,verification,run_finished" {
		t.Fatal(order)
	}
	d := lastFinished(t, a)
	ev, _ := d["verification_detail"].(map[string]any)["evidence"].(map[string]any)
	if d["outcome"] != OutcomeFailedVerification || ev["source"] != "configured" || ev["passed"] != false || ev["exit_code"] != float64(3) {
		t.Fatalf("%v", d)
	}
}

// Review: each operation starts from a clean outcome. A failing model run,
// then a typed shell write, then a compaction, then a new prompt, each
// report only their own result; the typed command never reaches the model.
func TestOperationIsolation(t *testing.T) {
	testutil.RequireUnixShell(t)
	zero := 0
	a, root := testAgent(t)
	a.Grants["create_file"] = true
	a.Config.Verify = &core.VerifyConfig{Command: "exit 1", RepairAttempts: &zero}
	a.Config.AutoTitle = true
	modelCalls := 0
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		modelCalls++
		turn++
		if turn == 1 {
			return answer(toolCall("c", "create_file", map[string]any{"path": "w.txt", "content": "x\n"})), nil
		}
		return answer(), nil
	})
	a.Run(context.Background(), "make a change")
	if got := lastFinished(t, a)["outcome"]; got != OutcomeFailedVerification {
		t.Fatal(got)
	}
	before := modelCalls
	if status, err := a.RunShell(context.Background(), "echo hi > typed.txt"); status != "completed" {
		t.Fatal(status, err)
	}
	if _, err := os.Stat(filepath.Join(root, "typed.txt")); err != nil {
		t.Fatal(err)
	}
	d := lastFinished(t, a)
	if d["outcome"] != OutcomeUnverified || d["operation"] != "user_shell" {
		t.Fatalf("typed write: %v", d)
	}
	if modelCalls != before {
		t.Fatal("typed !command made a model request (auto-title)")
	}
	a.RunShell(context.Background(), "pwd")
	if got := lastFinished(t, a)["outcome"]; got != OutcomeNoChanges {
		t.Fatalf("typed read inherited %v", got)
	}
	a.Config.Verify = nil
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return answer(), nil
	})
	a.Run(context.Background(), "just answer")
	if got := lastFinished(t, a)["outcome"]; got != OutcomeNoChanges {
		t.Fatalf("new prompt inherited %v", got)
	}
}

// The run_finished hook receives the settled outcome, and run_finished is
// the last public event of the run.
func TestRunFinishedHookOrder(t *testing.T) {
	testutil.RequireUnixShell(t)
	a, root := testAgent(t)
	a.Grants["create_file"] = true
	out := filepath.Join(root, "hook.json")
	a.Config.Hooks = []core.HookConfig{{Event: "run_finished", Command: "/bin/sh", Args: []string{"-c", "cat > " + out}}}
	var events []string
	a.Emit = func(e core.Event) { events = append(events, e.Type) }
	a.Provider = scripted([]core.Call{toolCall("c", "create_file", map[string]any{"path": "w.txt", "content": "x\n"})})
	a.Run(context.Background(), "go")
	if events[len(events)-1] != "run_finished" {
		t.Fatalf("last event %v", events[len(events)-3:])
	}
	b, _ := os.ReadFile(out)
	if !strings.Contains(string(b), `"outcome":"unverified"`) {
		t.Fatalf("hook payload %s", b)
	}
}

// Auto-title reads the typed prompt, not @file contents.
func TestAutoTitleUsesTypedPrompt(t *testing.T) {
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "secret.txt"), []byte("FILE-BODY-MARKER"), 0o600)
	a.Config.AutoTitle = true
	var titleInput string
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, tools []core.Tool, _ func(string)) (core.Reply, error) {
		if len(h) == 2 && h[0].Role == "system" && tools == nil {
			titleInput = h[1].Content
			return core.Reply{Message: core.Message{Role: "assistant", Content: "Title"}, Finish: "stop"}, nil
		}
		return answer(), nil
	})
	a.Run(context.Background(), "summarize @secret.txt")
	if titleInput == "" || strings.Contains(titleInput, "FILE-BODY-MARKER") {
		t.Fatalf("title input %q", titleInput)
	}
}

// Follow-up review P1: a command that only mentions a test runner, and a
// test run that executed no tests, do not verify a change.
func TestMentionsAndEmptyRunsAreNotChecks(t *testing.T) {
	for cmd, want := range map[string]string{
		`echo "now running go test ./..."`:     "",
		`printf 'pytest -q\n'`:                 "",
		`echo done && cd pkg && go test ./...`: "full",
		`timeout 60 go test -run TestA ./x`:    "selected",
		`uv run pytest -q`:                     "full",
		`node_modules/.bin/jest`:               "full",
		`grep -r "go test" Makefile`:           "",
	} {
		if got := testScope(cmd); got != want {
			t.Errorf("testScope(%q) = %q want %q", cmd, got, want)
		}
	}
	for out, want := range map[string]bool{
		"?   \tex/p\t[no test files]\n":                          true,
		"ok  \tex/p\t0.01s [no tests to run]\n":                  true,
		"ok  \tex/p\t0.01s\n?   \tex/q\t[no test files]\n":       false,
		"\n----\nRan 0 tests in 0.000s\n\nNO TESTS RAN\n":        true,
		"collected 0 items\n\n==== no tests ran in 0.01s ====\n": true,
		"Ran 3 tests in 0.01s\n\nOK\n":                           false,
	} {
		if got := ranNoTests(out); got != want {
			t.Errorf("ranNoTests(%q) = %v want %v", out, got, want)
		}
	}
	testutil.RequireUnixShell(t)
	for name, command := range map[string]string{
		"mention": `echo "running go test ./... now"`,
		"empty":   `printf 'Ran 0 tests in 0.000s\n\nOK\n' >&2; ` + "PATH=./stub:$PATH go test ./...",
	} {
		t.Run(name, func(t *testing.T) {
			a, root := testAgent(t)
			os.MkdirAll(filepath.Join(root, "stub"), 0o755)
			os.WriteFile(filepath.Join(root, "stub", "go"), []byte("#!/bin/sh\necho '?   \tex/p\t[no test files]'\n"), 0o755)
			a.Grants["create_file"], a.Grants["shell"] = true, true
			a.Provider = scripted(
				[]core.Call{toolCall("c", "create_file", map[string]any{"path": "new.txt", "content": "x\n"})},
				[]core.Call{toolCall("t", "shell", map[string]any{"command": command})},
			)
			a.Run(context.Background(), "go")
			if got := lastFinished(t, a)["outcome"]; got != OutcomeUnverified {
				t.Fatalf("outcome %v", got)
			}
		})
	}
}

// Follow-up review P1: a configured check that changes the worktree passes
// only for the revision it leaves, after re-checking it once.
func TestVerifierThatChangesFiles(t *testing.T) {
	testutil.RequireUnixShell(t)
	for name, tc := range map[string]struct {
		command, want string
	}{
		"keeps rewriting": {`date +%s%N >> stamp.txt`, OutcomeUnverified},
		"formats once":    {`grep -q formatted w.txt || printf 'formatted\n' > w.txt`, OutcomeVerified},
	} {
		t.Run(name, func(t *testing.T) {
			zero := 0
			a, _ := testAgent(t)
			a.Grants["create_file"] = true
			a.Config.Verify = &core.VerifyConfig{Command: tc.command, RepairAttempts: &zero}
			a.Provider = scripted([]core.Call{toolCall("c", "create_file", map[string]any{"path": "w.txt", "content": "raw\n"})})
			a.Run(context.Background(), "go")
			if got := lastFinished(t, a)["outcome"]; got != tc.want {
				t.Fatalf("outcome %v want %v", got, tc.want)
			}
			if len(a.Store.Pending()) != 0 {
				t.Fatal("settled verifier left pending")
			}
		})
	}
}

// Follow-up review P1: an interrupted verifier is a pending effect: the run
// ends without further model work and the next one needs acknowledgement.
func TestInterruptedVerifierIsPending(t *testing.T) {
	testutil.RequireUnixShell(t)
	a, root := testAgent(t)
	a.Grants["create_file"] = true
	a.Config.Verify = &core.VerifyConfig{Command: "touch started && sleep 30"}
	modelCalls := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		modelCalls++
		if modelCalls == 1 {
			return answer(toolCall("c", "create_file", map[string]any{"path": "w.txt", "content": "x\n"})), nil
		}
		return answer(), nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for i := 0; i < 500; i++ {
			if _, err := os.Stat(filepath.Join(root, "started")); err == nil {
				cancel()
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	status, _ := a.Run(ctx, "go")
	if status != "cancelled" || modelCalls != 2 {
		t.Fatalf("status %s, model calls %d", status, modelCalls)
	}
	pending := a.Store.Pending()
	if len(pending) != 1 || !strings.HasPrefix(pending[0], "verify_") {
		t.Fatalf("pending %v", pending)
	}
	if st, err := a.Run(context.Background(), "again"); st != "failed" || err == nil || !strings.Contains(err.Error(), "acknowledge") || modelCalls != 2 {
		t.Fatalf("next run admitted: %s %v", st, err)
	}
}

// Follow-up review: with the token allowance spent, a large tool result
// does not buy a compaction (summary) call before the run stops.
func TestBudgetCheckedBeforeCompaction(t *testing.T) {
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "small.txt"), []byte("small\n"), 0o600)
	os.WriteFile(filepath.Join(root, "big.txt"), []byte(strings.Repeat("data line number with some words\n", 6000)), 0o600)
	a.Config.ContextWindow = 131072
	a.Config.CompactionThreshold = 5
	a.Config.MaxRunTokens = 100
	turns, summaries := 0, 0
	_ = summaries
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, tools []core.Tool, _ func(string)) (core.Reply, error) {
		if tools == nil {
			summaries++
			return core.Reply{Message: core.Message{Role: "assistant", Content: "summary"}, Finish: "stop", Usage: &core.Usage{Prompt: 10, Completion: 1, Total: 11}}, nil
		}
		turns++
		path := "small.txt"
		if turns >= 3 {
			path = "big.txt"
		}
		r := answer(toolCall(fmt.Sprint("r", turns), "read_file", map[string]any{"path": path}))
		r.Usage = &core.Usage{Completion: 34, Total: 34} // no prompt count: the size estimate stays conservative
		return r, nil
	})
	compactions := 0
	a.Emit = func(e core.Event) {
		// Only compaction after the third turn spent the allowance counts.
		if (e.Type == "context_compacted" || e.Type == "context_compaction_skipped") && turns >= 3 {
			compactions++
		}
	}
	status, err := a.Run(context.Background(), "read them")
	if status != "budget_exhausted" || compactions != 0 {
		t.Fatalf("status %s, turns %d, summary calls %d, compactions %d (%v)", status, turns, summaries, compactions, err)
	}
}
