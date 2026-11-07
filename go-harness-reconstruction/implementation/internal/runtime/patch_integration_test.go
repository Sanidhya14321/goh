package runtime

import (
	"context"
	"encoding/json"
	"example.com/independent-harness/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/session"
	"example.com/independent-harness/internal/tools"
)

func runtimePatch(lines ...string) string {
	return "*** Begin Patch\n" + strings.Join(lines, "\n") + "\n*** End Patch"
}
func TestActualPatchReadHashApprovalAndDurability(t *testing.T) {
	a, root := testAgent(t)
	before := "alpha\r\nomega\r\n"
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte(before), 0750); err != nil {
		t.Fatal(err)
	}
	approvals := 0
	a.Approve = func(_ context.Context, p core.Permission) (core.Decision, error) {
		approvals++
		if p.Tool != "apply_patch" || !strings.Contains(p.Preview, filepath.Join(root, "source.txt")) || !strings.Contains(p.Preview, filepath.Join(root, "created.txt")) || !strings.Contains(p.Preview, "-alpha") || !strings.Contains(p.Preview, "+BETA") {
			t.Fatal("patch approval incomplete", p)
		}
		return core.Decision{Allow: true}, nil
	}
	var events []core.Event
	a.Emit = func(e core.Event) { events = append(events, e) }
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, defs []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		switch turn {
		case 1:
			advertised := false
			for _, d := range defs {
				if d.Function.Name == "apply_patch" {
					advertised = true
				}
			}
			if !advertised {
				t.Fatal("patch not advertised")
			}
			return answer(toolCall("hash-source", "read_file", map[string]any{"path": "source.txt"})), nil
		case 2:
			read := toolResult(t, h[len(h)-1])
			return answer(toolCall("patch-effect", "apply_patch", map[string]any{"patchText": runtimePatch("*** Update File: source.txt", "@@", "-alpha", "+BETA", " omega", "*** Add File: created.txt", "+created"), "expected_sha256": map[string]any{"source.txt": read["sha256"]}})), nil
		case 3:
			result := toolResult(t, h[len(h)-1])
			if result["status"] != "completed" {
				t.Fatal(result)
			}
			return answer(), nil
		}
		t.Fatal("unexpected provider call")
		return core.Reply{}, nil
	})
	status, err := a.Run(context.Background(), "Read source, apply exact patch, create another file")
	if status != "completed" || err != nil || approvals != 1 {
		t.Fatal(status, err, approvals)
	}
	terminal(t, events, "completed")
	for p, want := range map[string]string{"source.txt": "BETA\r\nomega\r\n", "created.txt": "created\n"} {
		b, err := os.ReadFile(filepath.Join(root, p))
		if err != nil || string(b) != want {
			t.Fatal(p, string(b), err)
		}
	}
	st, _ := os.Stat(filepath.Join(root, "source.txt"))
	if testutil.UnixModes() && st.Mode().Perm() != 0750 {
		t.Fatal("mode lost")
	}
	if len(a.Store.Pending()) != 0 {
		t.Fatal(a.Store.Pending())
	}
	start, resultIndex := -1, -1
	for i, r := range a.Store.Records() {
		var d struct {
			ID string `json:"id"`
		}
		json.Unmarshal(r.Data, &d)
		if d.ID == "patch-effect" {
			if r.Type == "tool_start" {
				start = i
			}
			if r.Type == "tool_result" {
				resultIndex = i
			}
		}
	}
	if start < 0 || resultIndex <= start {
		t.Fatal("missing durable paired patch checkpoints", start, resultIndex)
	}
}
func TestActualPatchPermissionPoliciesAndPlanMode(t *testing.T) {
	for _, mode := range []string{"headless", "deny", "grant", "plan"} {
		t.Run(mode, func(t *testing.T) {
			a, root := testAgent(t)
			approvals := 0
			if mode == "deny" {
				a.Approve = func(context.Context, core.Permission) (core.Decision, error) {
					approvals++
					return core.Decision{}, nil
				}
			}
			if mode == "grant" || mode == "plan" {
				a.Grants["apply_patch"] = true
			}
			if mode == "plan" {
				a.Config.AgentMode = "plan"
				a.Approve = func(context.Context, core.Permission) (core.Decision, error) {
					t.Fatal("plan requested permission for patch")
					return core.Decision{Allow: true}, nil
				}
			}
			turn := 0
			a.Provider = modelFunc(func(_ context.Context, h []core.Message, defs []core.Tool, _ func(string)) (core.Reply, error) {
				turn++
				if mode == "plan" {
					for _, d := range defs {
						if d.Function.Name == "apply_patch" {
							t.Fatal("plan advertises patch")
						}
					}
				}
				if turn == 1 {
					return answer(toolCall("patch-policy", "apply_patch", map[string]any{"patchText": runtimePatch("*** Add File: target.txt", "+created")})), nil
				}
				if mode == "deny" || mode == "plan" {
					var r tools.Result
					json.Unmarshal([]byte(h[len(h)-1].Content), &r)
					if !r.Error {
						t.Fatal("forbidden patch reported success")
					}
				}
				return answer(), nil
			})
			status, err := a.Run(context.Background(), "create with patch")
			expected := "completed"
			if mode == "headless" {
				expected = "needs_permission"
				if err == nil {
					t.Fatal("headless permission missing")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if status != expected {
				t.Fatal(status, expected)
			}
			b, fileErr := os.ReadFile(filepath.Join(root, "target.txt"))
			if mode == "grant" {
				if fileErr != nil || string(b) != "created\n" {
					t.Fatal(string(b), fileErr)
				}
			} else if !os.IsNotExist(fileErr) {
				t.Fatal("patch violated authority", mode)
			}
			if mode == "deny" && approvals != 1 {
				t.Fatal(approvals)
			}
			if len(a.Store.Pending()) != 0 {
				t.Fatal("nonexecuted patch left pending", a.Store.Pending())
			}
		})
	}
}
func TestActualPatchSessionGrantIsNotRestoredOnResume(t *testing.T) {
	a, _ := testAgent(t)
	approvals := 0
	a.Approve = func(context.Context, core.Permission) (core.Decision, error) {
		approvals++
		return core.Decision{Allow: true, Session: true}, nil
	}
	turn := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		turn++
		if turn <= 2 {
			path := "one.txt"
			if turn == 2 {
				path = "two.txt"
			}
			return answer(toolCall(path, "apply_patch", map[string]any{"patchText": runtimePatch("*** Add File: "+path, "+created")})), nil
		}
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "create two files"); status != "completed" || err != nil || approvals != 1 {
		t.Fatal(status, err, approvals)
	}
	id := a.Store.ID
	a.Store.Close()
	store, err := session.Open(a.Config.SessionDir, id)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	resumed, err := New(a.Config, store)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Grants["apply_patch"] {
		t.Fatal("patch grant restored")
	}
	resumed.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return answer(toolCall("three", "apply_patch", map[string]any{"patchText": runtimePatch("*** Add File: three.txt", "+created")})), nil
	})
	if status, err := resumed.Run(context.Background(), "create third"); status != "needs_permission" || err == nil {
		t.Fatal("resume reused grant", status, err)
	}
}

