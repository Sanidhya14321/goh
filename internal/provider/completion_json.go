package provider

import (
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/core"
)

func rejectsStreamOptions(r io.Reader) bool { return rejectsParam(r, "stream_options") }

// rejectsParam reports a structured 400 naming param as unsupported.
func rejectsParam(r io.Reader, param string) bool {
	const limit = 4096
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil || len(data) > limit {
		return false
	}
	var envelope struct {
		Error struct {
			Param string `json:"param"`
			Code  string `json:"code"`
		} `json:"error"`
	}
	return json.Unmarshal(data, &envelope) == nil && envelope.Error.Param == param &&
		(envelope.Error.Code == "unsupported_parameter" || envelope.Error.Code == "unknown_parameter")
}

// Some compatible endpoints return a complete JSON reply even when streaming
// was requested. Validate the entire reply before emitting text or tools.
func parseJSONCompletion(r io.Reader, delta func(string)) (core.Reply, error) {
	var reply core.Reply
	raw, err := io.ReadAll(io.LimitReader(r, maxResponse+1))
	if err != nil {
		return reply, errors.New("provider JSON response interrupted")
	}
	if len(raw) > maxResponse || !utf8.Valid(raw) {
		return reply, errors.New("provider JSON response exceeds limits or contains invalid UTF-8")
	}
	var envelope struct {
		Error   json.RawMessage `json:"error"`
		Usage   *chatUsage      `json:"usage"`
		Choices []struct {
			Index   int    `json:"index"`
			Finish  string `json:"finish_reason"`
			Message struct {
				core.Message
				Refusal string `json:"refusal"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(raw, &envelope) != nil || (len(envelope.Error) > 0 && string(envelope.Error) != "null") ||
		len(envelope.Choices) != 1 || envelope.Choices[0].Index != 0 {
		return reply, errors.New("invalid provider JSON completion")
	}
	choice := envelope.Choices[0]
	reply = core.Reply{Message: choice.Message.Message, Finish: choice.Finish}
	if envelope.Usage != nil {
		// Cached prompt tokens, as in the streaming parser, so cost
		// estimates price them at the cache rate.
		u := envelope.Usage.Usage
		if d := envelope.Usage.Details; d != nil && d.Cached >= 0 && d.Cached <= u.Prompt {
			u.CacheRead = d.Cached
		}
		reply.Usage = &u
	}
	if reply.Finish != "stop" && reply.Finish != "tool_calls" {
		reason := reply.Finish
		if reason != "length" && reason != "content_filter" {
			reason = "unsupported"
		}
		return core.Reply{}, &FinishError{Reason: reason}
	}
	calls := reply.Message.ToolCalls
	if reply.Message.Role != "assistant" || reply.Message.ToolCallID != "" || len(calls) > 128 ||
		(len(calls) > 0) != (reply.Finish == "tool_calls") {
		return core.Reply{}, errors.New("invalid provider JSON message")
	}
	seen := make(map[string]bool, len(calls))
	for i := range calls {
		call := &calls[i]
		if call.Type != "function" || call.ID == "" || len(call.ID) > 512 || seen[call.ID] ||
			call.Function.Name == "" || len(call.Function.Name) > 256 || len(call.Function.Arguments) > 1<<20 {
			return core.Reply{}, errors.New("invalid provider JSON tool call")
		}
		seen[call.ID] = true
		call.Index = i
	}
	if reply.Message.Content == "" {
		reply.Message.Content = choice.Message.Refusal
	}
	if delta != nil && reply.Message.Content != "" {
		delta(reply.Message.Content)
	}
	return reply, nil
}
