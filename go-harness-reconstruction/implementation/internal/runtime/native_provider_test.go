package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"example.com/independent-harness/internal/core"
	api "example.com/independent-harness/internal/provider"
	"example.com/independent-harness/internal/tools"
)

// This fixture writes protocol frames independently of the product encoder/parser.
func nativeFrame(w io.Writer, kind string, payload any) {
	raw, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, raw)
}
func nativeStart(w io.Writer) {
	nativeFrame(w, "message_start", map[string]any{"type": "message_start", "message": map[string]any{"role": "assistant", "content": []any{}, "usage": map[string]int{"input_tokens": 12, "output_tokens": 0}}})
}
func nativeStop(w io.Writer, reason string) {
	nativeFrame(w, "message_delta", map[string]any{"type": "message_delta", "delta": map[string]string{"stop_reason": reason}, "usage": map[string]int{"output_tokens": 7}})
	nativeFrame(w, "message_stop", map[string]any{"type": "message_stop"})
}
func nativeText(w io.Writer, text string) {
	nativeStart(w)
	nativeFrame(w, "content_block_start", map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]string{"type": "text", "text": ""}})
	nativeFrame(w, "content_block_delta", map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]string{"type": "text_delta", "text": text}})
	nativeFrame(w, "content_block_stop", map[string]any{"type": "content_block_stop", "index": 0})
	nativeStop(w, "end_turn")
}
func nativeTool(w io.Writer, name, args string, complete bool) {
	nativeStart(w)
	nativeFrame(w, "content_block_start", map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]any{"type": "tool_use", "id": "native-call", "name": name, "input": map[string]any{}}})
	for _, part := range []string{args[:len(args)/2], args[len(args)/2:]} {
		nativeFrame(w, "content_block_delta", map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]string{"type": "input_json_delta", "partial_json": part}})
	}
	nativeFrame(w, "content_block_stop", map[string]any{"type": "content_block_stop", "index": 0})
	if complete {
		nativeStop(w, "tool_use")
	}
}
func chatReply(w io.Writer, call *core.Call, text string) {
	delta := map[string]any{"content": text}
	reason := "stop"
	if call != nil {
		delta = map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": call.ID, "type": "function", "function": call.Function}}}
		reason = "tool_calls"
	}
	raw, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": delta}}})
	fmt.Fprintf(w, "data: %s\n\n", raw)
	raw, _ = json.Marshal(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": reason}}})
	fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", raw)
}
func TestMixedProtocolChildRoleAndTypedRoundTrip(t *testing.T) {
	a, root := testAgent(t)
	os.WriteFile(filepath.Join(root, "target"), []byte("local evidence"), 0600)
	t.Setenv("PARENT_FIXTURE_KEY", "parent-key")
	t.Setenv("ANTHROPIC_API_KEY", "native-key")
	parentCalls, childCalls := atomic.Int32{}, atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if r.URL.Path == "/messages" {
			if r.Header.Get("x-api-key") != "native-key" || r.Header.Get("Authorization") != "" || r.Header.Get("anthropic-version") != "2023-06-01" || request["model"] != "native-model" {
				t.Error("native routing/header mismatch")
			}
			if childCalls.Add(1) == 1 {
				nativeTool(w, "read_file", `{"path":"target"}`, true)
			} else {
				messages := request["messages"].([]any)
				last := messages[len(messages)-1].(map[string]any)
				blocks := last["content"].([]any)
				if last["role"] != "user" || blocks[0].(map[string]any)["type"] != "tool_result" || !strings.Contains(blocks[0].(map[string]any)["content"].(string), "local evidence") {
					t.Error("tool result not normalized", last)
				}
				nativeText(w, `{"observed":"local evidence"}`)
			}
			return
		}
		if r.Header.Get("Authorization") != "Bearer parent-key" || r.Header.Get("x-api-key") != "" || request["model"] != "parent-model" {
			t.Error("parent routing/header mismatch")
		}
		if parentCalls.Add(1) == 1 {
			call := toolCall("mixed-task", "task", map[string]any{"description": "Typed native investigation", "prompt": "Read target and report JSON", "subagent_type": "general", "output_schema": map[string]any{"type": "object", "properties": map[string]any{"observed": map[string]any{"const": "local evidence"}}, "required": []string{"observed"}, "additionalProperties": false}})
			chatReply(w, &call, "")
		} else {
			messages := request["messages"].([]any)
			last := messages[len(messages)-1].(map[string]any)
			var envelope tools.Result
			if err := json.Unmarshal([]byte(last["content"].(string)), &envelope); err != nil || envelope.Error {
				t.Error("child result failed", envelope, err)
			}
			if !strings.Contains(envelope.Content, `"structured_output":{"observed":"local evidence"}`) {
				t.Error(envelope.Content)
			}
			chatReply(w, nil, "Mixed protocol task complete.")
		}
	}))
	defer server.Close()
	protocol, endpoint, model := "anthropic_messages", server.URL+"/messages", "native-model"
	a.Config.Endpoint = server.URL + "/chat"
	a.Config.Model = "parent-model"
	a.Config.APIKeyEnv = "PARENT_FIXTURE_KEY"
	a.Config.Profiles = map[string]core.ModelProfile{"native": {Protocol: &protocol, Endpoint: &endpoint, Model: &model}}
	a.Config.ModelRoles = map[string]string{"task": "native"}
	a.Grants["task"] = true
	a.Provider = api.New(a.Config)
	status, err := a.Run(context.Background(), "delegate typed investigation")
	if err != nil || status != "completed" || parentCalls.Load() != 2 || childCalls.Load() != 2 || len(a.Store.Pending()) != 0 {
		t.Fatal(status, err, parentCalls.Load(), childCalls.Load())
	}
}
func TestNativePartialToolNeverExecutes(t *testing.T) {
	a, root := testAgent(t)
	calls := atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		nativeTool(w, "create_file", `{"path":"uncommitted","content":"must not publish"}`, false)
	}))
	defer server.Close()
	a.Config.Endpoint = server.URL
	a.Config.Protocol = "anthropic_messages"
	a.Provider = api.New(a.Config)
	a.Grants["create_file"] = true
	status, err := a.Run(context.Background(), "partial provider stream")
	if err == nil || status != "failed" || calls.Load() != 1 {
		t.Fatal(status, err, calls.Load())
	}
	if _, err = os.Stat(filepath.Join(root, "uncommitted")); !os.IsNotExist(err) {
		t.Fatal("partial tool was executed")
	}
	if len(a.Store.Pending()) != 0 {
		t.Fatal(a.Store.Pending())
	}
}
