package provider

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// Tool results are journaled as a JSON envelope ({"content": ..., "error":
// ...}) whose content is itself the tool's JSON. Sent as is, a file's text
// reaches the model escaped twice (a newline as \\n, a quote as \\\"), and
// every result carries bookkeeping fields such as "truncated": false. Results
// stay in the history and are resent with every later request, so that
// overhead is paid many times. ToolResultText renders a result as the model
// reads it best: the main text verbatim, after one compact line of the
// fields that carry information. Only the wire request changes: the runtime,
// journal, exports, resume and the UI keep the envelope.
//
// The rendering is a pure function of the stored content, so a request's
// prefix stays byte-identical across turns and prompt caches keep hitting.
func ToolResultText(content string) (text string, isError bool, ok bool) {
	var envelope struct {
		Content *string `json:"content"`
		Error   bool    `json:"error"`
	}
	if !strings.HasPrefix(content, "{") || json.Unmarshal([]byte(content), &envelope) != nil || envelope.Content == nil {
		return content, false, false // a receipt, a summary, or another tool's own format
	}
	body := *envelope.Content
	prefix := ""
	if envelope.Error {
		prefix = "Error: "
	}
	fields, ok := decodeObject(body)
	if !ok {
		return prefix + body, envelope.Error, true
	}
	// List-shaped results read best one entry per line.
	if lines, ok := grepLines(fields["matches"]); ok {
		delete(fields, "matches")
		if len(lines) == 0 {
			lines = []string{"no matches"}
		}
		return prefix + joinNonEmpty(strings.Join(lines, "\n"), compactFields(fields)), envelope.Error, true
	}
	if names, ok := stringList(fields["files"]); ok {
		delete(fields, "files")
		if len(names) == 0 {
			names = []string{"no files"}
		}
		return prefix + joinNonEmpty(strings.Join(names, "\n"), compactFields(fields)), envelope.Error, true
	}
	if _, ok := fields["artifact_id"]; ok {
		delete(fields, "artifact") // the absolute log path; read_artifact takes the id
	}
	for _, key := range []string{"content", "output", "text"} {
		var main string
		if raw, ok := fields[key]; ok && json.Unmarshal(raw, &main) == nil {
			delete(fields, key)
			head := compactFields(fields)
			if main == "" {
				return prefix + head, envelope.Error, true
			}
			if head == "" {
				return prefix + main, envelope.Error, true
			}
			return prefix + head + "\n" + main, envelope.Error, true
		}
	}
	return prefix + compactFields(fields), envelope.Error, true
}

func decodeObject(body string) (map[string]json.RawMessage, bool) {
	if !strings.HasPrefix(strings.TrimSpace(body), "{") {
		return nil, false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal([]byte(body), &fields) != nil {
		return nil, false
	}
	return fields, true
}

// compactFields is one JSON line of the fields that say something: false,
// null, empty strings and empty lists are left out (their absence means the
// same), numbers and true flags are kept. Keys are sorted, so the line is
// stable.
func compactFields(fields map[string]json.RawMessage) string {
	kept := map[string]json.RawMessage{}
	for key, raw := range fields {
		switch strings.TrimSpace(string(raw)) {
		case "false", "null", `""`, "[]", "{}":
			continue
		}
		kept[key] = raw
	}
	if len(kept) == 0 {
		return ""
	}
	// No HTML escaping: "<" stays "<", not "\u003c".
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if enc.Encode(kept) != nil {
		return ""
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// grepLines renders search matches as "path:line: text".
func grepLines(raw json.RawMessage) ([]string, bool) {
	if raw == nil {
		return nil, false
	}
	var matches []struct {
		Path *string `json:"path"`
		Line *int    `json:"line"`
		Text *string `json:"text"`
	}
	if json.Unmarshal(raw, &matches) != nil {
		return nil, false
	}
	lines := make([]string, 0, len(matches))
	for _, m := range matches {
		if m.Path == nil || m.Line == nil || m.Text == nil {
			return nil, false
		}
		lines = append(lines, *m.Path+":"+strconv.Itoa(*m.Line)+": "+*m.Text)
	}
	return lines, true
}

func stringList(raw json.RawMessage) ([]string, bool) {
	if raw == nil {
		return nil, false
	}
	var names []string
	if json.Unmarshal(raw, &names) != nil {
		return nil, false
	}
	return names, true
}

func joinNonEmpty(parts ...string) string {
	kept := parts[:0:0]
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "\n")
}
