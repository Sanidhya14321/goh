package rewind

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
)

type script []core.Call

// model issues one tool call per scripted step, then a final answer.
func (s *script) Complete(_ context.Context, _ []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
	if len(*s) == 0 {
		return core.Reply{Message: core.Message{Role: "assistant", Content: "done"}, Finish: "stop"}, nil
	}
	c := (*s)[0]
	*s = (*s)[1:]
	return core.Reply{Message: core.Message{Role: "assistant", ToolCalls: []core.Call{c}}, Finish: "tool_calls"}, nil
}

func call(id, name string, args any) core.Call {
	b, _ := json.Marshal(args)
	c := core.Call{ID: id, Type: "function"}
	c.Function.Name, c.Function.Arguments = name, string(b)
	return c
}

// runTurns runs each prompt with its scripted tool calls in one session.
func runTurns(t *testing.T, root, dir string, turns map[string][]core.Call, order ...string) string {
	t.Helper()
	cfg := core.Defaults()
	cfg.SessionDir, cfg.Model, cfg.ContextWindow, cfg.Snapshots = dir, "fixture", 1<<20, true
	store, err := session.New(dir, root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	a, err := agent.New(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range []string{"create_file", "write_file", "edit_file", "shell"} {
		a.Grants[g] = true
	}
	for _, prompt := range order {
		s := script(turns[prompt])
		a.Provider = &s
		if status, err := a.Run(context.Background(), prompt); status != "completed" || err != nil {
			t.Fatal(prompt, status, err)
		}
	}
	return store.ID
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "<missing>"
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestUndoRedoRoundTrip(t *testing.T) {
	root, dir := t.TempDir(), t.TempDir()
	note := filepath.Join(root, "note.txt")
	id := runTurns(t, root, dir, map[string][]core.Call{
		"first":  {call("c1", "create_file", map[string]any{"path": "note.txt", "content": "one"})},
		"second": {call("c2", "create_file", map[string]any{"path": "other.txt", "content": "two"})},
	}, "first", "second")

	undo, err := Undo(context.Background(), dir, id)
	if err != nil {
		t.Fatal(err)
	}
	if undo.Prompt != "second" || undo.Turn != 2 || read(t, filepath.Join(root, "other.txt")) != "<missing>" || read(t, note) != "one" {
		t.Fatalf("%+v other=%q", undo, read(t, filepath.Join(root, "other.txt")))
	}
	if len(undo.Files) != 1 || undo.Files[0] != "other.txt" || len(undo.Unrestorable) != 0 {
		t.Fatalf("%+v", undo)
	}
	branch, err := session.Open(dir, undo.Session)
	if err != nil {
		t.Fatal(err)
	}
	if turns := branch.RecentUserTurns(10); len(turns) != 1 || turns[0].Prompt != "first" {
		t.Fatal("branch should end before the undone turn", turns)
	}
	branch.Close()
	if forked, err := session.Fork(dir, undo.Session); err != nil {
		t.Fatal("undo branch cannot be forked:", err)
	} else {
		forked.Close()
	}

	redo, err := Redo(context.Background(), dir, undo.Session)
	if err != nil {
		t.Fatal(err)
	}
	if redo.Session != id || read(t, filepath.Join(root, "other.txt")) != "two" {
		t.Fatalf("%+v", redo)
	}
	if !strings.Contains(undo.Summary(true), "/redo") || !strings.Contains(redo.Summary(false), "Redid turn 2") {
		t.Fatal(undo.Summary(true), redo.Summary(false))
	}
}

func TestUndoRefusesExternalEditWithoutChangingAnything(t *testing.T) {
	root, dir := t.TempDir(), t.TempDir()
	id := runTurns(t, root, dir, map[string][]core.Call{
		"edit": {
			call("a", "create_file", map[string]any{"path": "a.txt", "content": "A"}),
			call("b", "create_file", map[string]any{"path": "b.txt", "content": "B"}),
		},
	}, "edit")
	// a.txt is checkpointed first and so undone last; changing it outside
	// GoHarness makes undo refuse after b.txt was already reverted.
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("user edit"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Undo(context.Background(), dir, id); err == nil || !strings.Contains(err.Error(), "no files changed") {
		t.Fatal(err)
	}
	if read(t, filepath.Join(root, "b.txt")) != "B" || read(t, filepath.Join(root, "a.txt")) != "user edit" {
		t.Fatal("refused undo left files changed")
	}
	infos, err := session.List(dir)
	if err != nil || len(infos) != 1 {
		t.Fatal("refused undo created a branch", len(infos), err)
	}
}

func TestRedoUnavailableAfterNewPromptAndShellWarning(t *testing.T) {
	root, dir := t.TempDir(), t.TempDir()
	id := runTurns(t, root, dir, map[string][]core.Call{
		"run": {call("s", "shell", map[string]any{"command": "printf x > made-by-shell"})},
	}, "run")
	undo, err := Undo(context.Background(), dir, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(undo.Unrestorable) != 1 || undo.Unrestorable[0] != "shell" || read(t, filepath.Join(root, "made-by-shell")) != "x" {
		t.Fatalf("%+v", undo)
	}
	if !strings.Contains(undo.Summary(true), "Not restorable") {
		t.Fatal(undo.Summary(true))
	}
	branch, err := session.Open(dir, undo.Session)
	if err != nil {
		t.Fatal(err)
	}
	if err := branch.Append("message", core.Message{Role: "user", Content: "new direction"}); err != nil {
		t.Fatal(err)
	}
	branch.Close()
	if _, err := Redo(context.Background(), dir, undo.Session); err == nil || !strings.Contains(err.Error(), "after a new prompt") {
		t.Fatal(err)
	}
	if _, err := Redo(context.Background(), dir, id); err == nil || !strings.Contains(err.Error(), "not created by /undo") {
		t.Fatal(err)
	}
}

func runTurnIn(t *testing.T, dir, id, prompt string) {
	t.Helper()
	store, err := session.Open(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cfg := core.Defaults()
	cfg.SessionDir, cfg.Model, cfg.ContextWindow = dir, "fixture", 1<<20
	a, err := agent.New(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	s := script(nil)
	a.Provider = &s
	if status, err := a.Run(context.Background(), prompt); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
}

func TestUndoBranchContinuesAsNormalChat(t *testing.T) {
	root, dir := t.TempDir(), t.TempDir()
	id := runTurns(t, root, dir, map[string][]core.Call{
		"make": {call("c", "create_file", map[string]any{"path": "n.txt", "content": "n"})},
	}, "make")
	undo, err := Undo(context.Background(), dir, id)
	if err != nil {
		t.Fatal(err)
	}
	runTurnIn(t, dir, undo.Session, "continue in branch")
	if _, err := Redo(context.Background(), dir, undo.Session); err == nil || !strings.Contains(err.Error(), id) {
		t.Fatal("redo after a real branch run must refuse and name the original session:", err)
	}
	if read(t, filepath.Join(root, "n.txt")) != "<missing>" {
		t.Fatal("refused redo changed files")
	}
}
