package rewind

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/session"
)

func TestRevertToEarlierTurnUndoesItAndLaterTurns(t *testing.T) {
	root, dir := t.TempDir(), t.TempDir()
	id := runTurns(t, root, dir, map[string][]core.Call{
		"setup":  {call("c0", "create_file", map[string]any{"path": "base.txt", "content": "base"})},
		"change": {call("c1", "create_file", map[string]any{"path": "a.txt", "content": "a"})},
		"more":   {call("c2", "create_file", map[string]any{"path": "b.txt", "content": "b"})},
	}, "setup", "change", "more")

	if _, err := RevertTo(context.Background(), dir, id, "never sent", 0); err == nil || !strings.Contains(err.Error(), "nothing reverted") {
		t.Fatal("unknown prompt should be refused", err)
	}
	if read(t, filepath.Join(root, "a.txt")) != "a" {
		t.Fatal("refusal changed files")
	}

	r, err := RevertTo(context.Background(), dir, id, "change", 0)
	if err != nil {
		t.Fatal(err)
	}
	if r.Prompt != "change" || r.Turn != 2 || r.LastTurn != 3 {
		t.Fatalf("%+v", r)
	}
	if read(t, filepath.Join(root, "a.txt")) != "<missing>" || read(t, filepath.Join(root, "b.txt")) != "<missing>" || read(t, filepath.Join(root, "base.txt")) != "base" {
		t.Fatal("revert restored the wrong files")
	}
	if !strings.Contains(r.Summary(true), "Undid turns 2–3") {
		t.Fatal(r.Summary(true))
	}
	branch, err := session.Open(dir, r.Session)
	if err != nil {
		t.Fatal(err)
	}
	if turns := branch.RecentUserTurns(10); len(turns) != 1 || turns[0].Prompt != "setup" {
		t.Fatal("branch should end before the reverted turn", turns)
	}
	branch.Close()

	redo, err := Redo(context.Background(), dir, r.Session)
	if err != nil {
		t.Fatal(err)
	}
	if redo.Session != id || read(t, filepath.Join(root, "a.txt")) != "a" || read(t, filepath.Join(root, "b.txt")) != "b" {
		t.Fatalf("redo did not restore every reverted turn: %+v", redo)
	}
	if !strings.Contains(redo.Summary(false), "Redid turns 2–3") {
		t.Fatal(redo.Summary(false))
	}
}

func TestRevertPicksOccurrenceOfRepeatedPrompt(t *testing.T) {
	root, dir := t.TempDir(), t.TempDir()
	id := runTurns(t, root, dir, map[string][]core.Call{"again": nil, "other": nil}, "again", "other", "again")
	r, err := RevertTo(context.Background(), dir, id, "again", 1)
	if err != nil || r.Turn != 1 || r.LastTurn != 3 {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := RevertTo(context.Background(), dir, id, "again", 2); err == nil {
		t.Fatal("third occurrence does not exist")
	}
}

func TestRevertTurnByNumber(t *testing.T) {
	root, dir := t.TempDir(), t.TempDir()
	id := runTurns(t, root, dir, map[string][]core.Call{
		"one": {call("c1", "create_file", map[string]any{"path": "1.txt", "content": "1"})},
		"two": {call("c2", "create_file", map[string]any{"path": "2.txt", "content": "2"})},
	}, "one", "two")
	for _, bad := range []int{0, 3} {
		if _, err := RevertTurn(context.Background(), dir, id, bad); err == nil {
			t.Fatal("accepted turn", bad)
		}
	}
	r, err := RevertTurn(context.Background(), dir, id, 1)
	if err != nil || r.Prompt != "one" || r.LastTurn != 2 || read(t, filepath.Join(root, "1.txt")) != "<missing>" {
		t.Fatalf("%+v %v", r, err)
	}
}
