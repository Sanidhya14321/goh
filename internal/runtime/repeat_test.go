package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// loopingModel requests the same read forever.
func loopingModel(reads *int) modelFunc {
	return func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		*reads++
		if *reads > 10 {
			return answer(), nil
		}
		return answer(toolCall("loop", "read_file", map[string]any{"path": "a.txt"})), nil
	}
}

func countResults(a *Agent) (ok, blocked int) {
	for _, m := range a.History {
		if m.Role != "tool" {
			continue
		}
		if strings.Contains(m.Content, "repeated") || strings.Contains(m.Content, "declined repeating") {
			blocked++
		} else {
			ok++
		}
	}
	return
}

func TestRepeatedCallStopsHeadlessRun(t *testing.T) {
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("x"), 0o600)
	a.Approve = nil
	reads := 0
	a.Provider = loopingModel(&reads)
	status, err := a.Run(context.Background(), "go")
	if status != "needs_permission" || err == nil || !strings.Contains(err.Error(), "repeated") {
		t.Fatal(status, err)
	}
	if ok, blocked := countResults(a); ok != 2 || blocked != 1 {
		t.Fatal("expected two executions then a stop", ok, blocked)
	}
}

func TestRepeatedCallAskEachTimeOrForSession(t *testing.T) {
	for _, session := range []bool{false, true} {
		a, root := testAgent(t)
		os.WriteFile(filepath.Join(root, "a.txt"), []byte("x"), 0o600)
		asked := 0
		a.Approve = func(_ context.Context, p core.Permission) (core.Decision, error) {
			if p.Tool != "repeated_call" || !strings.Contains(p.Preview, "may be stuck") {
				t.Fatal("unexpected approval", p.Tool)
			}
			asked++
			return core.Decision{Allow: session, Session: session}, nil
		}
		reads := 0
		a.Provider = loopingModel(&reads)
		status, err := a.Run(context.Background(), "go")
		if status != "completed" || err != nil {
			t.Fatal(status, err)
		}
		ok, blocked := countResults(a)
		if session {
			if asked != 1 || ok != 10 || blocked != 0 {
				t.Fatal("session approval should ask once and allow repeats", asked, ok, blocked)
			}
		} else if asked != 8 || ok != 2 || blocked != 8 {
			t.Fatal("declines should block every further repeat", asked, ok, blocked)
		}
	}
}

func TestDifferentArgumentsResetRepeatCount(t *testing.T) {
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("x"), 0o600)
	a.Approve = nil
	n := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		n++
		if n > 6 {
			return answer(), nil
		}
		args := map[string]any{"path": "a.txt"}
		if n%3 == 0 {
			args["start_line"] = 1
		}
		return answer(toolCall("c", "read_file", args)), nil
	})
	if status, err := a.Run(context.Background(), "go"); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
}