type cancelWhenPatchPublished struct {
	context.Context
	root   string
	cancel context.CancelFunc
	once   sync.Once
}

func (c *cancelWhenPatchPublished) Err() error {
	if _, err := os.Stat(filepath.Join(c.root, "first.txt")); err == nil {
		c.once.Do(c.cancel)
	}
	return c.Context.Err()
}
func TestActualPartialPatchObservationPersistsAndNeverReplays(t *testing.T) {
	a, root := testAgent(t)
	a.Grants["apply_patch"] = true
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &cancelWhenPatchPublished{Context: base, root: root, cancel: cancel}
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return answer(toolCall("partial-patch", "apply_patch", map[string]any{"patchText": runtimePatch("*** Add File: first.txt", "+first", "*** Add File: second.txt", "+second")})), nil
	})
	var events []core.Event
	a.Emit = func(e core.Event) { events = append(events, e) }
	status, err := a.Run(ctx, "create two files")
	if status != "cancelled" || err == nil {
		t.Fatal(status, err)
	}
	terminal(t, events, "cancelled")
	if pending := a.Store.Pending(); len(pending) != 1 || pending[0] != "partial-patch" {
		t.Fatal(pending)
	}
	observationIndex, terminalIndex := -1, -1
	for i, record := range a.Store.Records() {
		if record.Type == "run_finished" {
			terminalIndex = i
		}
		if record.Type == "tool_observation" {
			var data struct {
				ID     string       `json:"id"`
				Result tools.Result `json:"result"`
			}
			if err := json.Unmarshal(record.Data, &data); err != nil {
				t.Fatal(err)
			}
			if data.ID != "partial-patch" || !data.Result.Error {
				t.Fatal(data)
			}
			var effects struct {
				Status    string   `json:"status"`
				Published []string `json:"published"`
			}
			if err := json.Unmarshal([]byte(data.Result.Content), &effects); err != nil || effects.Status != "interrupted" || len(effects.Published) != 1 || effects.Published[0] != "first.txt" {
				t.Fatal(effects, err)
			}
			observationIndex = i
		}
	}
	if observationIndex < 0 || terminalIndex <= observationIndex {
		t.Fatal("uncertain effects not durable before termination", observationIndex, terminalIndex)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "first.txt")); string(b) != "first\n" {
		t.Fatal("first effect missing")
	}
	if _, err := os.Stat(filepath.Join(root, "second.txt")); !os.IsNotExist(err) {
		t.Fatal("second effect executed after cancellation")
	}
	id := a.Store.ID
	a.Store.Close()
	store, err := session.Open(a.Config.SessionDir, id)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	resumed, err := New(a.Config, store)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	resumed.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		called = true
		return answer(), nil
	})
	if status, err := resumed.Run(context.Background(), "continue"); status != "failed" || err == nil || called {
		t.Fatal("pending patch bypassed recovery", status, err, called)
	}
	if err := store.Acknowledge(); err != nil {
		t.Fatal(err)
	}
	if err := resumed.RepairInterruptedHistory(); err != nil {
		t.Fatal(err)
	}
	if status, err := resumed.Run(context.Background(), "inspect the interrupted effect"); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	if _, err := os.Stat(filepath.Join(root, "second.txt")); !os.IsNotExist(err) {
		t.Fatal("interrupted patch replayed")
	}
	if b, _ := os.ReadFile(filepath.Join(root, "first.txt")); string(b) != "first\n" {
		t.Fatal("completed first effect changed on resume")
	}
}

