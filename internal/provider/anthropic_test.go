package provider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func anthroCall(id, name, args string) core.Call {
	c := core.Call{ID: id, Type: "function"}
	c.Function.Name = name
	c.Function.Arguments = args
	return c
}
func anthroFrame(kind string, v any) string {
	b, _ := json.Marshal(v)
	return "event: " + kind + "\ndata: " + string(b) + "\n\n"
}
func anthroStart() string {
	return anthroFrame("message_start", map[string]any{"type": "message_start", "message": map[string]any{"id": "msg_fixture", "type": "message", "role": "assistant", "content": []any{}, "usage": map[string]any{"input_tokens": 7, "output_tokens": 1, "cache_read_input_tokens": 2, "cache_creation_input_tokens": 3}}})
}
func anthroBlock(i int, block any) string {
	return anthroFrame("content_block_start", map[string]any{"type": "content_block_start", "index": i, "content_block": block})
}
func anthroDelta(i int, delta any) string {
	return anthroFrame("content_block_delta", map[string]any{"type": "content_block_delta", "index": i, "delta": delta})
}
func anthroClose(i int) string {
	return anthroFrame("content_block_stop", map[string]any{"type": "content_block_stop", "index": i})
}
func anthroFinish(reason string) string {
	return anthroFrame("message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": reason, "stop_sequence": nil}, "usage": map[string]any{"output_tokens": 4}})
}
func anthroStop() string { return anthroFrame("message_stop", map[string]any{"type": "message_stop"}) }
func anthroText(s string) string {
	return anthroBlock(0, map[string]any{"type": "text", "text": ""}) + anthroDelta(0, map[string]any{"type": "text_delta", "text": s}) + anthroClose(0)
}
func anthroConfig() core.Config { return core.Config{Model: "fixture-claude", OutputBudget: 100} }

func TestEncodeAnthropicToolRoundTrip(t *testing.T) {
	tools := []core.Tool{core.NewTool("read_file", "Read", map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"$ref": "https://uncontacted.invalid/schema"}}})}
	messages := []core.Message{{Role: "system", Content: "original system"}, {Role: "system", Content: "task data"}, {Role: "user", Content: "inspect"}, {Role: "assistant", Content: "checking", ToolCalls: []core.Call{anthroCall("tool_a", "read_file", `{"path":"a"}`), anthroCall("tool_b", "read_file", `{"path":"b"}`)}}, {Role: "tool", ToolCallID: "tool_b", Content: `{"error":true,"content":"denied"}`}, {Role: "tool", ToolCallID: "tool_a", Content: `{"error":false,"content":"text"}`}, {Role: "user", Content: "also summarize"}}
	b, err := EncodeAnthropic(anthroConfig(), messages, tools)
	if err != nil {
		t.Fatal(err)
	}
	var req map[string]any
	if err = json.Unmarshal(b, &req); err != nil {
		t.Fatal(err)
	}
	if req["model"] != "fixture-claude" || req["max_tokens"] != float64(100) || req["stream"] != true {
		t.Fatalf("request header: %s", b)
	}
	if _, ok := req["stream_options"]; ok {
		t.Fatal("Chat Completions field leaked")
	}
	if len(req["system"].([]any)) != 2 {
		t.Fatal("lost system blocks")
	}
	msgs := req["messages"].([]any)
	if len(msgs) != 3 {
		t.Fatalf("consecutive tool/user messages not merged: %s", b)
	}
	assistant := msgs[1].(map[string]any)["content"].([]any)
	if len(assistant) != 3 || assistant[1].(map[string]any)["type"] != "tool_use" || assistant[1].(map[string]any)["input"].(map[string]any)["path"] != "a" {
		t.Fatalf("tool_use: %s", b)
	}
	results := msgs[2].(map[string]any)["content"].([]any)
	if len(results) != 3 || results[0].(map[string]any)["tool_use_id"] != "tool_b" || results[0].(map[string]any)["is_error"] != true {
		t.Fatalf("tool_result: %s", b)
	}
	if !strings.Contains(string(b), "https://uncontacted.invalid/schema") || strings.Contains(string(b), "thinking") {
		t.Fatal("schema changed or unsupported option added")
	}
	// The caller's normalized messages and schema remain untouched.
	if messages[3].ToolCalls[0].Function.Arguments != `{"path":"a"}` || tools[0].Function.Parameters["type"] != "object" {
		t.Fatal("mutated normalized input")
	}
}

