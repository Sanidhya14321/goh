package runtime

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
)

// A capped run tells the model what is left at 50, 75 and 90 percent, once
// each, on the newest tool result; earlier results are never rewritten.
func TestBudgetNotesAtMarks(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.MaxToolCalls = 10
	var notes []string
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		if last := h[len(h)-1]; last.Role == "tool" {
			text, _, _ := provider.ToolResultText(last.Content)
			if i := strings.Index(text, "\n[Run budget: "); i >= 0 {
				notes = append(notes, text[i+1:])
			}
		}
		if turn++; turn > 9 {
			return answer(), nil
		}
		return answer(toolCall(ID(), "glob", map[string]any{"pattern": strings.Repeat("*", turn)})), nil
	})
	if status, err := a.Run(context.Background(), "look"); status != "completed" {
		t.Fatal(status, err)
	}
	want := []string{"5 of 10 tool calls left. Plan", "2 of 10 tool calls left. Plan", "1 of 10 tool calls left. Finish now"}
	if len(notes) != len(want) {
		t.Fatalf("notes %q", notes)
	}
	for i, w := range want {
		if !strings.Contains(notes[i], w) {
			t.Fatalf("note %d = %q, want %q", i, notes[i], w)
		}
	}
	// Only the result that crossed a mark carries a note.
	carried := 0
	for _, m := range a.History {
		if m.Role == "tool" && strings.Contains(m.Content, `"budget"`) {
			carried++
		}
	}
	if carried != 3 {
		t.Fatalf("%d results carry a note", carried)
	}
}

func TestNoBudgetNoNote(t *testing.T) {
	a, _ := testAgent(t)
	turn := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		if turn++; turn > 3 {
			return answer(), nil
		}
		return answer(toolCall(ID(), "glob", map[string]any{"pattern": strings.Repeat("*", turn)})), nil
	})
	if status, err := a.Run(context.Background(), "look"); status != "completed" {
		t.Fatal(status, err)
	}
	for _, m := range a.History {
		if strings.Contains(m.Content, `"budget"`) {
			t.Fatal("uncapped run got a budget note:", m.Content)
		}
	}
}

func TestShortDuration(t *testing.T) {
	for sec, want := range map[float64]string{0: "0s", 45: "45s", 720: "12m", 3900: "1h05m"} {
		if got := shortDuration(sec); got != want {
			t.Errorf("%v: %s, want %s", sec, got, want)
		}
	}
}

func TestBudgetNoteTime(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.MaxRunSeconds = 1000
	a.resetBudget(time.Now())
	ctx, stop := context.WithDeadline(context.Background(), time.Now().Add(400*time.Second))
	defer stop()
	if note := a.budgetNote(ctx); !strings.Contains(note, "6m of 16m left") || !strings.Contains(note, "Plan") {
		t.Fatalf("note %q", note)
	}
	if note := a.budgetNote(ctx); note != "" {
		t.Fatalf("mark reported twice: %q", note)
	}
}
