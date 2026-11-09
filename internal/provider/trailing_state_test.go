package provider

import (
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// The runtime appends session task state as a final user message after tool
// results, keeping the system prefix cache-stable. Every encoder must accept it.
func TestEncodersAcceptTrailingStateAfterToolResults(t *testing.T) {
	cfg := core.Defaults()
	cfg.Model = "fixture"
	call := core.Call{ID: "call_1", Type: "function"}
	call.Function.Name, call.Function.Arguments = "read_file", `{"path":"a"}`
	history := []core.Message{
		{Role: "system", Content: "framing"},
		{Role: "user", Content: "do it"},
		{Role: "assistant", ToolCalls: []core.Call{call}},
		{Role: "tool", ToolCallID: "call_1", Content: `{"content":"x"}`},
		{Role: "user", Content: "[Session state, not a new request] Current session task list"},
	}
	a, err := EncodeAnthropic(cfg, history, nil)
	if err != nil || !strings.Contains(string(a), "Session state") || !strings.Contains(string(a), "tool_result") {
		t.Fatal("anthropic", err, string(a))
	}
	r, err := encodeResponses(cfg, history, nil)
	if err != nil || !strings.Contains(string(r), "Session state") || !strings.Contains(string(r), "function_call_output") {
		t.Fatal("responses", err, string(r))
	}
}
