package provider

import (
	"context"
	"encoding/json"
	"example.com/independent-harness/internal/core"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func frame(v any) string { b, _ := json.Marshal(v); return "data: " + string(b) + "\n\n" }
func textFrame(s string) string {
	return frame(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": s}}}})
}
func finish(reason string) string {
	return frame(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": reason}}})
}

type byteReader struct{ s string }

func (r *byteReader) Read(p []byte) (int, error) {
	if r.s == "" {
		return 0, io.EOF
	}
	p[0] = r.s[0]
	r.s = r.s[1:]
	return 1, nil
}
func TestFragmentedStream(t *testing.T) {
	wire := textFrame("こんにちは ✓") + finish("stop") + frame(map[string]any{"choices": []any{}, "usage": map[string]any{"total_tokens": 7}}) + "data: [DONE]\r\n\r\n"
	var deltas string
	r, e := Parse(&byteReader{wire}, func(s string) { deltas += s })
	if e != nil || r.Message.Content != "こんにちは ✓" || deltas != r.Message.Content || r.Usage.Total != 7 {
		t.Fatalf("%+v %v %q", r, e, deltas)
	}
}
func TestToolAssembly(t *testing.T) {
	wire := ""
	for _, args := range []string{`{"path":`, `"x.go"}`} {
		d := map[string]any{"index": 0, "function": map[string]any{"arguments": args}}
		if wire == "" {
			d["id"] = "call1"
			d["type"] = "function"
			d["function"].(map[string]any)["name"] = "read_file"
		}
		wire += frame(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{d}}}}})
	}
	r, e := Parse(strings.NewReader(wire+finish("tool_calls")+"data: [DONE]\n\n"), nil)
	if e != nil || len(r.Message.ToolCalls) != 1 || r.Message.ToolCalls[0].Function.Arguments != `{"path":"x.go"}` {
		t.Fatalf("%+v %v", r, e)
	}
}
func TestFailureStreams(t *testing.T) {
	for _, wire := range []string{textFrame("partial"), finish("length") + "data: [DONE]\n\n", `data: {bad}` + "\n\n", frame(map[string]any{"error": map[string]any{"message": "error"}}), finish("stop"), finish("tool_calls") + "data: [DONE]\n\n", finish("stop") + textFrame("late") + "data: [DONE]\n\n"} {
		if _, e := Parse(strings.NewReader(wire), nil); e == nil {
			t.Fatalf("accepted %q", wire)
		}
	}
}
func TestRetryAndNoPartialReplay(t *testing.T) {
	var n atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) < 3 {
			w.WriteHeader(503)
			return
		}
		io.WriteString(w, textFrame("ok")+finish("stop")+"data: [DONE]\n\n")
	}))
	defer s.Close()
	c := New(core.Config{Endpoint: s.URL, Model: "fixture"})
	if _, e := c.Complete(context.Background(), nil, nil, nil); e != nil || n.Load() != 3 {
		t.Fatalf("%d %v", n.Load(), e)
	}
	n.Store(0)
	s.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { n.Add(1); io.WriteString(w, textFrame("partial")) })
	if _, e := c.Complete(context.Background(), nil, nil, nil); e == nil || n.Load() != 1 {
		t.Fatalf("partial replay: %d %v", n.Load(), e)
	}
}
func FuzzSSE(f *testing.F) {
	f.Add(textFrame("hi") + finish("stop") + "data: [DONE]\n\n")
	f.Add("data: {}\n\n")
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 1<<16 {
			t.Skip()
		}
		Parse(strings.NewReader(s), nil)
	})
}

func TestDuplicateToolIDsRejected(t *testing.T) {
	calls := []any{map[string]any{"index": 0, "id": "duplicate", "type": "function", "function": map[string]any{"name": "read_file", "arguments": "{}"}}, map[string]any{"index": 1, "id": "duplicate", "type": "function", "function": map[string]any{"name": "read_file", "arguments": "{}"}}}
	wire := frame(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"tool_calls": calls}}}}) + finish("tool_calls") + "data: [DONE]\n\n"
	if _, e := Parse(strings.NewReader(wire), nil); e == nil {
		t.Fatal("accepted duplicate call IDs")
	}
}
func TestCancelledRequestIsNotRetried(t *testing.T) {
	var n atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { n.Add(1); w.WriteHeader(503) }))
	defer s.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := New(core.Config{Endpoint: s.URL, Model: "fixture"})
	if _, e := c.Complete(ctx, nil, nil, nil); e == nil {
		t.Fatal("cancelled request succeeded")
	}
	if n.Load() != 0 {
		t.Fatal("started cancelled network request")
	}
}