func TestEncodeAnthropicRejectsInvalidHistory(t *testing.T) {
	base := []core.Message{{Role: "user", Content: "x"}}
	cases := map[string][]core.Message{
		"no messages":       nil,
		"only system":       {{Role: "system", Content: "x"}},
		"unknown role":      {{Role: "developer", Content: "x"}},
		"mid system":        {{Role: "user", Content: "x"}, {Role: "system", Content: "late"}},
		"orphan result":     {{Role: "tool", ToolCallID: "missing", Content: "x"}},
		"unresolved":        {{Role: "assistant", ToolCalls: []core.Call{anthroCall("a", "tool", "{}")}}},
		"split result":      {{Role: "assistant", ToolCalls: []core.Call{anthroCall("a", "tool", "{}")}}, {Role: "user", Content: "x"}, {Role: "tool", ToolCallID: "a", Content: "x"}},
		"duplicate call":    {{Role: "assistant", ToolCalls: []core.Call{anthroCall("a", "tool", "{}"), anthroCall("a", "tool", "{}")}}},
		"duplicate result":  {{Role: "assistant", ToolCalls: []core.Call{anthroCall("a", "tool", "{}")}}, {Role: "tool", ToolCallID: "a", Content: "x"}, {Role: "tool", ToolCallID: "a", Content: "x"}},
		"invalid input":     {{Role: "assistant", ToolCalls: []core.Call{anthroCall("a", "tool", "[]")}}},
		"invalid utf8":      {{Role: "user", Content: string([]byte{0xff})}},
		"too much text":     {{Role: "user", Content: strings.Repeat("x", anthropicMaxText+1)}},
		"empty user":        {{Role: "user"}},
		"empty assistant":   {{Role: "assistant"}},
		"misplaced tool id": {{Role: "user", Content: "x", ToolCallID: "a"}},
		"user tool calls":   {{Role: "user", Content: "x", ToolCalls: []core.Call{anthroCall("a", "tool", "{}")}}},
	}
	for name, msgs := range cases {
		t.Run(name, func(t *testing.T) {
			if b, err := EncodeAnthropic(anthroConfig(), msgs, nil); err == nil || b != nil {
				t.Fatalf("accepted invalid input: %s %v", b, err)
			}
		})
	}
	if _, err := EncodeAnthropic(core.Config{Model: "x"}, base, nil); err == nil {
		t.Fatal("accepted missing budget")
	}
	typeBad := anthroCall("a", "tool", "{}")
	typeBad.Type = "server_tool_use"
	if _, err := EncodeAnthropic(anthroConfig(), []core.Message{{Role: "assistant", ToolCalls: []core.Call{typeBad}}}, nil); err == nil {
		t.Fatal("accepted non-client tool")
	}
}

