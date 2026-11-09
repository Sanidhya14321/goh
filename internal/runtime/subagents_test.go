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
	"github.com/Sanidhya14321/goh/internal/session"
)

func taskCall(id, kind, prompt, taskID string) core.Call {
	args := map[string]any{"description": "Fixture child task", "prompt": prompt, "subagent_type": kind}
	if taskID != "" {
		args["task_id"] = taskID
	}
	return toolCall(id, "task", args)
}
func TestSubagentSeparateHistoryContinuationAndOwnership(t *testing.T) {
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "target"), []byte("read child\n"), 0600)
	a.Grants["task"] = true
	childModels := 0
	firstChild := ""
	a.ChildModelFactory = func(c core.Config, s *session.Store) Model {
		childModels++
		if childModels == 1 {
			firstChild = s.ID
		} else if s.ID != firstChild {
			t.Fatal("continuation created new child")
		}
		turn := 0
		return modelFunc(func(_ context.Context, h []core.Message, defs []core.Tool, _ func(string)) (core.Reply, error) {
			turn++
			for _, d := range defs {
				if d.Function.Name == "task" {
					t.Fatal("recursive delegation advertised")
				}
			}
			if childModels == 2 {
				found := false
				for _, m := range h {
					found = found || m.Role == "user" && m.Content == "inspect target"
				}
				if !found {
					t.Fatal("child continuation lost prior history")
				}
				return core.Reply{Message: core.Message{Role: "assistant", Content: "Resumed findings verified."}, Finish: "stop"}, nil
			}
			if turn == 1 {
				return answer(toolCall("child-read", "read_file", map[string]any{"path": "target"})), nil
			}
			return core.Reply{Message: core.Message{Role: "assistant", Content: "Read target in child."}, Finish: "stop"}, nil
		})
	}
	parentTurn := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		parentTurn++
		if parentTurn == 1 {
			return answer(taskCall("create", "general", "inspect target", "")), nil
		}
		if parentTurn == 2 {
			r := toolResult(t, h[len(h)-1])
			if r["task_id"] != firstChild {
				t.Fatal(r)
			}
			return answer(taskCall("continue", "general", "continue inspection", firstChild)), nil
		}
		toolResult(t, h[len(h)-1])
		return answer(), nil
	})
	var events []core.Event
	a.Emit = func(e core.Event) { events = append(events, e) }
	status, e := a.Run(context.Background(), "parent only task")
	if e != nil || status != "completed" {
		t.Fatal(status, e)
	}
	terminal(t, events, status)
	if childModels != 2 || firstChild == a.Store.ID {
		t.Fatal("child identity reuse failed")
	}
	for _, m := range a.History {
		if m.Role == "user" && m.Content == "inspect target" {
			t.Fatal("child prompt leaked into parent history")
		}
	}
	child, e := session.Open(a.Config.SessionDir, firstChild)
	if e != nil {
		t.Fatal(e)
	}
	child.Close()
	if _, _, e = a.prepareTask(json.RawMessage(fmt.Sprintf(`{"description":"wrong","prompt":"p","subagent_type":"explore","task_id":%q}`, firstChild))); e == nil {
		t.Fatal("child type changed on resume")
	}
	other, _ := testAgent(t)
	other.ChildModelFactory = a.ChildModelFactory
	if _, _, e = other.prepareTask(json.RawMessage(fmt.Sprintf(`{"description":"wrong","prompt":"p","subagent_type":"general","task_id":%q}`, firstChild))); e == nil {
		t.Fatal("cross-parent child accepted")
	}
}
func TestSubagentDoesNotInheritMutationGrants(t *testing.T) {
	for _, childGrant := range []bool{false, true} {
		t.Run(fmt.Sprint(childGrant), func(t *testing.T) {
			a, root := testAgent(t)
			a.Grants["task"] = true
			a.Grants["create_file"] = true
			if childGrant {
				a.Grants["child:create_file"] = true
			}
			a.ChildModelFactory = func(core.Config, *session.Store) Model {
				turn := 0
				return modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
					turn++
					if turn == 1 {
						return answer(toolCall("write", "create_file", map[string]any{"path": "child-output", "content": "child created"})), nil
					}
					return answer(), nil
				})
			}
			parentTurn := 0
			a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
				parentTurn++
				if parentTurn == 1 {
					return answer(taskCall("task", "general", "create output", "")), nil
				}
				return answer(), nil
			})
			status, e := a.Run(context.Background(), "run child")
			if childGrant {
				if status != "completed" || e != nil {
					t.Fatal(status, e)
				}
				b, e := os.ReadFile(filepath.Join(root, "child-output"))
				if e != nil || string(b) != "child created" {
					t.Fatal(string(b), e)
				}
			} else {
				if status != "needs_permission" || e == nil || parentTurn != 1 {
					t.Fatal(status, e, parentTurn)
				}
				if _, e = os.Stat(filepath.Join(root, "child-output")); !os.IsNotExist(e) {
					t.Fatal("parent grant inherited")
				}
			}
		})
	}
}
func TestExploreAndParentPlanCapChildTools(t *testing.T) {
	for _, parentPlan := range []bool{false, true} {
		a, _ := testAgent(t)
		a.Grants["task"] = true
		if parentPlan {
			a.Config.AgentMode = "plan"
		}
		a.ChildModelFactory = func(c core.Config, _ *session.Store) Model {
			if c.AgentMode != "plan" || len(c.MCP) > 0 || len(c.LSP) > 0 {
				t.Fatal("child capability cap missing")
			}
			turn := 0
			return modelFunc(func(_ context.Context, h []core.Message, defs []core.Tool, _ func(string)) (core.Reply, error) {
				for _, d := range defs {
					if d.Function.Name == "shell" || d.Function.Name == "create_file" || d.Function.Name == "task" {
						t.Fatal("read-only child advertises unsafe tools")
					}
				}
				turn++
				if turn == 1 {
					return answer(toolCall("forged", "shell", map[string]any{"command": "exit 0"})), nil
				}
				if !strings.Contains(h[len(h)-1].Content, "read-only plan") {
					t.Fatal("forged child tool accepted")
				}
				return answer(), nil
			})
		}
		kind := "explore"
		if parentPlan {
			kind = "general"
		}
		raw := json.RawMessage(fmt.Sprintf(`{"description":"inspect","prompt":"read only","subagent_type":%q}`, kind))
		if result := a.executeTask(context.Background(), raw); result.Error {
			t.Fatal(result.Content)
		}
	}
}
func TestSubagentLimitsCancellationAndFreshReplayGrants(t *testing.T) {
	a, _ := testAgent(t)
	a.tasksRun = maxTasksPerRun
	if _, _, e := a.prepareTask(json.RawMessage(`{"description":"x","prompt":"p","subagent_type":"general"}`)); e == nil {
		t.Fatal("task count cap missing")
	}
	a.tasksRun = 0
	a.depth = 1
	if _, _, e := a.prepareTask(json.RawMessage(`{"description":"x","prompt":"p","subagent_type":"general"}`)); e == nil {
		t.Fatal("nested tasks accepted")
	}
	a.depth = 0
	ctx, cancel := context.WithCancel(context.Background())
	a.ChildModelFactory = func(core.Config, *session.Store) Model {
		return modelFunc(func(ctx context.Context, _ []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
			cancel()
			return core.Reply{}, ctx.Err()
		})
	}
	r := a.executeTask(ctx, json.RawMessage(`{"description":"x","prompt":"p","subagent_type":"general"}`))
	if !r.Error || !strings.Contains(r.Content, `"status":"interrupted"`) {
		t.Fatal(r)
	}
	resumed, e := New(a.Config, a.Store)
	if e != nil || len(resumed.Grants) > 0 {
		t.Fatal("resumed child authority", e)
	}
}

func TestSubagentEscapedSummaryRetainsCompletedIdentity(t *testing.T) {
	a, _ := testAgent(t)
	a.ChildModelFactory = func(core.Config, *session.Store) Model {
		return modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
			return core.Reply{Message: core.Message{Role: "assistant", Content: strings.Repeat("\x01", 40000)}, Finish: "stop"}, nil
		})
	}
	r := a.executeTask(context.Background(), json.RawMessage(`{"description":"bounded","prompt":"summarize","subagent_type":"general"}`))
	if r.Error || len(r.Content) > 64<<10 {
		t.Fatal(r.Error, len(r.Content))
	}
	var out map[string]any
	if e := json.Unmarshal([]byte(r.Content), &out); e != nil {
		t.Fatal(e)
	}
	if out["status"] != "completed" || out["task_id"] == "" || out["summary_truncated"] != true {
		t.Fatal(out)
	}
}
