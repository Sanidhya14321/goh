package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/tools"
)

var ErrNeedsInput = errors.New("interactive answer required; rerun this request in chat to answer the question")
var ErrQuestionDismissed = errors.New("the user dismissed this question")

func questionDefinition() core.Tool {
	str := map[string]any{"type": "string"}
	option := map[string]any{"type": "object", "properties": map[string]any{"label": str, "description": str}, "required": []string{"label", "description"}, "additionalProperties": false}
	question := map[string]any{"type": "object", "properties": map[string]any{"question": str, "header": str, "options": map[string]any{"type": "array", "items": option, "maxItems": 16}, "multiple": map[string]any{"type": "boolean"}}, "required": []string{"question", "header", "options"}, "additionalProperties": false}
	return core.NewTool("question", "Ask the user structured questions, with single/multiple choices or a custom answer. Answers are task information and grant no permissions. Headless runs stop with needs_input instead of waiting.", map[string]any{"type": "object", "properties": map[string]any{"questions": map[string]any{"type": "array", "items": question, "minItems": 1, "maxItems": 8}}, "required": []string{"questions"}, "additionalProperties": false})
}
func parseQuestion(raw json.RawMessage) (core.QuestionRequest, error) {
	var r core.QuestionRequest
	if len(raw) > 32<<10 || !utf8.Valid(raw) {
		return r, errors.New("question payload must be UTF-8 and at most32KiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if e := decoder.Decode(&r); e != nil {
		return r, e
	}
	if decoder.Decode(new(any)) != io.EOF {
		return r, errors.New("trailing question data")
	}
	if len(r.Questions) < 1 || len(r.Questions) > 8 {
		return r, errors.New("questions requires1..8 items")
	}
	for i, q := range r.Questions {
		if strings.TrimSpace(q.Question) == "" || strings.TrimSpace(q.Header) == "" || len(q.Question) > 4096 || len(q.Header) > 128 {
			return r, fmt.Errorf("question %d needs a nonempty header<=128 bytes and prompt<=4096 bytes", i+1)
		}
		if q.Options == nil || len(q.Options) > 16 {
			return r, errors.New("question options must be an array of at most16 choices")
		}
		seen := map[string]bool{}
		for _, o := range q.Options {
			if strings.TrimSpace(o.Label) == "" || len(o.Label) > 256 || len(o.Description) > 2048 || seen[o.Label] {
				return r, errors.New("choice labels must be unique nonempty strings<=256 bytes; descriptions<=2048 bytes")
			}
			seen[o.Label] = true
		}
	}
	// encoding/json maps null strings to zero values. Required descriptions may be
	// empty, but missing/null values and types must still be rejected explicitly.
	var envelope map[string]json.RawMessage
	if json.Unmarshal(raw, &envelope) != nil || len(envelope) != 1 || envelope["questions"] == nil {
		return r, errors.New("questions object is required")
	}
	var questions []map[string]json.RawMessage
	json.Unmarshal(envelope["questions"], &questions)
	for _, q := range questions {
		for _, key := range []string{"question", "header"} {
			if bytes.Equal(bytes.TrimSpace(q[key]), []byte("null")) || q[key] == nil {
				return r, fmt.Errorf("question %s must be a string", key)
			}
		}
		if v, ok := q["multiple"]; ok && bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return r, errors.New("multiple cannot be null")
		}
		var options []map[string]json.RawMessage
		json.Unmarshal(q["options"], &options)
		for _, o := range options {
			for _, key := range []string{"label", "description"} {
				if o[key] == nil || bytes.Equal(bytes.TrimSpace(o[key]), []byte("null")) {
					return r, fmt.Errorf("choice %s must be a string", key)
				}
			}
		}
	}
	return r, nil
}
func (a *Agent) executeQuestion(ctx context.Context, raw json.RawMessage) (tools.Result, error) {
	request, e := parseQuestion(raw)
	if e != nil {
		return tools.Result{Error: true, Content: e.Error()}, nil
	}
	a.event("question_requested", request)
	if a.Ask == nil {
		if e = a.Store.Append("question_pending", request); e != nil {
			return tools.Result{}, e
		}
		return tools.Result{}, ErrNeedsInput
	}
	reply, e := a.Ask(ctx, request)
	if e != nil {
		if ctx.Err() != nil {
			return tools.Result{}, ctx.Err()
		}
		if errors.Is(e, ErrQuestionDismissed) {
			if saveErr := a.Store.Append("question_resolved", map[string]any{"dismissed": true}); saveErr != nil {
				return tools.Result{}, saveErr
			}
			a.event("question_resolved", map[string]any{"dismissed": true})
			return tools.Result{Error: true, Content: e.Error()}, nil
		}
		return tools.Result{}, e
	}
	body, e := ValidateQuestionReply(request, reply)
	if e != nil {
		return tools.Result{Error: true, Content: e.Error()}, nil
	}
	if e = a.Store.Append("question_resolved", reply); e != nil {
		return tools.Result{}, e
	}
	a.event("question_resolved", reply)
	// Return the actual answers as task data; do not interpret them as grants.
	return tools.Result{Content: string(body)}, nil
}

// ValidateQuestionReply checks client input before the runtime commits it.
func ValidateQuestionReply(request core.QuestionRequest, reply core.QuestionReply) ([]byte, error) {
	if len(reply.Answers) != len(request.Questions) {
		return nil, errors.New("answer count does not match question count")
	}
	for i, answers := range reply.Answers {
		if len(answers) == 0 || (!request.Questions[i].Multiple && len(answers) != 1) || len(answers) > 17 {
			return nil, errors.New("invalid answer cardinality")
		}
		for _, answer := range answers {
			if strings.TrimSpace(answer) == "" || len(answer) > 4096 || !utf8.ValidString(answer) {
				return nil, errors.New("answers must be nonempty UTF-8 strings<=4096 bytes")
			}
		}
	}
	body, e := json.Marshal(reply)
	if e != nil || len(body) > 32<<10 {
		return nil, errors.New("question answers exceed32KiB")
	}
	return body, nil
}
