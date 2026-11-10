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
