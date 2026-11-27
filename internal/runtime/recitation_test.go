package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// With task-recitation on, a long run is shown its task once before it
// finishes; short runs and runs without the experiment are not.
func TestTaskRecitation(t *testing.T) {
	run := func(experiment bool, calls int) (recited int, last string) {
		a, _ := testAgent(t)
		if experiment {
			a.Config.Experiments = []string{core.ExperimentTaskRecitation}
		}
		turn := 0
		a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
			if m := h[len(h)-1]; m.Role == "user" && strings.Contains(m.Content, "re-read the task") {
				recited++
				if !strings.Contains(m.Content, "The task:\nwrite the socket to /tmp/q.sock") {
					t.Fatalf("task not quoted: %q", m.Content)
				}
			}
			if turn++; turn > calls {
				return answer(), nil
			}
			return answer(toolCall(ID(), "glob", map[string]any{"pattern": strings.Repeat("*", turn)})), nil
		})
		status, err := a.Run(context.Background(), "write the socket to /tmp/q.sock")
		if status != "completed" {
			t.Fatal(status, err)
		}
		return recited, a.History[len(a.History)-1].Role
	}
	if n, last := run(true, recitationMinCalls); n != 1 || last != "assistant" {
		t.Fatalf("long run: recited %d times, last message %s", n, last)
	}
	if n, _ := run(true, recitationMinCalls-1); n != 0 {
		t.Fatalf("short run recited %d times", n)
	}
	if n, _ := run(false, recitationMinCalls+2); n != 0 {
		t.Fatalf("experiment off, recited %d times", n)
	}
}

func TestRecitationPromptCapsTask(t *testing.T) {
	p := recitationPrompt(strings.Repeat("é", recitationMaxBytes))
	if !strings.HasSuffix(p, "\n[…]") || len(p) > recitationMaxBytes+600 || !strings.Contains(p, "é") {
		t.Fatal(len(p))
	}
}
