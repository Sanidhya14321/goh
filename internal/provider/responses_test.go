package provider

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestResponsesCompletesBeforeMakingToolCallsAvailable(t *testing.T) {
	const partial = "data: {\"type\":\"response.output_item.added\",\"item\":{\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"read_file\",\"arguments\":\"{}\"}}\n\n"
	reply, err := parseResponses(strings.NewReader(partial), nil)
	if !errors.Is(err, ErrInterrupted) || len(reply.Message.ToolCalls) != 0 {
		t.Fatal("partial Responses stream exposed tool calls")
	}
	complete := "data: " + `{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":6,"output_tokens":4,"total_tokens":10},"output":[{"type":"reasoning","id":"rs_fixture","summary":[],"encrypted_content":"opaque-fixture"},{"type":"function_call","namespace":"goharness","call_id":"call_1","name":"read_file","arguments":"{}","status":"completed"}]}}` + "\n\n"
	reply, err = parseResponses(strings.NewReader(partial+complete), nil)
	if err != nil || len(reply.Message.ToolCalls) != 1 || reply.Usage.Total != 10 || reply.Finish != "tool_calls" {
		t.Fatalf("did not assemble the completed tool reply: %+v %v", reply, err)
	}
	config := core.Config{Protocol: core.ChatGPTProtocol, Model: "fixture", OutputBudget: 2048}
	reply.Message.ResponsesRoute = responsesRoute(config)
	encoded, err := encodeResponses(config,
		[]core.Message{{Role: "system", Content: "local instructions"}, reply.Message, {Role: "tool", ToolCallID: "call_1", Content: "local result"}},
		[]core.Tool{core.NewTool("read_file", "Read a local file", map[string]any{"type": "object"})})
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatal(err)
	}
	input := body["input"].([]any)
	if body["store"] != false || body["stream"] != true || body["max_output_tokens"] != nil ||
		input[0].(map[string]any)["role"] != "developer" || input[1].(map[string]any)["encrypted_content"] != "opaque-fixture" ||
		input[2].(map[string]any)["namespace"] != "goharness" || input[3].(map[string]any)["call_id"] != "call_1" ||
		body["tools"].([]any)[0].(map[string]any)["type"] != "namespace" {
		t.Fatal("ChatGPT request omitted required history, routing or preview options")
	}
	config.Model = "another-model"
	other, err := encodeResponses(config, []core.Message{reply.Message}, nil)
	if err != nil || strings.Contains(string(other), "opaque-fixture") {
		t.Fatal("reasoning from a different model route was replayed")
	}
}
