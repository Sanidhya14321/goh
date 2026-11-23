package tui

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
)

// The status line says what the run is doing at each stage instead of only
// "Working".
func TestActivityLabels(t *testing.T) {
	var a activity
	step := func(kind string, data any, want string) {
		t.Helper()
		a.track(core.Event{Type: kind, Data: data})
		if got := a.label("gpt-x"); !strings.Contains(got, want) {
			t.Fatalf("after %s: %q, want %q", kind, got, want)
		}
	}
	step("run_started", nil, "Working")
	step("assistant_started", map[string]any{"turn": 0}, "Waiting for gpt-x")
	step("reasoning_updated", map[string]any{"text": "plan"}, "Thinking")
	step("tool_progress", core.ToolProgress{Tool: "create_file", Chars: 40}, "Writing create_file · 10 tokens")
	step("tool_progress", core.ToolProgress{Tool: "create_file", Target: "web/index.html", Chars: 8000}, "Writing web/index.html · 2.0k tokens")
	a.since = time.Now().Add(-4 * time.Second)
	step("tool_progress", core.ToolProgress{Tool: "create_file", Target: "web/index.html", Chars: 8000}, "/s)")
	step("tool_progress", core.ToolProgress{Tool: "shell", Target: "npm test"}, "Preparing command npm test")
	step("tool_started", map[string]any{"name": "shell", "arguments": json.RawMessage(`{"command":"npm test"}`)}, "Running npm test")
	step("tool_finished", nil, "Working")
	step("assistant_started", nil, "Waiting for gpt-x")
	step("assistant_updated", map[string]any{"text": "Done"}, "Writing the reply")
	if got := a.counts(); got != "step 2 · 1 tool" {
		t.Fatalf("counts %q", got)
	}
}

// The first wait of a run is timed once, by the run timer; a later phase
// shows its own time beside the run's.
func TestPhaseTimerNotRepeated(t *testing.T) {
	var a activity
	a.track(core.Event{Type: "run_started"})
	a.track(core.Event{Type: "assistant_started"})
	a.since = a.since.Add(-3 * time.Second)
	a.start = a.start.Add(-3 * time.Second)
	if got := a.label("m"); got != "Waiting for m" {
		t.Fatalf("first wait: %q", got)
	}
	a.start = a.start.Add(-10 * time.Second) // the run began well before this phase
	if got := a.label("m"); !strings.HasPrefix(got, "Waiting for m · 3.") {
		t.Fatalf("later wait: %q", got)
	}
}
