package provider

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
)

const toolNamespace = "goharness"

func responsesRoute(c core.Config) string {
	endpoint, _ := c.ProviderEndpoint()
	identity := sha256.Sum256([]byte(c.EffectiveProtocol() + "\n" + endpoint + "\n" + c.Model))
	return hex.EncodeToString(identity[:])
}

func encodeResponses(c core.Config, messages []core.Message, tools []core.Tool) ([]byte, error) {
	plan := c.EffectiveProtocol() == core.ChatGPTProtocol
	route := responsesRoute(c)
	input := make([]any, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "system", "user", "assistant":
			role := m.Role
			if role == "system" {
				role = "developer"
			}
			for _, raw := range m.ResponsesReasoning {
				if m.Role == "assistant" && m.ResponsesRoute == route && validResponsesReasoning(raw) {
					input = append(input, raw)
				}
			}
			if len(m.Images) > 0 && m.Role == "user" {
				parts := []any{map[string]any{"type": "input_text", "text": m.Content}}
				for _, img := range m.Images {
					parts = append(parts, map[string]any{"type": "input_image", "image_url": img.DataURL()})
				}
				input = append(input, map[string]any{"role": role, "content": parts})
			} else if m.Content != "" {
				input = append(input, map[string]any{"role": role, "content": m.Content})
			}
			for _, call := range m.ToolCalls {
				item := map[string]any{"type": "function_call", "call_id": call.ID,
					"name": call.Function.Name, "arguments": call.Function.Arguments}
				if plan {
					item["namespace"] = toolNamespace
				}
				input = append(input, item)
			}
		case "tool":
			input = append(input, map[string]any{"type": "function_call_output", "call_id": m.ToolCallID, "output": m.Content})
		default:
			return nil, errors.New("unsupported message role for Responses")
		}
	}
	body := map[string]any{"model": c.Model, "input": input, "stream": true, "store": false,
		"include": []string{"reasoning.encrypted_content"}}
	// ChatGPT plan preview rejects this request field. The runtime still
	// reserves output space and enforces its context and turn budgets.
	if !plan && c.OutputBudget > 0 {
		body["max_output_tokens"] = c.OutputBudget
	}
	if c.CacheKey != "" {
		// One key per conversation keeps its requests on the same prompt
		// cache (the history is append-only), as Codex does.
		body["prompt_cache_key"] = c.CacheKey
	}
	if c.ReasoningEffort != "" {
		// A summary is requested only with an explicit effort, where the model
		// is known to reason; it is shown in the UI and never replayed.
		body["reasoning"] = map[string]any{"effort": c.ReasoningEffort, "summary": "auto"}
	}
	if len(tools) > 0 {
		functions := make([]map[string]any, 0, len(tools))
		for _, tool := range tools {
			functions = append(functions, map[string]any{"type": "function", "name": tool.Function.Name,
				"description": tool.Function.Description, "parameters": tool.Function.Parameters, "strict": false})
		}
		if plan {
			body["tools"] = []any{map[string]any{"type": "namespace", "name": toolNamespace,
				"description": "Local GoHarness project tools, subject to local approvals.", "tools": functions}}
		} else {
			body["tools"] = functions
		}
	}
	return json.Marshal(body)
}

func validResponsesReasoning(raw json.RawMessage) bool {
	var item struct {
		Type      string `json:"type"`
		Encrypted string `json:"encrypted_content"`
	}
	return len(raw) <= 1<<20 && json.Unmarshal(raw, &item) == nil && item.Type == "reasoning" && item.Encrypted != ""
}

type responsesResult struct {
	Status string            `json:"status"`
	Output []json.RawMessage `json:"output"`
	Usage  *struct {
		Input   int `json:"input_tokens"`
		Output  int `json:"output_tokens"`
		Total   int `json:"total_tokens"`
		Details *struct {
			Cached int `json:"cached_tokens"`
		} `json:"input_tokens_details"`
	} `json:"usage"`
	Incomplete *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
}