func TestActualPatchApprovalCannotAuthorizeAnInterveningFileVersion(t *testing.T) {
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "source.txt"), []byte("old\n"), 0644)
	turn := 0
	a.Approve = func(context.Context, core.Permission) (core.Decision, error) {
		if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("old\nexternal\n"), 0644); err != nil {
			t.Fatal(err)
		}
		return core.Decision{Allow: true}, nil
	}
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		switch turn {
		case 1:
			return answer(toolCall("read-before-approval", "read_file", map[string]any{"path": "source.txt"})), nil
		case 2:
			source := toolResult(t, h[len(h)-1])
			return answer(toolCall("stale-approved-patch", "apply_patch", map[string]any{"patchText": runtimePatch("*** Add File: premature.txt", "+must not publish", "*** Update File: source.txt", "@@", "-old", "+new"), "expected_sha256": map[string]any{"source.txt": source["sha256"]}})), nil
		default:
			var r tools.Result
			json.Unmarshal([]byte(h[len(h)-1].Content), &r)
			if !r.Error || !strings.Contains(r.Content, "stale") {
				t.Fatal("intervening version not reported", r)
			}
			return answer(), nil
		}
	})
	if status, err := a.Run(context.Background(), "patch after approval"); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "source.txt"))
	if string(b) != "old\nexternal\n" {
		t.Fatal("external change overwritten")
	}
	if _, err := os.Stat(filepath.Join(root, "premature.txt")); !os.IsNotExist(err) {
		t.Fatal("earlier create published before stale validation")
	}
	if len(a.Store.Pending()) != 0 {
		t.Fatal("validation refusal incorrectly marked uncertain", a.Store.Pending())
	}
}
