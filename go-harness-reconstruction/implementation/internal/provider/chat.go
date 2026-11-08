// Package provider implements the deliberately bounded Chat Completions subset.
package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/independent-harness/internal/core"
)

const maxResponse = 16 << 20

var ErrConnection = errors.New("provider connection failed")
var ErrInterrupted = errors.New("interrupted stream: missing finish or DONE")

type FinishError struct{ Reason string }

func (e *FinishError) Error() string { return "provider finish reason: " + e.Reason }

type Client struct {
	Config core.Config
	HTTP   *http.Client
}

func New(c core.Config) *Client {
	return &Client{Config: c, HTTP: &http.Client{Timeout: 10 * time.Minute}}
}
func (c *Client) Complete(ctx context.Context, messages []core.Message, tools []core.Tool, delta func(string)) (core.Reply, error) {
	if err := c.Config.ValidateProtocol(); err != nil {
		return core.Reply{}, err
	}
	endpoint, err := c.Config.ProviderEndpoint()
	if err != nil {
		return core.Reply{}, err
	}
	protocol := c.Config.EffectiveProtocol()
	chatGPT := protocol == core.ChatGPTProtocol
	responses := chatGPT || protocol == "openai_responses"
	key, err := c.Config.ProviderKey()
	if chatGPT {
		key, err = core.ChatGPTAccessToken(ctx)
	}
	if err != nil {
		return core.Reply{}, err
	}
	var b []byte
	var e error
	parse := Parse
	anthropic := protocol == "anthropic_messages"
	var body map[string]any
	if anthropic {
		b, e = EncodeAnthropic(c.Config, messages, tools)
		sink := core.ReasoningSink(ctx)
		parse = func(r io.Reader, d func(string)) (core.Reply, error) { return parseAnthropic(r, d, sink) }
	} else if responses {
		b, e = encodeResponses(c.Config, messages, tools)
		sink := core.ReasoningSink(ctx)
		parse = func(r io.Reader, d func(string)) (core.Reply, error) { return parseResponsesReasoning(r, d, sink) }
	} else {
		var clean []core.Message
		for i := range messages {
			if len(messages[i].ResponsesReasoning) != 0 || messages[i].ResponsesRoute != "" || len(messages[i].AnthropicThinking) != 0 {
				if clean == nil {
					clean = append([]core.Message(nil), messages...)
				}
				clean[i].ResponsesReasoning = nil
				clean[i].ResponsesRoute = ""
				clean[i].AnthropicThinking = nil
			}
		}
		if clean != nil {
			messages = clean
		}
		wire := chatWireMessages(messages)
		body = map[string]any{"model": c.Config.Model, "messages": wire, "stream": true, "stream_options": map[string]any{"include_usage": true}}
		if len(tools) > 0 {
			body["tools"] = tools
		}
		if c.Config.ReasoningEffort != "" {
			body["reasoning_effort"] = c.Config.ReasoningEffort
		}
		if parameter := c.Config.OutputParameter; parameter != "" {
			u, _ := url.Parse(endpoint)
			if parameter == "max_tokens" && u.Scheme == "https" && strings.EqualFold(u.Hostname(), "api.openai.com") {
				parameter = "max_completion_tokens"
			}
			body[parameter] = c.Config.OutputBudget
		}
		b, e = json.Marshal(body)
	}
	if e != nil {
		return core.Reply{}, e
	}
	if len(b) > maxResponse {
		return core.Reply{}, errors.New("provider request exceeds 16MiB")
	}
	if c.HTTP == nil {
		return core.Reply{}, errors.New("provider HTTP client unavailable")
	}
	// Redirects never forward prompts or credentials to an unselected endpoint.
	client := *c.HTTP
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	renewed := false
	for attempt := 0; attempt < 3; attempt++ {
		req, e := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(b))
		if e != nil {
			return core.Reply{}, errors.New("invalid provider request configuration")
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		if anthropic {
			req.Header.Set("anthropic-version", "2023-06-01")
			if key != "" {
				req.Header.Set("x-api-key", key)
			}
		} else {
			if !responses {
				req.Header.Set("Accept", "text/event-stream, application/json")
			}
			if key != "" {
				req.Header.Set("Authorization", "Bearer "+key)
			}
		}
		resp, err := client.Do(req)
		retry := err != nil
		if err != nil {
			err = ErrConnection
		}
		delay := time.Duration(attempt+1) * 100 * time.Millisecond
		if err == nil {
			retry = resp.StatusCode == 429 || resp.StatusCode == 408 || resp.StatusCode >= 500
			if resp.StatusCode == 200 {
				mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
				read := parse
				if !anthropic && !responses && mediaType == "application/json" {
					read = parseJSONCompletion
				}
				reply, e := read(resp.Body, delta)
				resp.Body.Close()
				if e == nil && responses && len(reply.Message.ResponsesReasoning) != 0 {
					reply.Message.ResponsesRoute = responsesRoute(c.Config)
				}
				return reply, e
			}
			// Provider bodies can reflect credentials. Inspect only a bounded
			// structured capability error and never expose the body.
			if !anthropic && !responses && resp.StatusCode == 400 && attempt < 2 && body["stream_options"] != nil &&
				rejectsStreamOptions(resp.Body) {
				resp.Body.Close()
				delete(body, "stream_options")
				b, e = json.Marshal(body)
				if e != nil {
					return core.Reply{}, e
				}
				continue
			}
			resp.Body.Close()
			// A ChatGPT token can be revoked before its recorded expiry. A 401
			// means nothing was generated, so renew once and resend.
			if chatGPT && resp.StatusCode == 401 && !renewed {
				renewed = true
				if next, renewErr := core.RenewChatGPTAccessToken(ctx, key); renewErr == nil && next != "" {
					key = next
					attempt--
					continue
				}
			}
			err = classifyHTTP(resp.StatusCode)
			if n, e := strconv.Atoi(resp.Header.Get("Retry-After")); e == nil && n >= 0 {
				if n > 5 {
					n = 5
				}
				delay = time.Duration(n) * time.Second
			}
		}
		if ctx.Err() != nil {
			return core.Reply{}, ctx.Err()
		}
		if !retry || attempt == 2 {
			return core.Reply{}, err
		}
		select {
		case <-ctx.Done():
			return core.Reply{}, ctx.Err()
		case <-time.After(delay):
		}
	}
	panic("unreachable")
}

