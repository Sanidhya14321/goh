package runtime

import (
	"context"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
)

// A broken stream is re-requested only when configured, and the broken
// reply's tool calls never run.
func TestRetryInterruptedStreams(t *testing.T) {
	for name, tc := range map[string]struct {
		retries    int
		wantStatus string
		wantCalls  int
	}{
		"off": {0, "failed", 1},
		"on":  {1, "completed", 2},
	} {
		t.Run(name, func(t *testing.T) {
			a, _ := testAgent(t)
			a.Config.RetryInterruptedStreams = tc.retries
			calls := 0
			a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
				calls++
				if calls == 1 {
					return answer(toolCall("x", "shell", map[string]any{"command": "touch never"})), provider.ErrInterrupted
				}
				return core.Reply{Message: core.Message{Role: "assistant", Content: "done"}, Finish: "stop"}, nil
			})
			status, _ := a.Run(context.Background(), "go")
			if status != tc.wantStatus || calls != tc.wantCalls {
				t.Fatalf("status %s calls %d", status, calls)
			}
			for _, r := range a.Store.Records() {
				if r.Type == "tool_requested" || r.Type == "tool_finished" {
					t.Fatal("tool from the broken reply ran")
				}
			}
		})
	}
	if err := (core.Config{RetryInterruptedStreams: 3}).Validate(); err == nil {
		t.Fatal("3 retries accepted")
	}
}

// What interfaces show must end up equal to the journaled reply: a suffix
// the stream left out is sent, and a final reply that does not continue the
// streamed text replaces it.
func TestShownTextMatchesTheFinalReply(t *testing.T) {
	for name, tc := range map[string]struct {
		streamed, final string
		reset           bool
	}{
		"suffix":  {"FIRST", "FIRST AND LAST", false},
		"differs": {"DRAFT", "FINAL", true},
		"same":    {"SAME", "SAME", false},
		"silent":  {"", "QUIET", false},
	} {
		t.Run(name, func(t *testing.T) {
			a, _ := testAgent(t)
			a.Config.RetryInterruptedStreams = 1
			calls := 0
			a.Provider = modelFunc(func(_ context.Context, _ []core.Message, _ []core.Tool, delta func(string)) (core.Reply, error) {
				calls++
				if calls == 1 {
					delta("ABANDONED")
					return core.Reply{}, provider.ErrInterrupted
				}
				if tc.streamed != "" {
					delta(tc.streamed)
				}
				return core.Reply{Message: core.Message{Role: "assistant", Content: tc.final}, Finish: "stop"}, nil
			})
			shown, reset := "", false
			a.Emit = func(e core.Event) {
				switch e.Type {
				case "stream_retry", "assistant_reset":
					reset = reset || e.Type == "assistant_reset"
					shown = ""
				case "assistant_updated":
					shown += e.Data.(map[string]any)["text"].(string)
				}
			}
			if status, err := a.Run(context.Background(), "go"); status != "completed" {
				t.Fatal(status, err)
			}
			if shown != tc.final || reset != tc.reset {
				t.Fatalf("shown %q (reset %v), journal %q", shown, reset, tc.final)
			}
		})
	}
}
