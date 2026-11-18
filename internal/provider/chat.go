// Package provider adapts model APIs to one streaming interface: OpenAI Chat
// Completions, OpenAI Responses (including the ChatGPT sign-in) and Anthropic
// Messages. Each adapter implements only the subset the runtime needs.
package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/core"
)

const maxResponse = 16 << 20

var ErrConnection = errors.New("provider connection failed")
var ErrInterrupted = errors.New("interrupted stream: missing finish or DONE")

// ErrRejectedOutput marks a reply the provider refused while generating it,
// such as a malformed or unknown tool call that Groq validates server-side
// (tool_use_failed). Nothing was executed and sampling differs on each
// attempt, so the request can simply be sent again.
var ErrRejectedOutput = errors.New("the provider rejected the model's output")

var errorIdentifier = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

// streamError reports an error object sent inside a stream. Only its short
// code and type identifiers are shown: messages can reflect request data.
func streamError(raw json.RawMessage) error {
	var body struct {
		Code any    `json:"code"`
		Type string `json:"type"`
	}
	json.Unmarshal(raw, &body)
	code, _ := body.Code.(string)
	var ids []string
	for _, id := range []string{code, body.Type} {
		if errorIdentifier.MatchString(id) {
			ids = append(ids, id)
		}
	}
	if code == "tool_use_failed" {
		return fmt.Errorf("%w: an invalid tool call (tool_use_failed)", ErrRejectedOutput)
	}
	if len(ids) == 0 {
		return errors.New("provider stream error")
	}
	return markBusy(fmt.Errorf("provider stream error (%s)", strings.Join(ids, ", ")), code, body.Type)
}

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

	renewed, droppedCacheKey := false, false
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
				if e == nil && (responses && len(reply.Message.ResponsesReasoning) != 0 || anthropic && len(reply.Message.AnthropicThinking) != 0) {
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
			// The ChatGPT preview rejects some standard fields; drop the
			// optional cache key once if it is named as unsupported.
			if responses && resp.StatusCode == 400 && attempt < 2 && c.Config.CacheKey != "" && !droppedCacheKey &&
				rejectsParam(resp.Body, "prompt_cache_key") {
				resp.Body.Close()
				droppedCacheKey = true
				cfg := c.Config
				cfg.CacheKey = ""
				if b, e = encodeResponses(cfg, messages, tools); e != nil {
					return core.Reply{}, e
				}
				continue
			}
			var limit, requested int
			var wait time.Duration
			if resp.StatusCode == 413 || resp.StatusCode == 429 {
				limit, requested, wait = requestCap(resp.Body)
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
			failure := classifyHTTP(resp.StatusCode)
			failure.Limit, failure.Requested = limit, requested
			if failure.TooLarge() {
				// A rate limit smaller than the request itself (Groq states
				// "Limit 8000, Requested 19000") never clears by waiting.
				failure.Kind, failure.Retryable, retry = "request_too_large", false, false
			}
			delay = retryDelay(resp.StatusCode, resp.Header.Get("Retry-After"), attempt, delay)
			if resp.StatusCode == 429 && retry {
				if seconds, e := strconv.ParseFloat(strings.TrimSpace(resp.Header.Get("Retry-After")), 64); e == nil && seconds > 0 && seconds < 1e9 {
					wait = max(wait, time.Duration(seconds*float64(time.Second)))
				}
				if wait > maxRateLimitWait {
					// A quota that resets in minutes or hours: retrying
					// within this request only spends the time.
					failure.Wait, retry = wait, false
				}
			}
			err = failure
		}
		if ctx.Err() != nil {
			return core.Reply{}, ctx.Err()
		}
		if !retry || attempt == 2 {
			return core.Reply{}, err
		}
		if notify := core.RetryNotifier(ctx); notify != nil {
			notify(core.RetryNotice{Attempt: attempt + 2, Max: 3, DelayMS: delay.Milliseconds(), Reason: err.Error()})
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
	Usage   *chatUsage      `json:"usage"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content   *string `json:"content"`
			Refusal   *string `json:"refusal"`
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
			return streamError(ch.Error)
		}
		if ch.Usage != nil {
			u := ch.Usage.Usage
			if d := ch.Usage.Details; d != nil && d.Cached >= 0 && d.Cached <= u.Prompt {
				u.CacheRead = d.Cached
			}
			reply.Usage = &u
		}
		if len(ch.Choices) > 1 {
			return errors.New("multiple choices unsupported")
		}
		for _, v := range ch.Choices {
			if v.Index != 0 {
				return errors.New("unsupported choice index")
			}
			if finished && (v.Delta.Content != nil || v.Delta.Refusal != nil || len(v.Delta.ToolCalls) > 0) {
				return errors.New("content after finish")
			}
			// A refusal is the model's reply, as in the non-streaming parser.
			for _, text := range []*string{v.Delta.Content, v.Delta.Refusal} {
				if text != nil {
					content.WriteString(*text)
					if delta != nil {
						delta(*text)
					}
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
				// Some compatible servers repeat the whole id and name in every
				// chunk instead of sending them once; a repeat is not a new part.
				call.ID = joinPart(call.ID, part.ID)
				call.Function.Name = joinPart(call.Function.Name, part.Function.Name)
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
		if errors.Is(e, bufio.ErrTooLong) {
			return reply, e
		}
		// A dropped or timed-out connection: an interrupted stream, which
		// retry_interrupted_streams may re-send, not a raw socket error.
		return reply, fmt.Errorf("provider stream read failed (connection dropped or timed out): %w", ErrInterrupted)
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
		m.AnthropicThinking, m.ResponsesReasoning, m.ResponsesRoute, m.Automatic = nil, nil, "", false
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

// chatUsage is Chat Completions usage, including cached prompt tokens.
type chatUsage struct {
	core.Usage
	Details *struct {
		Cached int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
}

// joinPart adds a streamed id or name fragment, ignoring a chunk that repeats
// the value already received.
func joinPart(have, part string) string {
	if part == "" || part == have {
		return have
	}
	return have + part
}

// retryDelay is how long to wait before resending after an HTTP failure.
// Rate limits (429) usually reset within a minute: waiting a few hundred
// milliseconds just spends the remaining attempts and fails a long run
// midway, so a 429 waits as long as Retry-After asks (up to 30s), or 2s and
// then 4s without one. Other statuses keep short waits (up to 5s).
// maxRateLimitWait is the longest reset a 429 may state and still be
// retried: beyond it the limit is a quota, not a burst.
const maxRateLimitWait = 2 * time.Minute

func retryDelay(status int, retryAfter string, attempt int, fallback time.Duration) time.Duration {
	limit := 5 * time.Second
	if status == 429 {
		limit = 30 * time.Second
		fallback = time.Duration(attempt+1) * 2 * time.Second
	}
	if seconds, err := strconv.ParseFloat(strings.TrimSpace(retryAfter), 64); err == nil && seconds >= 0 && seconds < 1e6 {
		fallback = time.Duration(seconds * float64(time.Second))
	}
	return min(fallback, limit)
}
