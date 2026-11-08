package provider

import (
	"strings"
	"testing"
)

func TestResponsesReasoningSummaryStreamsToSink(t *testing.T) {
	stream := `data: {"type":"response.reasoning_summary_part.added"}` + "\n\n" +
		`data: {"type":"response.reasoning_summary_text.delta","delta":"Look at "}` + "\n\n" +
		`data: {"type":"response.reasoning_summary_text.delta","delta":"main.go"}` + "\n\n" +
		`data: {"type":"response.output_text.delta","delta":"Hi"}` + "\n\n" +
		`data: {"type":"response.completed","response":{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Hi"}]}]}}` + "\n\n"
	var thought, text strings.Builder
	reply, err := parseResponsesReasoning(strings.NewReader(stream), func(s string) { text.WriteString(s) }, func(s string) { thought.WriteString(s) })
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(thought.String()) != "Look at main.go" || text.String() != "Hi" || reply.Message.Content != "Hi" {
		t.Fatalf("thought %q text %q reply %q", thought.String(), text.String(), reply.Message.Content)
	}
	// Without a sink the same stream still parses.
	if _, err := parseResponses(strings.NewReader(stream), nil); err != nil {
		t.Fatal(err)
	}
}