func TestEncodeAnthropicBoundsAndSchemas(t *testing.T) {
	msg := []core.Message{{Role: "user", Content: "x"}}
	cycle := map[string]any{"type": "object"}
	cycle["cycle"] = cycle
	for _, schema := range []map[string]any{nil, {"type": "array"}, {"type": "object", "bad": func() {}}, cycle, {"type": "object", "description": strings.Repeat("x", anthropicMaxText+1)}, {"type": "object", "default": anthroMarshalTrap("x")}, {"type": "object", "default": anthroTextMarshalTrap("x")}} {
		tool := core.NewTool("tool", "desc", schema)
		if _, err := EncodeAnthropic(anthroConfig(), msg, []core.Tool{tool}); err == nil {
			t.Fatalf("accepted schema: %T", schema)
		}
	}
	tool := core.NewTool("tool", "desc", map[string]any{"type": "object"})
	if _, err := EncodeAnthropic(anthroConfig(), msg, []core.Tool{tool, tool}); err == nil {
		t.Fatal("duplicate tool definition")
	}
	tool.Type = "server_tool"
	if _, err := EncodeAnthropic(anthroConfig(), msg, []core.Tool{tool}); err == nil {
		t.Fatal("server tool definition")
	}
	msgs := make([]core.Message, 17)
	for i := range msgs {
		msgs[i] = core.Message{Role: "user", Content: strings.Repeat("x", anthropicMaxText)}
	}
	if _, err := EncodeAnthropic(anthroConfig(), msgs, nil); err == nil {
		t.Fatal("accepted excessive request")
	}
}

type anthroMarshalTrap string

func (anthroMarshalTrap) MarshalJSON() ([]byte, error) {
	panic("custom JSON marshaler must not execute")
}

type anthroTextMarshalTrap string

func (anthroTextMarshalTrap) MarshalText() ([]byte, error) {
	panic("custom text marshaler must not execute")
}

func TestAnthropicUnicodeSurrogates(t *testing.T) {
	for _, s := range []string{`{"value":"\ud83d\ude00"}`, `{"value":"\\ud800"}`, `{"value":"😀"}`} {
		if !anthropicObject([]byte(s)) {
			t.Fatalf("rejected safe Unicode: %s", s)
		}
	}
	for _, s := range []string{`{"value":"\ud800"}`, `{"value":"\udc00"}`, `{"value":"\ud800\u0041"}`, `{"value":"\ud800\\udc00"}`} {
		if anthropicObject([]byte(s)) {
			t.Fatalf("accepted lossy surrogate: %s", s)
		}
	}
	wire := anthroStart() + "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\\ud800\"}}\n\n"
	if _, err := ParseAnthropic(strings.NewReader(wire), nil); err == nil {
		t.Fatal("stream replaced unpaired surrogate")
	}
}

func TestAnthropicFragmentedTextAndUsage(t *testing.T) {
	unknown := anthroFrame("future_metadata", map[string]any{"type": "future_metadata", "extra": true})
	usage := anthroFrame("message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": nil}, "usage": map[string]any{"output_tokens": 2}})
	wire := ": comment\n\n" + unknown + anthroStart() + anthroFrame("ping", map[string]any{"type": "ping"}) + anthroText("こんにちは 😀") + usage + anthroFinish("end_turn") + anthroStop() + unknown
	wire = strings.ReplaceAll(wire, "\n", "\r\n")
	var streamed string
	r, err := ParseAnthropic(&byteReader{wire}, func(s string) { streamed += s })
	if err != nil || r.Message.Content != "こんにちは 😀" || streamed != r.Message.Content || r.Finish != "stop" {
		t.Fatalf("reply: %+v %v stream=%q", r, err, streamed)
	}
	if r.Usage == nil || r.Usage.Prompt != 12 || r.Usage.Completion != 4 || r.Usage.Total != 16 {
		t.Fatalf("usage: %+v", r.Usage)
	}
	// Multiple data lines form one JSON event; the event field is optional.
	wire = "data: {\"type\":\"message_start\",\n" + "data: \"message\":{\"role\":\"assistant\",\"content\":[]}}\n\n" + anthroText("ok") + anthroFinish("stop_sequence") + strings.TrimSuffix(anthroStop(), "\n\n")
	if r, err = ParseAnthropic(strings.NewReader(wire), nil); err != nil || r.Message.Content != "ok" {
		t.Fatalf("multiline/EOF framing: %+v %v", r, err)
	}
}

