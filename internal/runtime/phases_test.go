package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
)

type phasesSummary struct {
	TotalMS      int64                `json:"total_ms"`
	SetupMS      int64                `json:"setup_ms"`
	Requests     []requestPhase       `json:"requests"`
	FirstTextMS  int64                `json:"first_text_ms"`
	Tools        map[string]toolPhase `json:"tools"`
	ToolWallMS   int64                `json:"tool_wall_ms"`
	PromptBytes  int                  `json:"prompt_bytes"`
	Finish       map[string]int64     `json:"finish"`
	FinalToEndMS int64                `json:"final_reply_to_end_ms"`
}

// runWithPhases runs a with GOH_PHASES on and returns the summary, checking
// that run_phases comes right before run_finished, the last event.
func runWithPhases(t *testing.T, a *Agent, ctx context.Context) phasesSummary {
	t.Helper()
	var events []core.Event
	a.Emit = func(e core.Event) { events = append(events, e) }
	a.Run(ctx, "read the note")
	n := len(events)
	if n < 2 || events[n-1].Type != "run_finished" || events[n-2].Type != "run_phases" {
		t.Fatalf("run_phases must come just before run_finished, the last event: %v", eventTypes(events))
	}
	b, _ := json.Marshal(events[n-2].Data)
	var s phasesSummary
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func eventTypes(events []core.Event) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.Type
	}
	return out
}

// An injected provider delay is attributed to the attempt that took it: a
// tool-only reply has no first-text time, a text reply has one after its
// delay, and the tool between them is counted once.
func TestPhasesAttributeProviderDelayAndTools(t *testing.T) {
	t.Setenv("GOH_PHASES", "1")
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "note.txt"), []byte("alpha\n"), 0o600)
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, _ []core.Message, _ []core.Tool, emit func(string)) (core.Reply, error) {
		turn++
		if turn == 1 {
			time.Sleep(150 * time.Millisecond)
			return core.Reply{Message: core.Message{Role: "assistant", ToolCalls: []core.Call{toolCall("r1", "read_file", map[string]any{"path": "note.txt"})}}}, nil
		}
		time.Sleep(80 * time.Millisecond)
		emit("It says alpha.")
		time.Sleep(40 * time.Millisecond)
		return core.Reply{Message: core.Message{Role: "assistant", Content: "It says alpha."}}, nil
	})
	s := runWithPhases(t, a, context.Background())
	if len(s.Requests) != 2 {
		t.Fatalf("requests: %+v", s.Requests)
	}
	first, second := s.Requests[0], s.Requests[1]
	if first.FirstTextMS != 0 || first.EndMS-first.StartMS < 150 || first.Failed {
		t.Fatalf("tool-only attempt: %+v", first)
	}
	if second.FirstTextMS-second.StartMS < 80 || second.EndMS-second.StartMS < 120 || second.EndMS < second.FirstTextMS {
		t.Fatalf("text attempt: %+v", second)
	}
	if s.FirstTextMS != second.FirstTextMS || second.StartMS < first.EndMS {
		t.Fatalf("first text or order: %+v", s)
	}
	if tool := s.Tools["read_file"]; tool.Calls != 1 || s.ToolWallMS > tool.TotalMS {
		t.Fatalf("tools: %+v wall %d", s.Tools, s.ToolWallMS)
	}
	if s.SetupMS > first.StartMS || s.TotalMS < second.EndMS || s.PromptBytes == 0 {
		t.Fatalf("setup, total or prompt size: %+v", s)
	}
	for _, step := range []string{"integrations", "journal", "snapshot", "hooks"} {
		if _, ok := s.Finish[step]; !ok {
			t.Fatalf("finish step %s missing: %v", step, s.Finish)
		}
	}
}

// A busy provider's back-off belongs to the attempt after it, not to the
// failed attempt or the provider's own time.
func TestPhasesCountRetryWaitOnTheNextAttempt(t *testing.T) {
	t.Setenv("GOH_PHASES", "1")
	saved := busyWait
	busyWait = func(int) time.Duration { return 60 * time.Millisecond }
	t.Cleanup(func() { busyWait = saved })
	a, _ := testAgent(t)
	calls := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		calls++
		if calls == 1 {
			return core.Reply{}, provider.ErrBusy
		}
		return core.Reply{Message: core.Message{Role: "assistant", Content: "done"}}, nil
	})
	s := runWithPhases(t, a, context.Background())
	if len(s.Requests) != 2 || !s.Requests[0].Failed || s.Requests[1].Failed {
		t.Fatalf("attempts: %+v", s.Requests)
	}
	if s.Requests[0].EndMS-s.Requests[0].StartMS >= 60 || s.Requests[0].RetryWaitMS != 0 || s.Requests[1].RetryWaitMS < 60 {
		t.Fatalf("retry wait misattributed: %+v", s.Requests)
	}
}

// A cancelled run still reports, and its open attempt ends failed.
func TestPhasesReportCancelledRuns(t *testing.T) {
	t.Setenv("GOH_PHASES", "1")
	a, _ := testAgent(t)
	ctx, cancel := context.WithCancel(context.Background())
	a.Provider = modelFunc(func(c context.Context, _ []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		cancel()
		<-c.Done()
		return core.Reply{}, c.Err()
	})
	s := runWithPhases(t, a, ctx)
	if len(s.Requests) != 1 || !s.Requests[0].Failed {
		t.Fatalf("cancelled attempt: %+v", s.Requests)
	}
}

// Off by default: no collector, no event, default output unchanged.
func TestPhasesOffByDefault(t *testing.T) {
	t.Setenv("GOH_PHASES", "")
	a, _ := testAgent(t)
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return core.Reply{Message: core.Message{Role: "assistant", Content: "done"}}, nil
	})
	var events []core.Event
	a.Emit = func(e core.Event) { events = append(events, e) }
	a.Run(context.Background(), "hello")
	for _, e := range events {
		if e.Type == "run_phases" {
			t.Fatal("run_phases emitted without GOH_PHASES")
		}
	}
}

func TestUnionOfOverlappingIntervals(t *testing.T) {
	if got := unionMS([][2]int64{{0, 10}, {5, 20}, {30, 40}, {35, 36}}); got != 30 {
		t.Fatal(got)
	}
	if unionMS(nil) != 0 {
		t.Fatal("empty")
	}
}
