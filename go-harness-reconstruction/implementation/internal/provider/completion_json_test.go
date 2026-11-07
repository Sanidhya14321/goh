package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestChatCompatibilityFallbackKeepsToolsAndText(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		var body struct {
			StreamOptions json.RawMessage `json:"stream_options"`
			Tools         []core.Tool     `json:"tools"`
			Stream        bool            `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !body.Stream || len(body.Tools) != 1 {
			t.Errorf("request changed shape or lost tools: %v", err)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if attempt == 1 {
			if len(body.StreamOptions) == 0 {
				t.Error("missing initial stream options")
			}
			w.WriteHeader(400)
			io.WriteString(w, `{"error":{"code":"unsupported_parameter","param":"stream_options","message":"PRIVATE"}}`)
			return
		}
		if len(body.StreamOptions) != 0 {
			t.Error("retry retained rejected option")
		}
		io.WriteString(w, `{"choices":[{"index":0,"finish_reason":"tool_calls","message":{"role":"assistant","content":"Inspecting","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{}"}}]}}],"usage":{"total_tokens":14}}`)
	}))
	defer server.Close()
	deltas := ""
	reply, err := New(core.Config{Endpoint: server.URL}).Complete(context.Background(), nil,
		[]core.Tool{core.NewTool("read_file", "", nil)}, func(s string) { deltas += s })
	if err != nil || attempt != 2 || deltas != "Inspecting" || len(reply.Message.ToolCalls) != 1 ||
		reply.Usage == nil || reply.Usage.Total != 14 {
		t.Fatalf("unexpected compatible reply: %v %d %q %+v", err, attempt, deltas, reply)
	}
}

func TestMalformedJSONCompletionNeverEmitsPartialText(t *testing.T) {
	for _, input := range []string{
		`{"error":{"message":"PRIVATE"},"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"partial"}}]}`,
		`{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":"partial","tool_calls":[{"id":"x","type":"function","function":{"name":"read_file"}},{"id":"x","type":"function","function":{"name":"read_file"}}]}}]}`,
		`{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"partial"}}]} {}`,
	} {
		_, err := parseJSONCompletion(strings.NewReader(input), func(string) { t.Error("invalid reply emitted text") })
		if err == nil || strings.Contains(err.Error(), "PRIVATE") {
			t.Fatal(err)
		}
	}
}
