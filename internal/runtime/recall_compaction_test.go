package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/memory"
	"github.com/Sanidhya14321/goh/internal/provider"
)

// echoSummaryModel answers a compaction request by echoing everything it was
// given, so any recall body in the summary input reaches the summary.
type echoSummaryModel struct{ calls int }

func (m *echoSummaryModel) Complete(_ context.Context, ms []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
	m.calls++
	var b strings.Builder
	for _, x := range ms {
		b.WriteString(x.Content)
	}
	echoed := strings.ReplaceAll(b.String(), recallPadding, "")
	return core.Reply{Message: core.Message{Role: "assistant", Content: "echo: " + echoed}, Finish: "stop"}, nil
}

const recallPadding = "older investigation detail. "

// recallSession runs one turn that recalls NOTE-777 (evidence config.go),
// then two more groups so the recall is older than the compaction cut.
func recallSession(t *testing.T, memoryOn bool) (*Agent, string) {
	t.Helper()
	a, root := testAgent(t)
	a.Config.Memory = true
	os.WriteFile(filepath.Join(root, "config.go"), []byte("package x // port 8080\n"), 0o644)
	ledger, err := memory.New(a.Config.SessionDir, a.Store.Project)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Add("NOTE-777 the server listens on port 8080", []string{"config.go"}); err != nil {
		t.Fatal(err)
	}
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		if turn == 1 {
			return answer(toolCall("m", "memory_recall", map[string]any{"query": "server port"})), nil
		}
		return core.Reply{Message: core.Message{Role: "assistant", Content: "noted " + strings.Repeat(recallPadding, 200)}, Finish: "stop"}, nil
	})
	for _, prompt := range []string{"what port?", "keep the API stable", "and now?"} {
		if status, err := a.Run(context.Background(), prompt); status != "completed" {
			t.Fatal(status, err)
		}
	}
	if !strings.Contains(joinContent(a.History), "NOTE-777") {
		t.Fatal("fixture did not recall the note")
	}
	a.Config.Memory = memoryOn
	return a, root
}

func joinContent(ms []core.Message) string {
	var b strings.Builder
	for _, m := range ms {
		b.WriteString(m.Content)
		b.WriteByte('\n')
	}
	return b.String()
}

// Handoff P1: a compaction summary never carries a project-note body, so a
// note that is stale now (or becomes stale later) cannot return as summary
// prose. The note stays reachable through ordinary validated recall.
func TestCompactionSummaryOmitsRecallBodies(t *testing.T) {
	for _, tc := range []struct {
		name     string
		memoryOn bool
		change   string // "before", "after" or "delete" (after)
		manual   bool
	}{
		{"stale before manual", true, "before", true},
		{"fresh at manual then changed", true, "after", true},
		{"fresh at manual then deleted", true, "delete", true},
		{"memory off manual", false, "", true},
		{"stale before automatic", true, "before", false},
		{"fresh at automatic then changed", true, "after", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, root := recallSession(t, tc.memoryOn)
			if tc.change == "before" {
				os.WriteFile(filepath.Join(root, "config.go"), []byte("package x // port 9090\n"), 0o644)
			}
			echo := &echoSummaryModel{}
			a.Provider = echo
			if tc.manual {
				if status, err := a.Compact(context.Background()); status != "completed" {
					t.Fatal(status, err)
				}
			} else {
				// The automatic path: over the trigger, masking off so the
				// summary path runs.
				off := false
				a.Config.CompactionMasking = &off
				a.Config.ContextWindow = 131072
				a.Config.CompactionThreshold = 5
				if err := a.compactWithMode(context.Background(), a.localDefinitions(), false); err != nil {
					t.Fatal(err)
				}
			}
			if echo.calls != 1 {
				t.Fatalf("summary calls = %d", echo.calls)
			}
			summary := a.History[1].Content
			if !strings.HasPrefix(summary, "Previous session state") {
				t.Fatalf("no summary: %.200s", summary)
			}
			if strings.Contains(summary, "NOTE-777") || strings.Contains(summary, "port 8080") {
				t.Fatalf("recall body reached the summary: %.600s", summary)
			}
			if !strings.Contains(summary, "memory_recall") {
				t.Fatalf("recall receipt missing from summary input: %.600s", summary)
			}
			// Current user constraints survive compaction.
			if !strings.Contains(joinContent(a.modelHistory()), "keep the API stable") {
				t.Fatal("user constraint lost")
			}
			switch tc.change {
			case "after":
				os.WriteFile(filepath.Join(root, "config.go"), []byte("package x // port 9090\n"), 0o644)
			case "delete":
				os.Remove(filepath.Join(root, "config.go"))
			}
			// Resume from the journal (older reader path) and confirm the
			// next ordinary request carries no stale body.
			resumed, err := New(a.Config, a.Store)
			if err != nil {
				t.Fatal(err)
			}
			if got := joinContent(resumed.modelHistory()); strings.Contains(got, "NOTE-777") {
				t.Fatalf("stale note replayed after compaction: %.600s", got)
			}
			// The original note-bearing record is kept for audit.
			audit := false
			for _, r := range a.Store.Records() {
				if r.Type == "message" && strings.Contains(string(r.Data), "NOTE-777") {
					audit = true
				}
			}
			if !audit {
				t.Fatal("original recall record lost")
			}
		})
	}
}

