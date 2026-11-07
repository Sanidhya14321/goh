package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"example.com/independent-harness/internal/testutil"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/session"
	"example.com/independent-harness/internal/tools"
)

type modelFunc func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error)

func (f modelFunc) Complete(c context.Context, h []core.Message, d []core.Tool, emit func(string)) (core.Reply, error) {
	return f(c, h, d, emit)
}
func testAgent(t *testing.T) (*Agent, string) {
	t.Helper()
	root := testutil.TempDir(t)
	c := core.Defaults()
	c.SessionDir = t.TempDir()
	c.ContextWindow = 1 << 20
	c.Model = "fixture"
	s, err := session.New(c.SessionDir, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	a, err := New(c, s)
	if err != nil {
		t.Fatal(err)
	}
	return a, root
}
func toolCall(id, name string, args any) core.Call {
	b, _ := json.Marshal(args)
	c := core.Call{ID: id, Type: "function"}
	c.Function.Name = name
	c.Function.Arguments = string(b)
	return c
}
func answer(cs ...core.Call) core.Reply {
	finish := "stop"
	if len(cs) > 0 {
		finish = "tool_calls"
	}
	return core.Reply{Message: core.Message{Role: "assistant", Content: "fixture response", ToolCalls: cs}, Finish: finish}
}
func terminal(t *testing.T, events []core.Event, status string) {
	t.Helper()
	count := 0
	lastSeq := 0
	run := ""
	for _, e := range events {
		if e.Sequence != lastSeq+1 {
			t.Fatal("nonmonotonic events", e.Sequence, lastSeq)
		}
		lastSeq = e.Sequence
		if e.RunID == "" || e.SessionID == "" || e.Version != 1 {
			t.Fatal("missing envelope", e)
		}
		if run == "" {
			run = e.RunID
		} else if e.RunID != run {
			t.Fatal("run identity changed")
		}
		if e.Type == "run_finished" {
			count++
			b, _ := json.Marshal(e.Data)
			var got struct {
				Status string `json:"status"`
			}
			json.Unmarshal(b, &got)
			if got.Status != status {
				t.Fatal(got, status)
			}
		}
	}
	if count != 1 {
		t.Fatal("terminal count", count)
	}
}
func toolResult(t *testing.T, m core.Message) map[string]any {
	t.Helper()
	if m.Role != "tool" {
		t.Fatal("expected tool result", m)
	}
	var r tools.Result
	if err := json.Unmarshal([]byte(m.Content), &r); err != nil || r.Error {
		t.Fatal(r, err)
	}
	v := map[string]any{}
	if err := json.Unmarshal([]byte(r.Content), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func TestReadEditShellEndToEndAndResultOrder(t *testing.T) {
	testutil.RequireUnixShell(t)
	a, root := testAgent(t)
	if err := os.WriteFile(filepath.Join(root, "target"), []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}
	a.Grants["edit_file"] = true
	a.Grants["shell"] = true
	var events []core.Event
	a.Emit = func(e core.Event) { events = append(events, e) }
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		switch turn {
		case 1:
			return answer(toolCall("read", "read_file", map[string]any{"path": "target"}), toolCall("list", "list_files", map[string]any{})), nil
		case 2:
			if h[len(h)-2].ToolCallID != "read" || h[len(h)-1].ToolCallID != "list" {
				t.Fatal("concurrent result order changed", h)
			}
			read := toolResult(t, h[len(h)-2])
			return answer(toolCall("edit", "edit_file", map[string]any{"path": "target", "expected_sha256": read["sha256"], "old_text": "old", "new_text": "changed"})), nil
		case 3:
			toolResult(t, h[len(h)-1])
			return answer(toolCall("test", "shell", map[string]any{"command": "test \"$(cat target)\" = changed"})), nil
		case 4:
			got := toolResult(t, h[len(h)-1])
			if got["exit_code"] != float64(0) {
				t.Fatal(got)
			}
			return answer(), nil
		}
		t.Fatal("unexpected model call")
		return core.Reply{}, nil
	})
	status, err := a.Run(context.Background(), "Change target and verify it")
	if err != nil || status != "completed" {
		t.Fatal(status, err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "target"))
	if string(b) != "changed\n" {
		t.Fatal(string(b))
	}
	if len(a.Store.Pending()) != 0 {
		t.Fatal(a.Store.Pending())
	}
	terminal(t, events, "completed")
}
func TestHeadlessPermissionAndDenial(t *testing.T) {
	for _, headless := range []bool{true, false} {
		t.Run(fmt.Sprint(headless), func(t *testing.T) {
			a, root := testAgent(t)
			turn := 0
			var events []core.Event
			a.Emit = func(e core.Event) { events = append(events, e) }
			a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
				turn++
				if turn == 1 {
					return answer(toolCall("write", "create_file", map[string]any{"path": "new", "content": "effect"})), nil
				}
				return answer(), nil
			})
			if !headless {
				a.Approve = func(context.Context, core.Permission) (core.Decision, error) { return core.Decision{Allow: false}, nil }
			}
			status, err := a.Run(context.Background(), "create")
			expected := "completed"
			if headless {
				expected = "needs_permission"
				if err == nil {
					t.Fatal("missing permission error")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if status != expected {
				t.Fatal(status, expected)
			}
			if _, err := os.Stat(filepath.Join(root, "new")); !os.IsNotExist(err) {
				t.Fatal("permissionless mutation")
			}
			if len(a.Store.Pending()) != 0 {
				t.Fatal("nonexecuted tool marked started")
			}
			terminal(t, events, expected)
		})
	}
}
func TestSessionGrantIsExplicitAndNotRestored(t *testing.T) {
	a, _ := testAgent(t)
	approvals := 0
	a.Approve = func(context.Context, core.Permission) (core.Decision, error) {
		approvals++
		return core.Decision{Allow: true, Session: true}, nil
	}
	turn := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		turn++
		if turn < 3 {
			return answer(toolCall(fmt.Sprint(turn), "create_file", map[string]any{"path": fmt.Sprint(turn), "content": "ok"})), nil
		}
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "create twice"); err != nil || status != "completed" {
		t.Fatal(status, err)
	}
	if approvals != 1 || !a.Grants["create_file"] {
		t.Fatal(approvals, a.Grants)
	}
	resumed, err := New(a.Config, a.Store)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Grants["create_file"] {
		t.Fatal("grant restored from journal")
	}
}
func TestPartialProviderResponseCannotExecuteTools(t *testing.T) {
	a, root := testAgent(t)
	a.Grants["create_file"] = true
	var events []core.Event
	a.Emit = func(e core.Event) { events = append(events, e) }
	a.Provider = modelFunc(func(_ context.Context, _ []core.Message, _ []core.Tool, emit func(string)) (core.Reply, error) {
		emit("transient")
		return answer(toolCall("incomplete", "create_file", map[string]any{"path": "bad", "content": "bad"})), errors.New("stream interrupted")
	})
	status, err := a.Run(context.Background(), "create")
	if status != "failed" || err == nil {
		t.Fatal(status, err)
	}
	if _, err := os.Stat(filepath.Join(root, "bad")); !os.IsNotExist(err) {
		t.Fatal("partial tool executed")
	}
	for _, r := range a.Store.Records() {
		if r.Type == "tool_start" {
			t.Fatal("incomplete tool started")
		}
	}
	if len(a.History) != 2 {
		t.Fatal("incomplete assistant message committed", a.History)
	}
	terminal(t, events, "failed")
}
func TestTurnAndMalformedCallLimits(t *testing.T) {
	t.Run("turn", func(t *testing.T) {
		a, _ := testAgent(t)
		a.Config.MaxTurns = 1
		a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
			return answer(toolCall("read", "list_files", map[string]any{})), nil
		})
		status, err := a.Run(context.Background(), "list")
		if status != "budget_exhausted" || err == nil {
			t.Fatal(status, err)
		}
		if a.History[len(a.History)-1].Role != "tool" {
			t.Fatal("missing tool result before turn limit")
		}
	})
	t.Run("malformed", func(t *testing.T) {
		a, _ := testAgent(t)
		turn := 0
		a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
			turn++
			return answer(toolCall(fmt.Sprint(turn), "create_file", map[string]any{"path": "bad", "unexpected": true})), nil
		})
		status, err := a.Run(context.Background(), "malformed")
		if status != "failed" || err == nil || turn != 3 || !strings.Contains(err.Error(), "invalid tool call") {
			t.Fatal(status, err, turn)
		}
		if len(a.Store.Pending()) != 0 {
			t.Fatal(a.Store.Pending())
		}
	})
}
func TestInterruptedEffectsRequireAcknowledgmentAndNeverReplay(t *testing.T) {
	a, root := testAgent(t)
	ctx, cancel := context.WithCancel(context.Background())
	a.Grants["shell"] = true
	a.Emit = func(e core.Event) {
		if e.Type == "tool_started" {
			cancel()
		}
	}
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return answer(toolCall("uncertain", "shell", map[string]any{"command": "touch must-not-replay"})), nil
	})
	status, err := a.Run(ctx, "run")
	if status != "cancelled" || err == nil || len(a.Store.Pending()) != 1 {
		t.Fatal(status, err, a.Store.Pending())
	}
	a.Store.Close()
	s, err := session.Open(a.Config.SessionDir, a.Store.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	resumed, err := New(a.Config, s)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Grants["shell"] {
		t.Fatal("restored shell grant")
	}
	called := false
	resumed.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		called = true
		return answer(), nil
	})
	if status, err := resumed.Run(context.Background(), "continue"); status != "failed" || err == nil || called {
		t.Fatal(status, err, called)
	}
	if err = s.Acknowledge(); err != nil {
		t.Fatal(err)
	}
	if err = resumed.RepairInterruptedHistory(); err != nil {
		t.Fatal(err)
	}
	if status, err := resumed.Run(context.Background(), "continue"); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	if _, err := os.Stat(filepath.Join(root, "must-not-replay")); !os.IsNotExist(err) {
		t.Fatal("uncertain operation replayed")
	}
}

