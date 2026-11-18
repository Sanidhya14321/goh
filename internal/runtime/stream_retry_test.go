package runtime

import (
	"context"
	"fmt"
	"testing"
	"time"

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

// A reply the provider refused mid-generation (Groq's tool_use_failed) is
// sampled again up to twice, without configuration; nothing from it runs.
func TestRejectedOutputIsResampled(t *testing.T) {
	for name, tc := range map[string]struct {
		failures   int
		wantStatus string
		wantCalls  int
	}{
		"once":   {1, "completed", 2},
		"always": {5, "failed", 3},
	} {
		t.Run(name, func(t *testing.T) {
			a, _ := testAgent(t)
			calls := 0
			a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
				calls++
				if calls <= tc.failures {
					return core.Reply{}, provider.ErrRejectedOutput
				}
				return core.Reply{Message: core.Message{Role: "assistant", Content: "done"}, Finish: "stop"}, nil
			})
			status, _ := a.Run(context.Background(), "go")
			if status != tc.wantStatus || calls != tc.wantCalls {
				t.Fatalf("status %s calls %d", status, calls)
			}
		})
	}
}

// A provider that reports itself overloaded inside the stream is asked
// again after a pause, up to three times, and the pause is announced.
func TestBusyProviderIsRetriedAfterAPause(t *testing.T) {
	saved := busyWait
	t.Cleanup(func() { busyWait = saved })
	var waits []time.Duration
	busyWait = func(retry int) time.Duration { w := saved(retry); waits = append(waits, w); return time.Millisecond }
	for name, tc := range map[string]struct {
		failures   int
		wantStatus string
		wantCalls  int
	}{
		"twice":  {2, "completed", 3},
		"always": {9, "failed", 4},
	} {
		t.Run(name, func(t *testing.T) {
			waits = nil
			a, _ := testAgent(t)
			var notices []core.RetryNotice
			a.Emit = func(e core.Event) {
				if n, ok := e.Data.(core.RetryNotice); ok && e.Type == "provider_retry" {
					notices = append(notices, n)
				}
			}
			calls := 0
			a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
				calls++
				if calls <= tc.failures {
					return core.Reply{}, fmt.Errorf("wrapped: %w", provider.ErrBusy)
				}
				return core.Reply{Message: core.Message{Role: "assistant", Content: "done"}, Finish: "stop"}, nil
			})
			status, _ := a.Run(context.Background(), "go")
			if status != tc.wantStatus || calls != tc.wantCalls {
				t.Fatalf("status %s calls %d", status, calls)
			}
			if len(notices) != calls-1 || len(waits) != calls-1 || waits[0] != 5*time.Second || notices[0].Reason != "overloaded" || notices[0].DelayMS != 1 || notices[len(notices)-1].Max != busyRetries+1 {
				t.Fatalf("notices %+v waits %v", notices, waits)
			}
		})
	}
}
