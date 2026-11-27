package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// AskFunc puts questions to the user and returns the answers.
type AskFunc func(context.Context, core.QuestionRequest) (core.QuestionReply, error)

// SetAsk lets servers ask the user for input (MCP elicitation, form mode)
// through ask. Without it, or in a run nobody answers, servers are not
// offered elicitation. Set it before Connect.
func (m *Manager) SetAsk(ask AskFunc) {
	m.mu.Lock()
	m.ask = ask
	m.mu.Unlock()
}

// ForAgent returns m, letting its servers ask the user when someone can
// answer (canAsk: an interactive chat or a client that relays questions).
func ForAgent(m *Manager, canAsk bool, ask AskFunc) *Manager {
	if canAsk {
		m.SetAsk(ask)
	}
	return m
}

const maxElicitFields = 8

// elicitField is one property of a form request.
type elicitField struct {
	key, kind, title, description string
	required                      bool
	labels                        []string // choices shown to the user
	values                        []any    // the value sent for each choice
}

func (m *Manager) elicitHandler(server string, ask AskFunc) func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
	return func(ctx context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
		if req.Params == nil || (req.Params.Mode != "" && req.Params.Mode != "form") {
			return &mcp.ElicitResult{Action: "decline"}, nil // URL mode is not offered
		}
		fields, err := elicitFields(req.Params.RequestedSchema)
		if err != nil {
			return &mcp.ElicitResult{Action: "decline"}, nil
		}
		request := elicitQuestions(server, req.Params.Message, fields)
		reply, err := ask(ctx, request)
		if err != nil {
			if ctx.Err() != nil {
				return &mcp.ElicitResult{Action: "cancel"}, nil
			}
			return &mcp.ElicitResult{Action: "decline"}, nil
		}
		content, err := elicitContent(fields, reply)
		if err != nil {
			return &mcp.ElicitResult{Action: "decline"}, nil
		}
		return &mcp.ElicitResult{Action: "accept", Content: content}, nil
	}
}