func TestAnthropicToolAssembly(t *testing.T) {
	wire := anthroStart() + anthroText("I will inspect ")
	wire += anthroBlock(1, map[string]any{"type": "tool_use", "id": "a", "name": "read_file", "input": map[string]any{}})
	for _, part := range []string{`{"path":`, `"a😀.go","emoji":"\ud83d`, `\ude00"}`} {
		wire += anthroDelta(1, map[string]any{"type": "input_json_delta", "partial_json": part})
	}
	wire += anthroClose(1) + anthroBlock(2, map[string]any{"type": "text", "text": "then another"}) + anthroClose(2)
	wire += anthroBlock(3, map[string]any{"type": "tool_use", "id": "b", "name": "list_files", "input": map[string]any{"path": "."}}) + anthroClose(3) + anthroFinish("tool_use") + anthroStop()
	r, err := ParseAnthropic(&byteReader{wire}, nil)
	if err != nil || r.Finish != "tool_calls" || len(r.Message.ToolCalls) != 2 || r.Message.Content != "I will inspect then another" {
		t.Fatalf("tools: %+v %v", r, err)
	}
	a := r.Message.ToolCalls[0]
	if a.ID != "a" || a.Index != 0 || a.Type != "function" || a.Function.Name != "read_file" {
		t.Fatalf("tool identity: %+v", a)
	}
	var input map[string]string
	if err = json.Unmarshal([]byte(a.Function.Arguments), &input); err != nil || input["path"] != "a😀.go" || input["emoji"] != "😀" {
		t.Fatalf("input: %+v %v %s", input, err, a.Function.Arguments)
	}
	if r.Message.ToolCalls[1].Index != 1 || r.Message.ToolCalls[1].Function.Arguments != `{"path":"."}` {
		t.Fatalf("nonempty initial input: %+v", r.Message.ToolCalls[1])
	}
}

func TestAnthropicStrictLifecycle(t *testing.T) {
	textStart := anthroBlock(0, map[string]any{"type": "text", "text": "x"})
	tool := anthroBlock(0, map[string]any{"type": "tool_use", "id": "a", "name": "tool", "input": map[string]any{}})
	cases := map[string]string{
		"no message start":      anthroText("x") + anthroFinish("end_turn") + anthroStop(),
		"duplicate start":       anthroStart() + anthroStart(),
		"missing stop":          anthroStart() + anthroText("x") + anthroFinish("end_turn"),
		"missing finish":        anthroStart() + anthroText("x") + anthroStop(),
		"dangling block":        anthroStart() + textStart + anthroFinish("end_turn") + anthroStop(),
		"delta without start":   anthroStart() + anthroDelta(0, map[string]any{"type": "text_delta", "text": "x"}),
		"wrong index":           anthroStart() + anthroBlock(1, map[string]any{"type": "text", "text": "x"}),
		"overlapping blocks":    anthroStart() + textStart + anthroBlock(1, map[string]any{"type": "text", "text": "x"}),
		"duplicate block stop":  anthroStart() + anthroText("x") + anthroClose(0),
		"delta after close":     anthroStart() + anthroText("x") + anthroDelta(0, map[string]any{"type": "text_delta", "text": "late"}),
		"duplicate finish":      anthroStart() + anthroText("x") + anthroFinish("end_turn") + anthroFinish("end_turn"),
		"content after finish":  anthroStart() + anthroFinish("end_turn") + textStart,
		"content after stop":    anthroStart() + anthroFinish("end_turn") + anthroStop() + textStart,
		"duplicate stop":        anthroStart() + anthroFinish("end_turn") + anthroStop() + anthroStop(),
		"tool mismatch":         anthroStart() + tool + anthroClose(0) + anthroFinish("end_turn") + anthroStop(),
		"missing tool":          anthroStart() + anthroFinish("tool_use") + anthroStop(),
		"text delta into tool":  anthroStart() + tool + anthroDelta(0, map[string]any{"type": "text_delta", "text": "bad"}),
		"input delta into text": anthroStart() + textStart + anthroDelta(0, map[string]any{"type": "input_json_delta", "partial_json": "{}"}),
		"duplicate id":          anthroStart() + tool + anthroClose(0) + anthroBlock(1, map[string]any{"type": "tool_use", "id": "a", "name": "tool", "input": map[string]any{}}),
		"header mismatch":       "event: wrong\ndata: {\"type\":\"ping\"}\n\n",
	}
	for name, wire := range cases {
		t.Run(name, func(t *testing.T) {
			r, err := ParseAnthropic(strings.NewReader(wire), nil)
			if err == nil || len(r.Message.ToolCalls) > 0 {
				t.Fatalf("accepted invalid lifecycle: %+v %v", r, err)
			}
		})
	}
}

