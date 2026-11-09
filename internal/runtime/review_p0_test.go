package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

// Review P0: post_tool hooks on parallel read batches raced on event state.
func TestHooksOnParallelReadsAreRaceFree(t *testing.T) {
	testutil.RequireUnixShell(t)
	a, root := testAgent(t)
	for _, n := range []string{"a", "b", "c", "d"} {
		os.WriteFile(filepath.Join(root, n), []byte(n), 0o644)
	}
	hook := writeHook(t, t.TempDir(), "post", "cat >/dev/null; exit 0")
	a.Config.Hooks = []core.HookConfig{{Event: "post_tool", Command: hook}}
	var events []core.Event
	a.Emit = func(e core.Event) { events = append(events, e) }
	turn := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		if turn++; turn == 1 {
			return answer(toolCall("1", "read_file", map[string]any{"path": "a"}), toolCall("2", "read_file", map[string]any{"path": "b"}),
				toolCall("3", "read_file", map[string]any{"path": "c"}), toolCall("4", "read_file", map[string]any{"path": "d"})), nil
		}
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "read four"); status != "completed" {
		t.Fatal(status, err)
	}
	seen := map[int]bool{}
	for _, e := range events {
		if seen[e.Sequence] {
			t.Fatalf("duplicate event sequence %d", e.Sequence)
		}
		seen[e.Sequence] = true
	}
}

// Review P0: after three identical invalid calls halt the run, a later
// approved shell call in the same batch must not run.
func TestNoExecutionAfterHaltInBatch(t *testing.T) {
	testutil.RequireUnixShell(t)
	a, root := testAgent(t)
	a.Grants["shell"] = true
	bad := map[string]any{"path": 7}
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return answer(toolCall("1", "read_file", bad), toolCall("2", "read_file", bad), toolCall("3", "read_file", bad),
			toolCall("4", "shell", map[string]any{"command": "touch marker"})), nil
	})
	if status, _ := a.Run(context.Background(), "go"); status == "completed" {
		t.Fatal("run should stop")
	}
	if _, err := os.Stat(filepath.Join(root, "marker")); err == nil {
		t.Fatal("a call after the halt executed")
	}
}

// Review P0: identical reads batched together must still hit the repeat
// guard, even with reordered keys.
func TestRepeatGuardCoversBatchedReads(t *testing.T) {
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "f"), []byte("x"), 0o644)
	var results []string
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		if turn++; turn == 1 {
			r1 := toolCall("1", "read_file", map[string]any{"path": "f", "start_line": 1})
			r2 := toolCall("2", "read_file", map[string]any{"path": "f", "start_line": 1})
			r2.Function.Arguments = `{"start_line":1, "path":"f"}` // same call, different key order
			r3, r4 := r1, r1
			r3.ID, r4.ID = "3", "4"
			return answer(r1, r2, r3, r4), nil
		}
		for _, m := range h[len(h)-4:] {
			results = append(results, m.Content)
		}
		return answer(), nil
	})
	status, _ := a.Run(context.Background(), "go")
	if status == "completed" && len(results) == 4 && !strings.Contains(strings.Join(results, "|"), "repeated") {
		t.Fatalf("four identical reads ran unchecked: %v", results)
	}
	var all strings.Builder
	for _, r := range a.Store.Records() {
		all.Write(r.Data)
	}
	if !strings.Contains(all.String(), "repeated") {
		t.Fatalf("repeat guard never triggered (status %s)", status)
	}
}
