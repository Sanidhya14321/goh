package provider

import (
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// A tool call being streamed is reported with its name, size so far and the
// file path as soon as the arguments contain it, for each wire format.
func TestToolCallProgress(t *testing.T) {
	frame := func(s string) string { return "data: " + s + "\n\n" }
	streams := map[string]func(*strings.Reader, func(core.ToolProgress)) error{
		"responses": func(r *strings.Reader, p func(core.ToolProgress)) error {
			_, err := parseResponsesStream(r, nil, nil, p)
			return err
		},
		"chat": func(r *strings.Reader, p func(core.ToolProgress)) error {
			_, err := parseChat(r, nil, p)
			return err
		},
		"anthropic": func(r *strings.Reader, p func(core.ToolProgress)) error {
			_, err := parseAnthropicStream(r, nil, nil, p)
			return err
		},
	}
	chunks := []string{`{"path":"web/ind`, `ex.html","content":"<html>`, strings.Repeat("x", 500), `</html>"}`}
	bodies := map[string]string{}
	var b strings.Builder
	b.WriteString(frame(`{"type":"response.output_item.added","item":{"type":"function_call","name":"create_file","call_id":"c1"}}`))
	for _, c := range chunks {
		b.WriteString(frame(`{"type":"response.function_call_arguments.delta","delta":` + quote(c) + `}`))
	}
	b.WriteString(frame(`{"type":"response.completed","response":{"status":"completed","output":[{"type":"function_call","call_id":"c1","name":"create_file","arguments":` + quote(strings.Join(chunks, "")) + `}]}}`))
	bodies["responses"] = b.String()
	b.Reset()
	for i, c := range chunks {
		head := `"index":0,"function":{"arguments":` + quote(c) + `}`
		if i == 0 {
			head = `"index":0,"id":"c1","type":"function","function":{"name":"create_file","arguments":` + quote(c) + `}`
		}
		b.WriteString(frame(`{"choices":[{"index":0,"delta":{"tool_calls":[{` + head + `}]}}]}`))
	}
	b.WriteString(frame(`{"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`) + frame("[DONE]"))
	bodies["chat"] = b.String()
	b.Reset()
	ev := func(name, data string) { b.WriteString("event: " + name + "\ndata: " + data + "\n\n") }
	ev("message_start", `{"type":"message_start","message":{"id":"m","type":"message","role":"assistant","content":[],"model":"x","usage":{"input_tokens":1,"output_tokens":0}}}`)
	ev("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_01","name":"create_file","input":{}}}`)
	for _, c := range chunks {
		ev("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":`+quote(c)+`}}`)
	}
	ev("content_block_stop", `{"type":"content_block_stop","index":0}`)
	ev("message_delta", `{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":5}}`)
	ev("message_stop", `{"type":"message_stop"}`)
	bodies["anthropic"] = b.String()

	for name, parse := range streams {
		var seen []core.ToolProgress
		if err := parse(strings.NewReader(bodies[name]), func(p core.ToolProgress) { seen = append(seen, p) }); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		last := seen[len(seen)-1]
		if seen[0].Tool != "create_file" || last.Target != "web/index.html" || last.Chars != len(strings.Join(chunks, "")) {
			t.Fatalf("%s: %+v … %+v", name, seen[0], last)
		}
		if seen[1].Target != "" {
			t.Fatalf("%s: target guessed from a partial path: %+v", name, seen[1])
		}
	}
}

func quote(s string) string {
	b := strings.Builder{}
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('"')
	return b.String()
}
