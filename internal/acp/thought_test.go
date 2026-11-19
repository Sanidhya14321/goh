package acp

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
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

// Review: ACP shows the question that needs an answer and a failed
// configured check, since its stop reasons cannot carry either.
func TestNeedsInputAndFailedVerificationReachTheEditor(t *testing.T) {
	var out bytes.Buffer
	s := &server{ctx: context.Background(), output: &out}
	tr := &turn{server: s, sessionID: "s1", calls: map[string]*toolCall{}}
	tr.event(core.Event{Type: "question_requested", Data: core.QuestionRequest{Questions: []core.Question{{Question: "Which database?", Options: []core.QuestionOption{{Label: "sqlite"}, {Label: "postgres"}}}}}})
	if next := tr.inputNeeded(); !strings.Contains(next, "Which database? (options: sqlite, postgres)") || !strings.Contains(next, "Reply in your next message") {
		t.Fatal(next)
	}
	code := 2
	tr.event(core.Event{Type: "run_finished", Data: map[string]any{"status": "completed", "outcome": "failed_verification",
		"verification_detail": map[string]any{"evidence": &agent.Evidence{Command: "make test", ExitCode: &code}}}})
	if got := out.String(); !strings.Contains(got, "Verification failed: make test (exit 2)") {
		t.Fatal(got)
	}
}