type chunk struct {
	Error   json.RawMessage `json:"error"`
	Usage   *core.Usage     `json:"usage"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content   *string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		Finish *string `json:"finish_reason"`
	} `json:"choices"`
}

func Parse(r io.Reader, delta func(string)) (core.Reply, error) {
	reply := core.Reply{Message: core.Message{Role: "assistant"}}
	calls := map[int]*core.Call{}
	var content strings.Builder
	total := 0
	finished := false
	done := false
	s := bufio.NewScanner(io.LimitReader(r, maxResponse+1))
	s.Buffer(make([]byte, 4096), 2<<20)
	var data []string
	consume := func() error {
		if len(data) == 0 {
			return nil
		}
		raw := strings.Join(data, "\n")
		data = nil
		if raw == "[DONE]" && !done {
			done = true
			return nil
		}
		if done {
			return errors.New("data after DONE")
		}
		if !utf8.ValidString(raw) {
			return errors.New("invalid UTF-8 stream")
		}
		var ch chunk
		if e := json.Unmarshal([]byte(raw), &ch); e != nil {
			return errors.New("invalid SSE JSON")
		}
		if len(ch.Error) > 0 && string(ch.Error) != "null" {
			return errors.New("provider stream error")
		}
		if ch.Usage != nil {
			reply.Usage = ch.Usage
		}
		if len(ch.Choices) > 1 {
			return errors.New("multiple choices unsupported")
		}
		for _, v := range ch.Choices {
			if v.Index != 0 {
				return errors.New("unsupported choice index")
			}
			if finished && (v.Delta.Content != nil || len(v.Delta.ToolCalls) > 0) {
				return errors.New("content after finish")
			}
			if v.Delta.Content != nil {
				content.WriteString(*v.Delta.Content)
				if delta != nil {
					delta(*v.Delta.Content)
				}
			}
			for _, part := range v.Delta.ToolCalls {
				if part.Index < 0 || part.Index >= 128 {
					return errors.New("tool index outside limit")
				}
				call := calls[part.Index]
				if call == nil {
					call = &core.Call{Index: part.Index, Type: "function"}
					calls[part.Index] = call
				}
				if part.Type != "" && part.Type != "function" {
					return errors.New("unsupported tool type")
				}
				call.ID += part.ID
				call.Function.Name += part.Function.Name
				call.Function.Arguments += part.Function.Arguments
				if len(call.Function.Arguments) > 1<<20 || len(call.ID) > 512 || len(call.Function.Name) > 256 {
					return errors.New("tool payload limit")
				}
			}
			if v.Finish != nil {
				if finished {
					return errors.New("duplicate finish")
				}
				finished = true
				reply.Finish = *v.Finish
			}
		}
		return nil
	}
	for s.Scan() {
		line := s.Text()
		total += len(line) + 1
		if total > maxResponse {
			return reply, errors.New("stream exceeds byte limit")
		}
		if line == "" {
			if e := consume(); e != nil {
				return reply, e
			}
		} else if strings.HasPrefix(line, "data:") {
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if e := s.Err(); e != nil {
		return reply, e
	}
	if len(data) > 0 {
		if e := consume(); e != nil {
			return reply, e
		}
	}
	if !finished || !done {
		return reply, ErrInterrupted
	}
	if reply.Finish != "stop" && reply.Finish != "tool_calls" {
		return reply, &FinishError{Reason: reply.Finish}
	}
	reply.Message.Content = content.String()
	indices := make([]int, 0, len(calls))
	for i := range calls {
		indices = append(indices, i)
	}
	sort.Ints(indices)
	ids := map[string]bool{}
	for n, i := range indices {
		call := calls[i]
		if i != n || call.ID == "" || call.Function.Name == "" || ids[call.ID] {
			return reply, errors.New("invalid or duplicate tool identity")
		}
		ids[call.ID] = true
		reply.Message.ToolCalls = append(reply.Message.ToolCalls, *call)
	}
	if (len(calls) > 0) != (reply.Finish == "tool_calls") {
		return reply, errors.New("finish reason/tool-call mismatch")
	}
	return reply, nil
}

// chatWireMessages sends messages unchanged except user messages with images,
// whose content becomes text plus image_url parts.
func chatWireMessages(messages []core.Message) []any {
	out := make([]any, len(messages))
	for i, m := range messages {
		// Provider-specific state never reaches a Chat Completions request.
		m.AnthropicThinking, m.ResponsesReasoning, m.ResponsesRoute = nil, nil, ""
		if len(m.Images) == 0 {
			out[i] = m
			continue
		}
		parts := []any{map[string]any{"type": "text", "text": m.Content}}
		for _, img := range m.Images {
			parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]any{"url": img.DataURL()}})
		}
		out[i] = map[string]any{"role": m.Role, "content": parts}
	}
	return out
}
