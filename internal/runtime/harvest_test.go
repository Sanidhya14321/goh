package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/tools"
)

func TestStructuredChildResultAndEffectsNeverReplayed(t *testing.T) {
	for _, mode := range []string{"valid", "invalid", "fenced", "oversize"} {
		t.Run(mode, func(t *testing.T) {
			a, root := testAgent(t)
			a.Grants["task"] = true
			a.Grants["child:create_file"] = true
			children, childTurns := 0, 0
			a.ChildModelFactory = func(core.Config, *session.Store) Model {
				children++
				return modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
					childTurns++
					if !strings.Contains(h[1].Content, "caller-supplied data schema") {
						t.Fatal("missing schema guidance")
					}
					if childTurns == 1 {
						return answer(toolCall("child-write", "create_file", map[string]any{"path": "verified", "content": "one effect"})), nil
					}
					text := `{"ok":true,"count":1}`
					switch mode {
					case "invalid":
						text = `{"ok":"yes","count":1}`
					case "fenced":
						text = "```json\n" + text + "\n```"
					case "oversize":
						text = strings.Repeat("x", 32<<10)
					}
					return core.Reply{Message: core.Message{Role: "assistant", Content: text}, Finish: "stop"}, nil
				})
			}
			parentTurns := 0
			var result map[string]any
			var failed bool
			a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
				parentTurns++
				if parentTurns == 1 {
					return answer(toolCall("typed", "task", map[string]any{"description": "Typed task", "prompt": "Write once and report", "subagent_type": "general", "output_schema": map[string]any{"type": "object", "properties": map[string]any{"ok": map[string]any{"type": "boolean"}, "count": map[string]any{"type": "integer"}}, "required": []string{"ok", "count"}, "additionalProperties": false}})), nil
				}
				var envelope tools.Result
				if err := json.Unmarshal([]byte(h[len(h)-1].Content), &envelope); err != nil {
					t.Fatal(err)
				}
				failed = envelope.Error
				if err := json.Unmarshal([]byte(envelope.Content), &result); err != nil {
					t.Fatal(err)
				}
				return answer(), nil
			})
			status, err := a.Run(context.Background(), "Use one typed child")
			if err != nil || status != "completed" {
				t.Fatal(status, err)
			}
			if children != 1 || childTurns != 2 || result["child_status"] != "completed" || result["output_schema_checked"] != true {
				t.Fatal(children, childTurns, result)
			}
			if mode == "valid" {
				if failed || result["status"] != "completed" || result["structured_output"].(map[string]any)["ok"] != true {
					t.Fatal(result)
				}
			} else if !failed || result["status"] != "invalid_output" || result["structured_output"] != nil {
				t.Fatal(result)
			}
			b, err := os.ReadFile(filepath.Join(root, "verified"))
			if err != nil || string(b) != "one effect" || len(a.Store.Pending()) != 0 {
				t.Fatal(string(b), err)
			}
			child, err := session.Open(a.Config.SessionDir, result["task_id"].(string))
			if err != nil {
				t.Fatal(err)
			}
			if len(child.Pending()) != 0 {
				t.Fatal(child.Pending())
			}
			child.Close()
		})
	}
}
func TestInvalidTaskSchemaBeforeDispatch(t *testing.T) {
	a, _ := testAgent(t)
	a.Grants["task"] = true
	dispatches := 0
	a.ChildModelFactory = func(core.Config, *session.Store) Model { dispatches++; return &summaryModel{summary: "unused"} }
	for _, schema := range []any{nil, false, map[string]any{"$ref": "http://127.0.0.1:1/schema"}, map[string]any{"type": "unknown"}} {
		call := toolCall("bad", "task", map[string]any{"description": "Bad schema", "prompt": "do not run", "subagent_type": "general", "output_schema": schema})
		if _, _, err := a.prepare("task", json.RawMessage(call.Function.Arguments)); err == nil {
			t.Fatal("schema accepted", schema)
		}
		if result := a.execute(context.Background(), call); !result.Error {
			t.Fatal(result)
		}
	}
	if dispatches != 0 || len(a.children()) != 0 {
		t.Fatal("invalid schema created a child", dispatches)
	}
}
func TestTaskRolesCustomPrecedenceAndPlanCap(t *testing.T) {
	a, _ := testAgent(t)
	str := func(s string) *string { return &s }
	a.Config.Profiles = map[string]core.ModelProfile{"task": {Model: str("task-model")}, "explore": {Model: str("explore-model")}, "custom": {Model: str("custom-model")}}
	a.Config.ModelRoles = map[string]string{"task": "task", "explore": "explore"}
	a.Config.Agents = map[string]core.AgentProfile{"custom": {Mode: "build", Profile: "custom"}, "inherited": {Mode: "build"}}
	for kind, want := range map[string]string{"general": "task-model", "explore": "explore-model", "custom": "custom-model", "inherited": "task-model"} {
		config, err := a.childConfiguration(kind)
		if err != nil || config.Model != want || config.SessionDir != a.Config.SessionDir {
			t.Fatal(kind, config, err)
		}
	}
	a.Config.AgentMode = "plan"
	config, err := a.childConfiguration("custom")
	if err != nil || config.AgentMode != "plan" {
		t.Fatal(config, err)
	}
	delete(a.Config.ModelRoles, "explore")
	config, err = a.childConfiguration("explore")
	if err != nil || config.Model != "task-model" || config.AgentMode != "plan" {
		t.Fatal(config, err)
	}
}
func TestCompactionModelRoleAndContextAdmission(t *testing.T) {
	for _, small := range []bool{false, true} {
		t.Run(map[bool]string{false: "sufficient", true: "insufficient"}[small], func(t *testing.T) {
			a := contextAgent(t)
			a.Config.Endpoint = "http://fixture.invalid/not-contacted"
			a.Config.ContextWindow = 131072
			for _, m := range []core.Message{{Role: "user", Content: "original task"}, {Role: "assistant", Content: strings.Repeat("historical facts ", 2000)}, {Role: "user", Content: "current task"}, {Role: "assistant", Content: "recent one"}, {Role: "assistant", Content: "recent two"}} {
				if err := a.commit(m); err != nil {
					t.Fatal(err)
				}
			}
			model := "compact-model"
			window := 131072
			if small {
				window = 8192
			}
			a.Config.Profiles = map[string]core.ModelProfile{"compact": {Model: &model, ContextWindow: &window}}
			a.Config.ModelRoles = map[string]string{"compaction": "compact"}
			calls := 0
			a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
				t.Fatal("parent model used for compaction")
				return core.Reply{}, nil
			})
			a.RoleModelFactory = func(config core.Config, role string) Model {
				if role != "compaction" || config.Model != model || config.Profile != "compact" {
					t.Fatal(config, role)
				}
				return modelFunc(func(_ context.Context, h []core.Message, defs []core.Tool, _ func(string)) (core.Reply, error) {
					calls++
					if len(defs) != 0 || !strings.Contains(h[1].Content, "original task") {
						t.Fatal("summary request lost task")
					}
					return core.Reply{Message: core.Message{Role: "assistant", Content: "Historical task state."}, Finish: "stop"}, nil
				})
			}
			before := append([]core.Message{}, a.History...)
			status, err := a.Compact(context.Background())
			if small {
				if err == nil || status != "budget_exhausted" || calls != 0 || !reflect.DeepEqual(before, a.History) {
					t.Fatal(status, err, calls)
				}
			} else if err != nil || status != "completed" || calls != 1 || !strings.Contains(a.History[1].Content, "Historical task state") {
				t.Fatal(status, err, calls)
			}
		})
	}
}
