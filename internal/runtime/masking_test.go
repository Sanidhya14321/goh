package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
)

func readGroups(a *Agent, n, size int) {
	for i := 0; i < n; i++ {
		call := core.Call{ID: ID(), Type: "function"}
		call.Function.Name = "read_file"
		call.Function.Arguments = `{"path":"x"}`
		a.History = append(a.History, core.Message{Role: "assistant", ToolCalls: []core.Call{call}}, core.Message{Role: "tool", ToolCallID: call.ID, Content: strings.Repeat("a", size)})
	}
}

func TestMaskingCompactsWithoutSummaryModel(t *testing.T) {
	a := contextAgent(t)
	a.History = append(a.History, core.Message{Role: "user", Content: "active task: fix the parser"})
	readGroups(a, 6, 6000)
	old := append([]core.Message(nil), a.History...)
	m := &summaryModel{summary: "unused"}
	a.Provider = m
	if e := a.compact(context.Background(), nil); e != nil {
		t.Fatal(e)
	}
	if m.seen != nil {
		t.Fatal("summary model called although masking was enough")
	}
	if len(a.History) != len(old) || a.History[1].Content != "active task: fix the parser" {
		t.Fatal("masking must keep every message and the active request")
	}
	masked := 0
	for i, msg := range a.History {
		if strings.HasPrefix(msg.Content, maskPrefix) {
			masked++
			if !strings.Contains(msg.Content, `"tool":"read_file"`) || msg.ToolCallID != old[i].ToolCallID {
				t.Fatal("placeholder lost tool identity", msg)
			}
		}
	}
	// The two most recent groups stay verbatim.
	for i := len(old) - 4; i < len(old); i++ {
		if a.History[i].Content != old[i].Content {
			t.Fatal("recent group was masked")
		}
	}
	if masked != 4 {
		t.Fatalf("masked %d results, want 4", masked)
	}
	rec := a.Store.Records()[len(a.Store.Records())-1]
	if rec.Type != "context_compacted" || !strings.Contains(string(rec.Data), `"method":"observation_masking"`) {
		t.Fatal("masking not journaled", string(rec.Data))
	}

	// The masked view must survive resume through the normal journal path.
	id, dir := a.Store.ID, a.Config.SessionDir
	a.Store.Close()
	reopened, err := session.Open(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	resumed, err := New(a.Config, reopened)
	if err != nil {
		t.Fatal(err)
	}
	if len(resumed.History) != len(a.History) || !strings.HasPrefix(resumed.History[3].Content, maskPrefix) {
		t.Fatal("resume did not restore the masked view")
	}
}

func TestMaskingFallsBackToSummaryOfMaskedView(t *testing.T) {
	a := contextAgent(t)
	// Large assistant text cannot be masked, so a summary is still required.
	a.History = append(a.History, core.Message{Role: "user", Content: "older"}, core.Message{Role: "assistant", Content: strings.Repeat("x", 19000)})
	readGroups(a, 2, 6000)
	a.History = append(a.History, core.Message{Role: "user", Content: "active"})
	readGroups(a, 2, 600)
	m := &summaryModel{summary: "Earlier work summarized."}
	a.Provider = m
	if e := a.compact(context.Background(), nil); e != nil {
		t.Fatal(e)
	}
	if m.seen == nil {
		t.Fatal("expected a summary call")
	}
	if strings.Contains(m.seen[1].Content, strings.Repeat("a", 6000)) || !strings.Contains(m.seen[1].Content, "masked") {
		t.Fatal("summarizer should receive the masked view")
	}
}

func TestManualCompactionDoesNotMask(t *testing.T) {
	a := contextAgent(t)
	a.History = append(a.History, core.Message{Role: "user", Content: "task"})
	readGroups(a, 6, 6000)
	m := &summaryModel{summary: "Manual summary."}
	a.Provider = m
	if e := a.compactWithMode(context.Background(), nil, true); e != nil {
		t.Fatal(e)
	}
	if m.seen == nil || !strings.Contains(a.History[1].Content, "Manual summary.") {
		t.Fatal("manual /compact must summarize")
	}
}

// TestMaskingMeasurement logs estimator sizes for the fixture. It measures
// context bytes and model calls only, not answer quality or solve rate.
func TestMaskingMeasurement(t *testing.T) {
	for _, mask := range []bool{true, false} {
		a := contextAgent(t)
		a.Config.CompactionMasking = &mask
		a.History = append(a.History, core.Message{Role: "user", Content: "active task"})
		readGroups(a, 6, 6000)
		m := &summaryModel{summary: "Summary of earlier reads."}
		a.Provider = m
		before := estimate(a.modelHistory(), nil)
		if e := a.compact(context.Background(), nil); e != nil {
			t.Fatal(e)
		}
		sent := 0
		if m.seen != nil {
			sent = estimate(m.seen, nil)
		}
		t.Logf("masking=%v before=%d after=%d summarizer_calls=%d summarizer_input=%d", mask, before, estimate(a.modelHistory(), nil), map[bool]int{true: 1, false: 0}[m.seen != nil], sent)
	}
}
