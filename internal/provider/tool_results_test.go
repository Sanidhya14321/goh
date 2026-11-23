package provider

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func envelope(t *testing.T, inner any, failed bool) string {
	t.Helper()
	body, ok := inner.(string)
	if !ok {
		b, err := json.Marshal(inner)
		if err != nil {
			t.Fatal(err)
		}
		body = string(b)
	}
	b, _ := json.Marshal(map[string]any{"content": body, "error": failed})
	return string(b)
}

// Tool results reach the model as plain text: the main text verbatim (not
// escaped twice), one compact line of informative fields, lists one entry
// per line, errors marked. Receipts and other formats pass through.
func TestToolResultText(t *testing.T) {
	file := "package main\n\nfunc f() string {\n\treturn \"<x>\"\n}\n"
	for _, c := range []struct {
		name, in, want string
		failed         bool
	}{
		{"read_file", envelope(t, map[string]any{"content": file, "path": "m.go", "sha256": "ab12", "start_line": 1, "end_line": 6, "total_lines": 6, "truncated": false}, false),
			`{"end_line":6,"path":"m.go","sha256":"ab12","start_line":1,"total_lines":6}` + "\n" + file, false},
		{"grep", envelope(t, map[string]any{"matches": []map[string]any{{"path": "m.go", "line": 4, "text": "\treturn x"}, {"path": "n.go", "line": 9, "text": "y"}}, "truncated": true}, false),
			"m.go:4: \treturn x\nn.go:9: y\n{\"truncated\":true}", false},
		{"grep none", envelope(t, map[string]any{"matches": []any{}, "truncated": false}, false), "no matches", false},
		{"list_files", envelope(t, map[string]any{"files": []string{"a.go", "b/c.go"}, "truncated": false}, false), "a.go\nb/c.go", false},
		{"shell", envelope(t, map[string]any{"artifact": "/abs/path/shell-1.log", "artifact_id": "shell-1.log", "exit_code": 0, "output": "hi\n", "status": "completed", "truncated": false, "preview_truncated": false}, false),
			`{"artifact_id":"shell-1.log","exit_code":0,"status":"completed"}` + "\nhi\n", false},
		{"shell failure", envelope(t, map[string]any{"exit_code": 1, "output": "FAIL x_test.go:3\n", "status": "completed"}, true),
			"Error: {\"exit_code\":1,\"status\":\"completed\"}\nFAIL x_test.go:3\n", true},
		{"plain error", envelope(t, "tool \"create_file\" is disabled by read-only plan mode", true), "Error: tool \"create_file\" is disabled by read-only plan mode", true},
		{"error first", `{"error":true,"content":"denied"}`, "Error: denied", true},
		{"edit", envelope(t, map[string]any{"bytes": 10, "created": false, "path": "a.go", "sha256": "cd34"}, false), `{"bytes":10,"path":"a.go","sha256":"cd34"}`, false},
	} {
		got, failed, ok := ToolResultText(c.in)
		if !ok || got != c.want || failed != c.failed {
			t.Errorf("%s:\n got %q (error %v, ok %v)\nwant %q (error %v)", c.name, got, failed, ok, c.want, c.failed)
		}
		if again, _, _ := ToolResultText(c.in); again != got {
			t.Errorf("%s: rendering is not stable", c.name)
		}
		if len(got) > len(c.in) {
			t.Errorf("%s: rendering grew %d -> %d bytes", c.name, len(c.in), len(got))
		}
	}
	for _, unchanged := range []string{`{"masked":true,"tool":"read_file","bytes":9000}`, "plain text", `["a"]`, `{"content":5}`} {
		if got, failed, ok := ToolResultText(unchanged); ok || failed || got != unchanged {
			t.Errorf("%q changed to %q", unchanged, got)
		}
	}
}

// Every wire format sends the plain text; Anthropic keeps is_error.
func TestToolResultsOnTheWire(t *testing.T) {
	call := core.Call{ID: "call_1", Type: "function"}
	call.Function.Name, call.Function.Arguments = "read_file", `{"path":"m.go"}`
	result := envelope(t, map[string]any{"content": "line \"one\"\n", "path": "m.go", "sha256": "ab"}, false)
	failed := envelope(t, "no such file", true)
	history := []core.Message{{Role: "system", Content: "s"}, {Role: "user", Content: "u"},
		{Role: "assistant", ToolCalls: []core.Call{call}}, {Role: "tool", ToolCallID: "call_1", Content: result}}
	want := `{"path":"m.go","sha256":"ab"}` + "\nline \"one\"\n"

	wire := chatWireMessages(history)
	if got := wire[3].(core.Message).Content; got != want {
		t.Fatalf("chat: %q", got)
	}
	b, err := encodeResponses(core.Config{Model: "m"}, history, nil)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Input []map[string]any `json:"input"`
	}
	json.Unmarshal(b, &body)
	if got := body.Input[len(body.Input)-1]["output"]; got != want {
		t.Fatalf("responses: %q", got)
	}

	anthropicCall := call
	anthropicCall.ID = "toolu_1"
	history[2].ToolCalls = []core.Call{anthropicCall}
	history[3] = core.Message{Role: "tool", ToolCallID: "toolu_1", Content: failed}
	b, err = EncodeAnthropic(core.Config{Model: "claude", OutputBudget: 100}, history, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"content":"Error: no such file","is_error":true`) {
		t.Fatalf("anthropic: %s", b)
	}
}