// A note that is still fresh after compaction is available again through
// ordinary validated recall.
func TestFreshNoteRecallableAfterCompaction(t *testing.T) {
	a, _ := recallSession(t, true)
	a.Provider = &echoSummaryModel{}
	if status, err := a.Compact(context.Background()); status != "completed" {
		t.Fatal(status, err)
	}
	turn := 0
	var seen string
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		if turn == 1 {
			return answer(toolCall("m2", "memory_recall", map[string]any{"query": "server port"})), nil
		}
		seen = h[len(h)-1].Content
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "what port again?"); status != "completed" {
		t.Fatal(status, err)
	}
	if !strings.Contains(seen, "NOTE-777") {
		t.Fatalf("fresh note not recallable: %.300s", seen)
	}
}

// An oversized recall result is never excerpted with its raw head: the
// excerpt would hide the note from revalidation and replay it stale.
func TestLargeRecallResultIsNotExcerpted(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"notes": []memory.Note{{ID: "n1", Text: "NOTE-777 " + strings.Repeat("x", 4000)}}, "authority": "data"})
	result, _ := json.Marshal(map[string]any{"content": string(body)})
	history := []core.Message{
		{Role: "system", Content: "s"},
		{Role: "assistant", ToolCalls: []core.Call{toolCall("m", "memory_recall", map[string]any{})}},
		{Role: "tool", ToolCallID: "m", Content: string(result)},
	}
	out, shrunk := shrinkLargeResults(history, 1024)
	if shrunk != 1 || strings.Contains(out[2].Content, "NOTE-777") || !strings.HasPrefix(out[2].Content, maskPrefix) {
		t.Fatalf("recall excerpted: %d %.200s", shrunk, out[2].Content)
	}
}

// recall_observation does not replay a journaled note body unvalidated.
func TestRecallObservationRefusesProjectNotes(t *testing.T) {
	a, _ := recallSession(t, true)
	var id string
	for _, m := range a.History {
		if m.Role == "tool" && m.ToolCallID == "m" {
			id = recallID(m.ToolCallID, m.Content)
		}
	}
	args, _ := json.Marshal(map[string]any{"id": id})
	r := a.executeRecall(args)
	if !r.Error || strings.Contains(r.Content, "NOTE-777") {
		t.Fatalf("note replayed from journal: %+v", r)
	}
}

// Handoff P1: a compaction error reports its actual reason. Only genuine
// capacity exhaustion is budget_exhausted; a provider 401 or cancellation is
// not, through both the manual and the automatic entry point.
func TestCompactionStatusFollowsCause(t *testing.T) {
	seed := func(t *testing.T) *Agent {
		a := contextAgent(t)
		for _, m := range []core.Message{{Role: "user", Content: "older"}, {Role: "assistant", Content: strings.Repeat("x", 25000)}, {Role: "user", Content: "active"}, {Role: "assistant", Content: "recent1"}, {Role: "assistant", Content: strings.Repeat("y", 6000)}} {
			if e := a.commit(m); e != nil {
				t.Fatal(e)
			}
		}
		return a
	}
	unauthorized := &provider.HTTPError{Status: 401, Kind: "authentication"}
	for _, tc := range []struct {
		name   string
		model  Model
		cancel bool
		want   string
	}{
		{"provider 401", &summaryModel{err: unauthorized}, false, "failed"},
		{"ineffective summary", &summaryModel{summary: strings.Repeat("z", 45000)}, false, "budget_exhausted"},
		{"provider says too large", &summaryModel{err: &provider.HTTPError{Status: 413, Kind: "request_too_large"}}, false, "budget_exhausted"},
		{"cancelled", &summaryModel{err: context.Canceled}, true, "cancelled"},
	} {
		t.Run("manual "+tc.name, func(t *testing.T) {
			a := seed(t)
			a.Provider = tc.model
			ctx, cancel := context.WithCancel(context.Background())
			if tc.cancel {
				cancel()
			}
			defer cancel()
			status, err := a.Compact(ctx)
			if status != tc.want || err == nil {
				t.Fatalf("status %q (%v), want %q", status, err, tc.want)
			}
			if tc.name == "provider 401" && !errors.As(err, new(*provider.HTTPError)) {
				t.Fatalf("provider error not preserved: %v", err)
			}
		})
		t.Run("automatic "+tc.name, func(t *testing.T) {
			a := seed(t)
			off := false
			a.Config.CompactionMasking = &off
			a.Provider = tc.model
			ctx, cancel := context.WithCancel(context.Background())
			if tc.cancel {
				cancel()
			}
			defer cancel()
			status, err := a.Run(ctx, "next")
			if status != tc.want || err == nil {
				t.Fatalf("status %q (%v), want %q", status, err, tc.want)
			}
			if m, ok := tc.model.(*summaryModel); ok && !tc.cancel && len(m.seen) == 0 {
				t.Fatal("summary request never reached the provider")
			}
		})
	}
}
