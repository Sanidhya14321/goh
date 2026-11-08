package provider

import (
	"encoding/json"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestAnthropicThinkingRoundTrip(t *testing.T) {
	wire := anthroStart() +
		anthroBlock(0, map[string]any{"type": "thinking", "thinking": "", "signature": ""}) +
		anthroDelta(0, map[string]any{"type": "thinking_delta", "thinking": "Need to read "}) +
		anthroDelta(0, map[string]any{"type": "thinking_delta", "thinking": "the file."}) +
		anthroDelta(0, map[string]any{"type": "signature_delta", "signature": "sig-abc"}) +
		anthroClose(0) +
		anthroBlock(1, map[string]any{"type": "redacted_thinking", "data": "opaque"}) + anthroClose(1) +
		anthroBlock(2, map[string]any{"type": "tool_use", "id": "toolu_1", "name": "read_file", "input": map[string]any{"path": "a"}}) + anthroClose(2) +
		anthroFinish("tool_use") + anthroStop()
	reply, err := ParseAnthropic(strings.NewReader(wire), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(reply.Message.AnthropicThinking) != 2 || len(reply.Message.ToolCalls) != 1 {
		t.Fatalf("%+v", reply.Message)
	}
	if !strings.Contains(string(reply.Message.AnthropicThinking[0]), `"thinking":"Need to read the file."`) || !strings.Contains(string(reply.Message.AnthropicThinking[0]), `"signature":"sig-abc"`) {
		t.Fatal(string(reply.Message.AnthropicThinking[0]))
	}
	history := []core.Message{{Role: "user", Content: "go"}, reply.Message, {Role: "tool", ToolCallID: "toolu_1", Content: "{}"}}
	cfg := anthroConfig()
	cfg.OutputBudget, cfg.ThinkingBudget = 4096, 2048
	b, err := EncodeAnthropic(cfg, history, nil)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Thinking map[string]any `json:"thinking"`
		Messages []struct {
			Content []map[string]any `json:"content"`
		} `json:"messages"`
	}
	json.Unmarshal(b, &body)
	if body.Thinking["budget_tokens"].(float64) != 2048 || body.Thinking["type"] != "enabled" {
		t.Fatal(string(b))
	}
	assistant := body.Messages[1].Content
	if assistant[0]["type"] != "thinking" || assistant[0]["signature"] != "sig-abc" || assistant[1]["type"] != "redacted_thinking" || assistant[2]["type"] != "tool_use" {
		t.Fatal("thinking blocks must lead the assistant turn verbatim", assistant)
	}
	cfg.ThinkingBudget = 0
	b, _ = EncodeAnthropic(cfg, history, nil)
	if strings.Contains(string(b), "thinking") {
		t.Fatal("thinking replayed while disabled")
	}
	chat := chatWireMessages(history)
	if strings.Contains(mustJSON(t, chat), "anthropic_thinking") {
		t.Fatal("chat wire messages must not carry thinking (stripped before encoding)")
	}
}

func TestThinkingBudgetValidation(t *testing.T) {
	c := core.Defaults()
	c.Protocol, c.Endpoint, c.Model, c.ContextWindow, c.OutputBudget = "anthropic_messages", "https://api.anthropic.com/v1/messages", "m", 200000, 8000
	for _, bad := range []int{500, 8000, 9000} {
		c.ThinkingBudget = bad
		if c.Validate() == nil {
			t.Fatal("accepted", bad)
		}
	}
	c.ThinkingBudget = 4000
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Protocol, c.Endpoint = "chat_completions", "https://api.openai.com/v1/chat/completions"
	if c.Validate() == nil {
		t.Fatal("thinking_budget accepted for chat completions")
	}
}