func TestAnthropicUnsupportedContentAndFinish(t *testing.T) {
	for _, kind := range []string{"thinking", "redacted_thinking", "image", "server_tool_use", "web_search_tool_result", "web_fetch_tool_result", "code_execution_tool_result"} {
		wire := anthroStart() + anthroBlock(0, map[string]any{"type": kind})
		if _, err := ParseAnthropic(strings.NewReader(wire), nil); err == nil {
			t.Errorf("accepted block %s", kind)
		}
	}
	for _, kind := range []string{"thinking_delta", "signature_delta", "citations_delta", "future_content_delta"} {
		wire := anthroStart() + anthroBlock(0, map[string]any{"type": "text", "text": ""}) + anthroDelta(0, map[string]any{"type": kind})
		if _, err := ParseAnthropic(strings.NewReader(wire), nil); err == nil {
			t.Errorf("accepted delta %s", kind)
		}
	}
	for _, reason := range []string{"pause_turn", "future_reason", ""} {
		if _, err := ParseAnthropic(strings.NewReader(anthroStart()+anthroFinish(reason)+anthroStop()), nil); err == nil {
			t.Errorf("accepted finish %q", reason)
		}
	}
	initial := "data: {\"type\":\"message_start\",\"message\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"dropped\"}]}}\n\n"
	if _, err := ParseAnthropic(strings.NewReader(initial), nil); err == nil {
		t.Fatal("ignored initial content")
	}
}

func TestAnthropicFinishErrorsAndPartialTools(t *testing.T) {
	for reason, want := range map[string]string{"max_tokens": "length", "refusal": "content_filter"} {
		r, err := ParseAnthropic(strings.NewReader(anthroStart()+anthroText("partial")+anthroFinish(reason)+anthroStop()), nil)
		var finishErr *FinishError
		if !errors.As(err, &finishErr) || finishErr.Reason != want || len(r.Message.ToolCalls) > 0 || r.Message.Content != "" {
			t.Fatalf("finish: %+v %v", r, err)
		}
	}
	partial := anthroStart() + anthroBlock(0, map[string]any{"type": "tool_use", "id": "a", "name": "tool", "input": map[string]any{}}) + anthroDelta(0, map[string]any{"type": "input_json_delta", "partial_json": `{"path":`}) + anthroClose(0)
	r, err := ParseAnthropic(strings.NewReader(partial+anthroFinish("max_tokens")+anthroStop()), nil)
	var finishErr *FinishError
	if !errors.As(err, &finishErr) || finishErr.Reason != "length" || len(r.Message.ToolCalls) != 0 {
		t.Fatalf("truncated tool: %+v %v", r, err)
	}
	if _, err = ParseAnthropic(strings.NewReader(partial+anthroFinish("tool_use")+anthroStop()), nil); err == nil {
		t.Fatal("accepted invalid completed JSON")
	}
}

type anthroErrorReader struct{ err error }

func (r anthroErrorReader) Read([]byte) (int, error) { return 0, r.err }

