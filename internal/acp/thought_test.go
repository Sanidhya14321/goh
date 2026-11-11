package acp

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestReasoningBecomesThoughtChunk(t *testing.T) {
	var out bytes.Buffer
	s := &server{ctx: context.Background(), output: &out}
	tr := &turn{server: s, sessionID: "s1", calls: map[string]*toolCall{}}
	tr.event(core.Event{Type: "reasoning_updated", Data: map[string]any{"text": "plan first"}})
	got := out.String()
	if !strings.Contains(got, `"sessionUpdate":"agent_thought_chunk"`) || !strings.Contains(got, "plan first") {
		t.Fatal(got)
	}
}

func TestRetryAndFinalSuffixReachTheEditor(t *testing.T) {
	var out bytes.Buffer
	s := &server{ctx: context.Background(), output: &out}
	tr := &turn{server: s, sessionID: "s1", calls: map[string]*toolCall{}}
	tr.event(core.Event{Type: "assistant_updated", Data: map[string]any{"text": "BROKEN PART"}})
	tr.event(core.Event{Type: "stream_retry", Data: map[string]any{"attempt": 1}})
	tr.event(core.Event{Type: "assistant_updated", Data: map[string]any{"text": "FIRST "}})
	tr.event(core.Event{Type: "assistant_completed", Data: map[string]any{"message": core.Message{Role: "assistant", Content: "FIRST AND LAST"}}})
	got := out.String()
	if !strings.Contains(got, "partial reply above was discarded") {
		t.Fatal("retry not shown:", got)
	}
	if !strings.Contains(got, `"text":"AND LAST"`) {
		t.Fatal("final suffix missing:", got)
	}
}
