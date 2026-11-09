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
	"github.com/Sanidhya14321/goh/internal/tools"
)

func writeHook(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func rawResult(t *testing.T, m core.Message) tools.Result {
	t.Helper()
	var r tools.Result
	if m.Role != "tool" || json.Unmarshal([]byte(m.Content), &r) != nil {
		t.Fatal("expected tool result", m)
	}
	return r
}

func TestHooksBlockAddFeedbackAndNotify(t *testing.T) {
	testutil.RequireUnixShell(t)
	t.Setenv("OPENAI_API_KEY", "sk-hook-must-not-see")
	a, _ := testAgent(t)
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	pre := writeHook(t, dir, "pre", `input=$(cat); echo "pre $GOH_TOOL ${OPENAI_API_KEY:-nokey}" >> `+log+`
case "$input" in *'rm -rf'*) echo "no rm -rf here" >&2; exit 2;; esac; exit 0`)
	post := writeHook(t, dir, "post", `cat >/dev/null; echo "lint: 1 warning"; exit 2`)
	done := writeHook(t, dir, "done", `cat > `+filepath.Join(dir, "finished.json"))
	a.Config.Hooks = []core.HookConfig{
		{Event: "pre_tool", Tools: []string{"sh*"}, Command: pre},
		{Event: "post_tool", Tools: []string{"shell"}, Command: post},
		{Event: "post_tool", Tools: []string{"read_file"}, Command: "/nonexistent/never-runs"},
		{Event: "run_finished", Command: done},
	}
	a.Grants["shell"] = true
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		switch turn {
		case 1:
			return answer(toolCall("bad", "shell", map[string]any{"command": "rm -rf important"})), nil
		case 2:
			r := rawResult(t, h[len(h)-1])
			if !r.Error || !strings.Contains(r.Content, "no rm -rf here") {
				t.Fatalf("pre_tool hook did not block: %+v", r)
			}
			return answer(toolCall("ok", "shell", map[string]any{"command": "echo hi"})), nil
		case 3:
			v := toolResult(t, h[len(h)-1])
			fb, _ := json.Marshal(v["hook_feedback"])
			if !strings.Contains(string(fb), "lint: 1 warning") || !strings.Contains(v["output"].(string), "hi") {
				t.Fatalf("post_tool feedback missing: %v", v)
			}
			return answer(), nil
		}
		t.Fatal("unexpected model call")
		return core.Reply{}, nil
	})
	status, err := a.Run(context.Background(), "try things")
	if err != nil || status != "completed" {
		t.Fatal(status, err)
	}
	b, _ := os.ReadFile(log)
	if got := string(b); got != "pre shell nokey\npre shell nokey\n" {
		t.Fatalf("hook log %q (secret leaked or hook skipped)", got)
	}
	var fin map[string]any
	b, _ = os.ReadFile(filepath.Join(dir, "finished.json"))
	if json.Unmarshal(b, &fin) != nil || fin["status"] != "completed" || fin["event"] != "run_finished" {
		t.Fatalf("run_finished payload %s", b)
	}
}

func TestHooksCannotGrantAndFailClosed(t *testing.T) {
	testutil.RequireUnixShell(t)
	a, root := testAgent(t)
	dir := t.TempDir()
	marker := filepath.Join(dir, "ran")
	allow := writeHook(t, dir, "allow", "touch "+marker+"; exit 0")
	a.Config.Hooks = []core.HookConfig{{Event: "pre_tool", Command: allow}}
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		return answer(toolCall("x", "shell", map[string]any{"command": "touch made"})), nil
	})
	// No shell grant: an allowing hook must not let the command run.
	if status, _ := a.Run(context.Background(), "run it"); status != "needs_permission" {
		t.Fatalf("status %s", status)
	}
	if _, err := os.Stat(filepath.Join(root, "made")); err == nil {
		t.Fatal("hook granted a permission")
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("pre_tool hook ran before approval")
	}

	b, root2 := testAgent(t)
	broken := writeHook(t, dir, "broken", "exit 1")
	b.Config.Hooks = []core.HookConfig{{Event: "pre_tool", Command: broken}}
	b.Grants["shell"] = true
	turn := 0
	b.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		if turn++; turn == 1 {
			return answer(toolCall("y", "shell", map[string]any{"command": "touch made"})), nil
		}
		if r := rawResult(t, h[len(h)-1]); !r.Error || !strings.Contains(r.Content, "hook failed") {
			t.Fatalf("failing hook did not stop the tool: %+v", r)
		}
		return answer(), nil
	})
	if status, err := b.Run(context.Background(), "run it"); status != "completed" {
		t.Fatal(status, err)
	}
	if _, err := os.Stat(filepath.Join(root2, "made")); err == nil {
		t.Fatal("tool ran although its hook failed")
	}
}

func TestValidateHooks(t *testing.T) {
	bad := [][]core.HookConfig{
		{{Event: "before", Command: "x"}},
		{{Event: "pre_tool"}},
		{{Event: "pre_tool", Command: "x", Timeout: 500}},
		{{Event: "run_finished", Command: "x", Tools: []string{"shell"}}},
		{{Event: "pre_tool", Command: "x", Tools: []string{"["}}},
	}
	for _, h := range bad {
		if core.ValidateHooks(h) == nil {
			t.Fatal("accepted", h)
		}
	}
	if err := core.ValidateHooks([]core.HookConfig{{Event: "post_tool", Command: "x", Tools: []string{"mcp_*"}, Timeout: 30}}); err != nil {
		t.Fatal(err)
	}
}