func TestAnthropicErrorRedactionAndCancellation(t *testing.T) {
	secret := "upstream-secret-credential"
	for _, wire := range []string{anthroFrame("error", map[string]any{"type": "error", "error": map[string]any{"message": secret, "type": secret}}), "data: {" + secret + "}\n\n", anthroStart() + "data: {\"type\":123,\"secret\":\"" + secret + "\"}\n\n"} {
		_, err := ParseAnthropic(strings.NewReader(wire), nil)
		if err == nil || strings.Contains(err.Error(), secret) {
			t.Fatalf("unsanitized error: %v", err)
		}
	}
	_, err := ParseAnthropic(anthroErrorReader{errors.New(secret)}, nil)
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("reader error leak: %v", err)
	}
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		stream := io.MultiReader(strings.NewReader(anthroStart()+anthroBlock(0, map[string]any{"type": "text", "text": "partial"})), anthroErrorReader{cause})
		var rendered string
		r, err := ParseAnthropic(stream, func(s string) { rendered += s })
		if !errors.Is(err, cause) || rendered != "partial" || len(r.Message.ToolCalls) != 0 {
			t.Fatalf("cancel handling: %+v %v %q", r, err, rendered)
		}
	}
}

func TestAnthropicPayloadAndUsageBounds(t *testing.T) {
	for _, wire := range []string{
		anthroStart() + anthroText(strings.Repeat("x", anthropicMaxText+1)) + anthroFinish("end_turn") + anthroStop(),
		"data: " + strings.Repeat("x", anthropicMaxEvent+1) + "\n\n",
		"data: {\"type\":\"ping\",\"x\":\"" + string([]byte{0xff}) + "\"}\n\n",
		anthroStart() + anthroFrame("message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": "end_turn"}, "usage": map[string]any{"output_tokens": -1}}) + anthroStop(),
		anthroStart() + anthroFrame("message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": "end_turn"}, "usage": map[string]any{"output_tokens": 1.5}}) + anthroStop(),
		anthroStart() + anthroFrame("message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": "end_turn"}, "usage": map[string]any{"input_tokens": 1000000000, "cache_read_input_tokens": 1000000000, "output_tokens": 1}}) + anthroStop(),
		anthroStart() + anthroBlock(0, map[string]any{"type": "tool_use", "id": "a", "name": "tool", "input": []any{}}),
		anthroStart() + anthroBlock(0, map[string]any{"type": "tool_use", "id": "a", "name": "tool", "input": map[string]any{"x": 1}}) + anthroDelta(0, map[string]any{"type": "input_json_delta", "partial_json": "{}"}),
	} {
		if _, err := ParseAnthropic(strings.NewReader(wire), nil); err == nil {
			t.Fatal("accepted invalid or excessive payload")
		}
	}
	// Many small valid events cannot evade the whole-response bound.
	ping := anthroFrame("ping", map[string]any{"type": "ping"})
	wire := anthroStart() + strings.Repeat(ping, maxResponse/len(ping)+1) + anthroFinish("end_turn") + anthroStop()
	if _, err := ParseAnthropic(strings.NewReader(wire), nil); err == nil {
		t.Fatal("accepted excessive stream")
	}
}

func FuzzAnthropicSSE(f *testing.F) {
	f.Add(anthroStart() + anthroText("hello") + anthroFinish("end_turn") + anthroStop())
	f.Add("data: {}\n\n")
	f.Add(anthroStart() + anthroBlock(0, map[string]any{"type": "tool_use", "id": "a", "name": "tool", "input": map[string]any{}}) + anthroClose(0) + anthroFinish("tool_use") + anthroStop())
	f.Fuzz(func(t *testing.T, wire string) {
		if len(wire) > 1<<16 {
			t.Skip()
		}
		r, err := ParseAnthropic(strings.NewReader(wire), nil)
		if err != nil && len(r.Message.ToolCalls) > 0 {
			t.Fatal("partial tool calls escaped on failure")
		}
	})
}
