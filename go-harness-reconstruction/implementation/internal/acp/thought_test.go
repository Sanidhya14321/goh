package acp

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
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