// parseResponsesReasoning parses a Responses SSE stream, sending text to delta
// and reasoning summary text to reasoning (either may be nil).
func parseResponsesReasoning(r io.Reader, delta func(string), reasoning func(string)) (core.Reply, error) {
	scanner := bufio.NewScanner(io.LimitReader(r, maxResponse+1))
	scanner.Buffer(make([]byte, 4096), maxResponse)
	total := 0
	var data []string
	emitted := false
	consume := func() (core.Reply, bool, error) {
		if len(data) == 0 {
			return core.Reply{}, false, nil
		}
		raw := strings.Join(data, "\n")
		data = nil
		if raw == "[DONE]" {
			return core.Reply{}, false, nil
		}
		var event struct {
			Type     string          `json:"type"`
			Delta    json.RawMessage `json:"delta"`
			Response responsesResult `json:"response"`
		}
		if !utf8.ValidString(raw) || json.Unmarshal([]byte(raw), &event) != nil || event.Type == "" {
			return core.Reply{}, false, errors.New("invalid Responses stream event")
		}
		switch event.Type {
		case "response.output_text.delta", "response.refusal.delta":
			var text string
			if json.Unmarshal(event.Delta, &text) != nil {
				return core.Reply{}, false, errors.New("invalid Responses text delta")
			}
			if delta != nil && text != "" {
				delta(text)
				emitted = true
			}
		case "response.reasoning_summary_text.delta":
			var text string
			if json.Unmarshal(event.Delta, &text) == nil && reasoning != nil && text != "" {
				reasoning(text)
			}
		case "response.reasoning_summary_part.added":
			if reasoning != nil {
				reasoning("\n\n") // separate summary paragraphs
			}
		case "response.completed":
			reply, err := decodeResponses(event.Response)
			if err == nil && !emitted && delta != nil && reply.Message.Content != "" {
				delta(reply.Message.Content)
			}
			return reply, true, err
		case "response.incomplete":
			reason := "incomplete"
			if event.Response.Incomplete != nil {
				switch event.Response.Incomplete.Reason {
				case "max_output_tokens":
					reason = "length"
				case "content_filter":
					reason = "content_filter"
				}
			}
			return core.Reply{}, true, &FinishError{Reason: reason}
		case "response.failed", "error":
			return core.Reply{}, true, responsesStreamError(raw)
		}
		return core.Reply{}, false, nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		total += len(line) + 1
		if total > maxResponse {
			return core.Reply{}, errors.New("Responses stream exceeds byte limit")
		}
		if line == "" {
			if reply, done, err := consume(); done || err != nil {
				return reply, err
			}
		} else if strings.HasPrefix(line, "data:") {
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if scanner.Err() == nil {
		if reply, done, err := consume(); done || err != nil {
			return reply, err
		}
	}
	return core.Reply{}, ErrInterrupted
}

func decodeResponses(result responsesResult) (core.Reply, error) {
	var reply core.Reply
	if result.Status != "completed" {
		return reply, errors.New("Responses completion did not finish successfully")
	}
	reply = core.Reply{Message: core.Message{Role: "assistant"}, Finish: "stop"}
	if result.Usage != nil {
		reply.Usage = &core.Usage{Prompt: result.Usage.Input, Completion: result.Usage.Output, Total: result.Usage.Total}
		if d := result.Usage.Details; d != nil && d.Cached >= 0 && d.Cached <= result.Usage.Input {
			reply.Usage.CacheRead = d.Cached
		}
	}
	var content strings.Builder
	ids := map[string]bool{}
	for _, raw := range result.Output {
		var item struct {
			Type      string                                 `json:"type"`
			Status    string                                 `json:"status"`
			Role      string                                 `json:"role"`
			ID        string                                 `json:"call_id"`
			Name      string                                 `json:"name"`
			Namespace string                                 `json:"namespace"`
			Arguments string                                 `json:"arguments"`
			Content   []struct{ Type, Text, Refusal string } `json:"content"`
		}
		if json.Unmarshal(raw, &item) != nil {
			return core.Reply{}, errors.New("invalid Responses output item")
		}
		switch item.Type {
		case "reasoning":
			if validResponsesReasoning(raw) {
				if len(reply.Message.ResponsesReasoning) == 16 {
					return core.Reply{}, errors.New("Responses reasoning exceeds item limit")
				}
				reply.Message.ResponsesReasoning = append(reply.Message.ResponsesReasoning, raw)
			}
		case "message":
			if item.Role != "assistant" {
				return core.Reply{}, errors.New("invalid Responses message role")
			}
			for _, part := range item.Content {
				switch part.Type {
				case "output_text":
					content.WriteString(part.Text)
				case "refusal":
					content.WriteString(part.Refusal)
				}
			}
		case "function_call":
			if (item.Status != "" && item.Status != "completed") || item.ID == "" || ids[item.ID] || len(item.ID) > 512 || item.Name == "" || len(item.Name) > 256 ||
				len(item.Arguments) > 1<<20 || len(ids) >= 128 || (item.Namespace != "" && item.Namespace != toolNamespace) {
				return core.Reply{}, errors.New("invalid Responses function call")
			}
			call := core.Call{Type: "function", ID: item.ID, Index: len(ids)}
			call.Function.Name, call.Function.Arguments = item.Name, item.Arguments
			ids[item.ID] = true
			reply.Message.ToolCalls = append(reply.Message.ToolCalls, call)
			reply.Finish = "tool_calls"
		default:
			return core.Reply{}, errors.New("unsupported Responses output item")
		}
	}
	reply.Message.Content = content.String()
	if reply.Message.Content == "" && len(reply.Message.ToolCalls) == 0 {
		return core.Reply{}, errors.New("OpenAI returned no text or tool calls")
	}
	return reply, nil
}

// responsesStreamError turns an in-stream error event into a readable error:
// the provider's error code (such as a usage or rate limit) and a bounded,
// single-line message, so a quota problem is distinguishable from an access
// problem. Nothing else from the event is shown.
func responsesStreamError(raw string) error {
	var e struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Error    *struct{ Code, Type, Message string }
		Response *struct {
			Error *struct{ Code, Message string }
		}
	}
	_ = json.Unmarshal([]byte(raw), &e)
	code, msg := e.Code, e.Message
	if e.Error != nil {
		code, msg = firstNonEmpty(e.Error.Code, e.Error.Type, code), firstNonEmpty(e.Error.Message, msg)
	}
	if e.Response != nil && e.Response.Error != nil {
		code, msg = firstNonEmpty(e.Response.Error.Code, code), firstNonEmpty(e.Response.Error.Message, msg)
	}
	// Both fields come from the provider and end up in events and the
	// journal: keep the code to an identifier, and strip control characters
	// and credential-shaped text from a bounded message.
	if !errorCode.MatchString(code) {
		code = ""
	}
	msg = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, msg)
	msg = session.RedactSecrets(strings.Join(strings.Fields(msg), " "))
	if r := []rune(msg); len(r) > 300 {
		msg = string(r[:300]) + "…"
	}
	text := "OpenAI Responses request failed"
	if code != "" {
		text += " (" + code + ")"
	}
	if msg != "" {
		text += ": " + msg
	} else {
		text += "; check model access and account usage"
	}
	return errors.New(text)
}

var errorCode = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,64}$`)

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