type externalFixture struct {
	result     tools.Result
	executions int
	closes     int
	closeErr   error
	execute    func()
}

func (*externalFixture) Definitions() []core.Tool {
	return []core.Tool{core.NewTool("mcp__fixture__effect", "fixture", map[string]any{"type": "object"})}
}
func (*externalFixture) Has(name string) bool                   { return name == "mcp__fixture__effect" }
func (*externalFixture) Validate(string, json.RawMessage) error { return nil }
func (f *externalFixture) Execute(context.Context, string, json.RawMessage) tools.Result {
	f.executions++
	if f.execute != nil {
		f.execute()
	}
	return f.result
}
func (f *externalFixture) Close() error { f.closes++; return f.closeErr }
func TestInterruptedMCPCallIsUncertainAndNotRetried(t *testing.T) {
	a, _ := testAgent(t)
	external := &externalFixture{result: tools.Result{Error: true, Content: `{"status":"interrupted","error":"server disconnected"}`}}
	a.External = external
	a.Grants["mcp__fixture__effect"] = true
	modelCalls := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		modelCalls++
		return answer(toolCall("effect", "mcp__fixture__effect", map[string]any{})), nil
	})
	var events []core.Event
	a.Emit = func(e core.Event) { events = append(events, e) }
	status, err := a.Run(context.Background(), "invoke")
	if status != "failed" || err == nil || external.executions != 1 || external.closes != 1 || modelCalls != 1 || len(a.Store.Pending()) != 1 {
		t.Fatal(status, err, external, modelCalls, a.Store.Pending())
	}
	terminal(t, events, "failed")
}
func TestCleanupIncompleteTakesPriorityOverCancellation(t *testing.T) {
	for _, phase := range []string{"tool", "teardown"} {
		t.Run(phase, func(t *testing.T) {
			a, _ := testAgent(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			external := &externalFixture{result: tools.Result{Error: true, Content: `{"status":"cleanup_incomplete"}`}, execute: cancel}
			if phase == "teardown" {
				external.result = tools.Result{Content: `{"ok":true}`}
				external.closeErr = errors.New("fixture cleanup did not settle")
			}
			a.External = external
			a.Grants["mcp__fixture__effect"] = true
			a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
				return answer(toolCall("effect", "mcp__fixture__effect", map[string]any{})), nil
			})
			var events []core.Event
			a.Emit = func(e core.Event) { events = append(events, e) }
			status, err := a.Run(ctx, "invoke")
			if status != "cleanup_incomplete" || err == nil || external.executions != 1 || external.closes != 1 {
				t.Fatal(status, err, external)
			}
			terminal(t, events, "cleanup_incomplete")
		})
	}
}
