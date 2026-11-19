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