// elicitFields reads a form schema: a flat object of string, number,
// integer, boolean, enum and multi-select enum properties.
func elicitFields(schema any) ([]elicitField, error) {
	raw, err := json.Marshal(schema)
	if err != nil || len(raw) > 64<<10 {
		return nil, errors.New("schema unreadable")
	}
	var s struct {
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if json.Unmarshal(raw, &s) != nil || s.Type != "object" {
		return nil, errors.New("schema is not an object")
	}
	if len(s.Properties) == 0 || len(s.Properties) > maxElicitFields {
		return nil, errors.New("form has no fields or too many")
	}
	required := map[string]bool{}
	for _, r := range s.Required {
		required[r] = true
	}
	// The SDK hands the schema over as a map, so the server's field order
	// is gone: required fields come first, each group by name.
	order := make([]string, 0, len(s.Properties))
	for k := range s.Properties {
		order = append(order, k)
	}
	sort.Slice(order, func(i, j int) bool {
		if required[order[i]] != required[order[j]] {
			return required[order[i]]
		}
		return order[i] < order[j]
	})
	var fields []elicitField
	for _, key := range order {
		var p struct {
			Type        string   `json:"type"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Enum        []any    `json:"enum"`
			EnumNames   []string `json:"enumNames"`
			OneOf       []struct {
				Const any    `json:"const"`
				Title string `json:"title"`
			} `json:"oneOf"`
			Items *struct {
				Enum  []any `json:"enum"`
				AnyOf []struct {
					Const any    `json:"const"`
					Title string `json:"title"`
				} `json:"anyOf"`
			} `json:"items"`
		}
		if json.Unmarshal(s.Properties[key], &p) != nil {
			return nil, errors.New("property unreadable")
		}
		f := elicitField{key: key, kind: p.Type, title: p.Title, description: p.Description, required: required[key]}
		choice := func(v any, title string) {
			label := title
			if label == "" {
				label = fmt.Sprint(v)
			}
			f.labels, f.values = append(f.labels, label), append(f.values, v)
		}
		switch p.Type {
		case "boolean":
			choice(true, "Yes")
			choice(false, "No")
		case "string", "number", "integer":
			for i, v := range p.Enum {
				name := ""
				if i < len(p.EnumNames) {
					name = p.EnumNames[i]
				}
				choice(v, name)
			}
			for _, o := range p.OneOf {
				choice(o.Const, o.Title)
			}
		case "array":
			if p.Items == nil {
				return nil, errors.New("array without items")
			}
			for _, v := range p.Items.Enum {
				choice(v, "")
			}
			for _, o := range p.Items.AnyOf {
				choice(o.Const, o.Title)
			}
			if len(f.labels) == 0 {
				return nil, errors.New("array without choices")
			}
		default:
			return nil, errors.New("unsupported property type")
		}
		if len(f.labels) > 16 || !uniqueLabels(f.labels) {
			return nil, errors.New("too many or repeated choices")
		}
		fields = append(fields, f)
	}
	return fields, nil
}

func uniqueLabels(labels []string) bool {
	seen := map[string]bool{}
	for _, l := range labels {
		if strings.TrimSpace(l) == "" || len(l) > 256 || seen[l] {
			return false
		}
		seen[l] = true
	}
	return true
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n] + "…"
}

// elicitQuestions shows the server's message once, with each field as a
// question; the header names the server the answers go to.
func elicitQuestions(server, message string, fields []elicitField) core.QuestionRequest {
	header := clip("MCP "+server, 128)
	var r core.QuestionRequest
	for i, f := range fields {
		name := f.title
		if name == "" {
			name = f.key
		}
		text := name
		if f.description != "" {
			text += ": " + f.description
		}
		switch {
		case f.kind == "number" || f.kind == "integer":
			if len(f.labels) == 0 {
				text += " (a " + f.kind + ")"
			}
		case f.kind == "array":
			text += " (choose any)"
		}
		if !f.required {
			text += " (optional; leave empty to skip)"
		}
		if i == 0 {
			text = "The MCP server " + server + " asks for input; your answers are sent to it.\n\n" + strings.TrimSpace(message) + "\n\n" + text
		}
		q := core.Question{Question: clip(text, 4096), Header: header, Options: []core.QuestionOption{}, Multiple: f.kind == "array"}
		for _, l := range f.labels {
			q.Options = append(q.Options, core.QuestionOption{Label: l, Description: ""})
		}
		r.Questions = append(r.Questions, q)
	}
	return r
}

// elicitContent converts the answers to the values the schema asks for.
func elicitContent(fields []elicitField, reply core.QuestionReply) (map[string]any, error) {
	if len(reply.Answers) != len(fields) {
		return nil, errors.New("answer count")
	}
	content := map[string]any{}
	for i, f := range fields {
		var answers []string
		for _, a := range reply.Answers[i] {
			if a = strings.TrimSpace(a); a != "" {
				answers = append(answers, a)
			}
		}
		if len(answers) == 0 {
			if f.required {
				return nil, fmt.Errorf("%s is required", f.key)
			}
			continue
		}
		pick := func(answer string) (any, bool) {
			for j, l := range f.labels {
				if l == answer {
					return f.values[j], true
				}
			}
			return nil, false
		}
		if f.kind == "array" {
			var values []any
			for _, a := range answers {
				v, ok := pick(a)
				if !ok {
					return nil, fmt.Errorf("%s: %q is not a choice", f.key, a)
				}
				values = append(values, v)
			}
			content[f.key] = values
			continue
		}
		if len(answers) != 1 {
			return nil, fmt.Errorf("%s takes one answer", f.key)
		}
		answer := answers[0]
		if v, ok := pick(answer); ok {
			content[f.key] = v
			continue
		}
		if len(f.labels) > 0 && f.kind != "boolean" {
			return nil, fmt.Errorf("%s: %q is not a choice", f.key, answer)
		}
		switch f.kind {
		case "boolean":
			b, err := strconv.ParseBool(strings.ToLower(answer))
			if err != nil {
				return nil, fmt.Errorf("%s takes yes or no", f.key)
			}
			content[f.key] = b
		case "integer":
			n, err := strconv.ParseInt(answer, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%s takes a whole number", f.key)
			}
			content[f.key] = n
		case "number":
			n, err := strconv.ParseFloat(answer, 64)
			if err != nil {
				return nil, fmt.Errorf("%s takes a number", f.key)
			}
			content[f.key] = n
		default:
			content[f.key] = answer
		}
	}
	return content, nil
}
