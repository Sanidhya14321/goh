package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Sanidhya14321/goh/internal/core"
)

// This original codec supports the explicit text/client-tool subset of the
// Messages protocol. It does not request or silently discard reasoning, media,
// citations, provider-executed tools, or continuation-only finish reasons.
const (
	anthropicMaxBlocks   = 128
	anthropicMaxText     = 1 << 20
	anthropicMaxEvent    = 2 << 20
	anthropicMaxMessages = 2048
)

var anthropicToolName = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

type anthropicRequest struct {
	Model     string             `json:"model"`
	System    []anthropicText    `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
	Stream    bool               `json:"stream"`
	MaxTokens int                `json:"max_tokens"`
	Thinking  *anthropicThinking `json:"thinking,omitempty"`
}
type anthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}
type anthropicText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type anthropicMessage struct {
	Role    string `json:"role"`
	Content []any  `json:"content"`
}
type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}
type anthropicToolUse struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}
type anthropicToolResult struct {
	Type    string `json:"type"`
	ID      string `json:"tool_use_id"`
	Content string `json:"content"`
	IsError bool   `json:"is_error,omitempty"`
}

func anthropicString(s string, max int, nonempty bool) bool {
	if len(s) > max || !utf8.ValidString(s) || (nonempty && s == "") {
		return false
	}
	return true
}
func anthropicID(s string) bool {
	if !anthropicString(s, 512, true) {
		return false
	}
	for _, r := range s {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}
func anthropicObject(raw []byte) bool {
	if len(raw) > anthropicMaxText || !utf8.Valid(raw) || !json.Valid(raw) || !anthropicJSONUnicode(raw) {
		return false
	}
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 1 && trimmed[0] == '{'
}

// encoding/json replaces unpaired escaped UTF-16 surrogates with U+FFFD. Refuse
// that lossy conversion while allowing literal backslashes in streamed JSON
// argument fragments; the completed argument object is validated separately.
func anthropicJSONUnicode(raw []byte) bool {
	hex4 := func(b []byte) (uint16, bool) {
		if len(b) < 4 {
			return 0, false
		}
		var n uint16
		for _, c := range b[:4] {
			n <<= 4
			switch {
			case c >= '0' && c <= '9':
				n += uint16(c - '0')
			case c >= 'a' && c <= 'f':
				n += uint16(c - 'a' + 10)
			case c >= 'A' && c <= 'F':
				n += uint16(c - 'A' + 10)
			default:
				return 0, false
			}
		}
		return n, true
	}
	inString := false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '"' {
			inString = !inString
			continue
		}
		if !inString || raw[i] != '\\' {
			continue
		}
		i++
		if i >= len(raw) {
			return false
		}
		if raw[i] != 'u' {
			continue
		}
		if len(raw)-i < 5 {
			return false
		}
		n, ok := hex4(raw[i+1:])
		if !ok {
			return false
		}
		i += 4
		if n >= 0xdc00 && n <= 0xdfff {
			return false
		}
		if n < 0xd800 || n > 0xdbff {
			continue
		}
		if len(raw)-i < 7 || raw[i+1] != '\\' || raw[i+2] != 'u' {
			return false
		}
		low, ok := hex4(raw[i+3:])
		if !ok || low < 0xdc00 || low > 0xdfff {
			return false
		}
		i += 6
	}
	return true
}

// Preflight JSON-compatible schemas without invoking custom marshal methods.
// The conservative escaping allowance bounds allocations before json.Marshal.
func anthropicSchemaBudget(value reflect.Value, depth int, budget *int) bool {
	if depth > 64 || *budget < 8 {
		return false
	}
	*budget -= 8
	if !value.IsValid() {
		return true
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return true
		}
		return anthropicSchemaBudget(value.Elem(), depth+1, budget)
	}
	if value.CanInterface() {
		if _, ok := value.Interface().(json.Marshaler); ok {
			return false
		}
		if _, ok := value.Interface().(interface{ MarshalText() ([]byte, error) }); ok {
			return false
		}
	}
	switch value.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return true
	case reflect.String:
		if !utf8.ValidString(value.String()) || value.Len() > anthropicMaxText {
			return false
		}
		*budget -= 6*value.Len() + 2
		return *budget >= 0
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String || value.Len() > *budget/8 {
			return false
		}
		it := value.MapRange()
		for it.Next() {
			if !anthropicSchemaBudget(it.Key(), depth+1, budget) || !anthropicSchemaBudget(it.Value(), depth+1, budget) {
				return false
			}
		}
		return true
	case reflect.Slice, reflect.Array:
		if value.Len() > *budget/8 {
			return false
		}
		for i := 0; i < value.Len(); i++ {
			if !anthropicSchemaBudget(value.Index(i), depth+1, budget) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// EncodeAnthropic lowers normalized text and complete client-tool call/result
// groups. It never fetches or dereferences input-schema references.
func EncodeAnthropic(config core.Config, messages []core.Message, tools []core.Tool) ([]byte, error) {
	invalid := func() ([]byte, error) { return nil, errors.New("invalid or unsupported Anthropic request") }
	if !anthropicString(config.Model, 512, true) || config.OutputBudget < 1 || len(messages) == 0 || len(messages) > anthropicMaxMessages || len(tools) > anthropicMaxBlocks {
		return invalid()
	}
	body := anthropicRequest{Model: config.Model, Stream: true, MaxTokens: config.OutputBudget}
	if config.ThinkingBudget > 0 {
		body.Thinking = &anthropicThinking{Type: "enabled", BudgetTokens: config.ThinkingBudget}
	}
	budget := maxResponse - 4096
	account := func(s string) bool { budget -= 6*len(s) + 512; return budget >= 0 }
	toolNames := map[string]bool{}
	for _, t := range tools {
		if t.Type != "function" || !anthropicToolName.MatchString(t.Function.Name) || toolNames[t.Function.Name] || !anthropicString(t.Function.Description, anthropicMaxText, false) || t.Function.Parameters == nil {
			return invalid()
		}
		if kind, ok := t.Function.Parameters["type"]; ok && kind != "object" {
			return invalid()
		}
		if !account(t.Function.Name) || !account(t.Function.Description) || !anthropicSchemaBudget(reflect.ValueOf(t.Function.Parameters), 0, &budget) {
			return invalid()
		}
		schema, err := json.Marshal(t.Function.Parameters)
		if err != nil || len(schema) > anthropicMaxText {
			return invalid()
		}
		toolNames[t.Function.Name] = true
		body.Tools = append(body.Tools, anthropicTool{Name: t.Function.Name, Description: t.Function.Description, InputSchema: schema})
	}
	appendMessage := func(role string, blocks []any) {
		if len(body.Messages) > 0 && body.Messages[len(body.Messages)-1].Role == role {
			body.Messages[len(body.Messages)-1].Content = append(body.Messages[len(body.Messages)-1].Content, blocks...)
		} else {
			body.Messages = append(body.Messages, anthropicMessage{Role: role, Content: blocks})
		}
	}
	pending := map[string]bool{}
	ids := map[string]bool{}
	started := false
	for _, m := range messages {
		if !anthropicString(m.Content, anthropicMaxText, false) {
			return invalid()
		}
		if !account(m.Content) {
			return invalid()
		}
		if m.Role != "assistant" && len(m.ToolCalls) != 0 || m.Role != "tool" && m.ToolCallID != "" {
			return invalid()
		}
		switch m.Role {
		case "system":
			if started || len(m.ToolCalls) > 0 || m.ToolCallID != "" {
				return invalid()
			}
			if m.Content != "" {
				body.System = append(body.System, anthropicText{Type: "text", Text: m.Content})
			}
		case "user":
			if len(pending) != 0 || m.Content == "" {
				return invalid()
			}
			started = true
			blocks := []any{anthropicText{Type: "text", Text: m.Content}}
			for _, img := range m.Images {
				// Base64 needs no JSON escaping, so it counts at its length.
				if budget -= len(img.Data) + 512; budget < 0 {
					return invalid()
				}
				blocks = append(blocks, map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": img.MediaType, "data": img.Data}})
			}
			appendMessage("user", blocks)
		case "assistant":
			if len(pending) != 0 || len(m.ToolCalls) > anthropicMaxBlocks {
				return invalid()
			}
			started = true
			blocks := []any{}
			if config.ThinkingBudget > 0 {
				// Thinking blocks are replayed verbatim (signatures intact) and
				// only while thinking is enabled.
				for _, raw := range m.AnthropicThinking {
					var kind struct {
						Type string `json:"type"`
					}
					if json.Unmarshal(raw, &kind) != nil || (kind.Type != "thinking" && kind.Type != "redacted_thinking") {
						return invalid()
					}
					if budget -= len(raw) + 512; budget < 0 {
						return invalid()
					}
					blocks = append(blocks, raw)
				}
			}
			if m.Content != "" {
				blocks = append(blocks, anthropicText{Type: "text", Text: m.Content})
			}
			for _, c := range m.ToolCalls {
				if c.Type != "function" || !anthropicID(c.ID) || ids[c.ID] || !anthropicToolName.MatchString(c.Function.Name) || !anthropicObject([]byte(c.Function.Arguments)) {
					return invalid()
				}
				if !account(c.ID) || !account(c.Function.Name) || !account(c.Function.Arguments) {
					return invalid()
				}
				ids[c.ID] = true
				pending[c.ID] = true
				blocks = append(blocks, anthropicToolUse{Type: "tool_use", ID: c.ID, Name: c.Function.Name, Input: json.RawMessage(c.Function.Arguments)})
			}
			if len(blocks) == 0 {
				return invalid()
			}
			appendMessage("assistant", blocks)
		case "tool":
			if !anthropicID(m.ToolCallID) || !pending[m.ToolCallID] {
				return invalid()
			}
			started = true
			delete(pending, m.ToolCallID)
			var envelope struct {
				Error bool `json:"error"`
			}
			isError := json.Unmarshal([]byte(m.Content), &envelope) == nil && envelope.Error
			appendMessage("user", []any{anthropicToolResult{Type: "tool_result", ID: m.ToolCallID, Content: m.Content, IsError: isError}})
		default:
			return invalid()
		}
	}
	if len(pending) != 0 || len(body.Messages) == 0 {
		return invalid()
	}
	for _, m := range body.Messages {
		if len(m.Content) > anthropicMaxBlocks {
			return invalid()
		}
	}
	b, err := json.Marshal(body)
	if err != nil || len(b) > maxResponse {
		return invalid()
	}
	return b, nil
}

type anthropicUsage struct {
	Input      *int `json:"input_tokens"`
	Output     *int `json:"output_tokens"`
	CacheRead  *int `json:"cache_read_input_tokens"`
	CacheWrite *int `json:"cache_creation_input_tokens"`
}
type anthropicStreamEvent struct {
	Type    string `json:"type"`
	Index   *int   `json:"index"`
	Message *struct {
		Type       string          `json:"type"`
		Role       string          `json:"role"`
		Content    json.RawMessage `json:"content"`
		Usage      *anthropicUsage `json:"usage"`
		StopReason *string         `json:"stop_reason"`
	} `json:"message"`
	Block *struct {
		Type      string          `json:"type"`
		ID        string          `json:"id"`
		Name      string          `json:"name"`
		Text      *string         `json:"text"`
		Input     json.RawMessage `json:"input"`
		Citations json.RawMessage `json:"citations"`
		Thinking  *string         `json:"thinking"`
		Signature *string         `json:"signature"`
		Data      *string         `json:"data"`
	} `json:"content_block"`
	Delta *struct {
		Type         string  `json:"type"`
		Text         *string `json:"text"`
		PartialJSON  *string `json:"partial_json"`
		Thinking     *string `json:"thinking"`
		Signature    *string `json:"signature"`
		StopReason   *string `json:"stop_reason"`
		StopSequence *string `json:"stop_sequence"`
	} `json:"delta"`
	Usage *anthropicUsage `json:"usage"`
	Error json.RawMessage `json:"error"`
}

func mergeAnthropicUsage(current *anthropicUsage, next *anthropicUsage) (*anthropicUsage, error) {
	if next == nil {
		return current, nil
	}
	merged := anthropicUsage{}
	if current != nil {
		merged = *current
	}
	for _, pair := range []struct {
		from *int
		to   **int
	}{{next.Input, &merged.Input}, {next.Output, &merged.Output}, {next.CacheRead, &merged.CacheRead}, {next.CacheWrite, &merged.CacheWrite}} {
		if pair.from != nil {
			if *pair.from < 0 || *pair.from > 1000000000 {
				return nil, errors.New("invalid Anthropic usage")
			}
			*pair.to = pair.from
		}
	}
	if _, err := anthropicCoreUsage(&merged); err != nil {
		return nil, err
	}
	return &merged, nil
}
func anthropicCoreUsage(u *anthropicUsage) (*core.Usage, error) {
	if u == nil {
		return nil, nil
	}
	v := core.Usage{}
	for _, n := range []*int{u.Input, u.CacheRead, u.CacheWrite} {
		if n != nil {
			if v.Prompt > 2000000000-*n {
				return nil, errors.New("invalid Anthropic usage")
			}
			v.Prompt += *n
		}
	}
	if u.Output != nil {
		v.Completion = *u.Output
	}
	if v.Prompt > 2000000000-v.Completion {
		return nil, errors.New("invalid Anthropic usage")
	}
	v.Total = v.Prompt + v.Completion
	return &v, nil
}

// ParseAnthropic consumes one complete Messages SSE response. Partial text may
// already have been delivered to delta when an error occurs; no partial call is
// returned for execution. All diagnostics are static and redact provider bodies.
func ParseAnthropic(r io.Reader, delta func(string)) (core.Reply, error) {
	return parseAnthropic(r, delta, nil)
}

// parseAnthropic also passes thinking text to reasoning as it streams.
func parseAnthropic(r io.Reader, delta func(string), reasoning func(string)) (core.Reply, error) {
	reply := core.Reply{Message: core.Message{Role: "assistant"}}
	var text strings.Builder
	var usage *anthropicUsage
	started, finished, stopped := false, false, false
	deltaPhase := false
	nextIndex := 0
	active := -1
	blockType := ""
	var call core.Call
	var arguments strings.Builder
	var initialInput json.RawMessage
	ids := map[string]bool{}
	var calls []core.Call
	invalidToolInput := false
	var thinking, signature strings.Builder
	redacted := ""
	appendText := func(value string) error {
		if !utf8.ValidString(value) || text.Len()+len(value) > anthropicMaxText {
			return errors.New("Anthropic text limit exceeded")
		}
		text.WriteString(value)
		if delta != nil {
			delta(value)
		}
		return nil
	}
	invalid := func() error { return errors.New("invalid Anthropic stream lifecycle or payload") }
	consumeEvent := func(raw, eventName string) error {
		if len(raw) > anthropicMaxEvent || !utf8.ValidString(raw) || !anthropicJSONUnicode([]byte(raw)) {
			return errors.New("invalid or oversized Anthropic SSE event")
		}
		var e anthropicStreamEvent
		if json.Unmarshal([]byte(raw), &e) != nil || e.Type == "" {
			return errors.New("invalid Anthropic SSE JSON")
		}
		if eventName != "" && eventName != e.Type {
			return invalid()
		}
		if e.Type == "error" || len(e.Error) > 0 && string(e.Error) != "null" {
			return errors.New("Anthropic provider stream error")
		}
		if e.Type == "ping" {
			return nil
		}
		switch e.Type {
		case "message_start":
			if started || stopped || e.Message == nil || e.Message.Role != "assistant" || e.Message.Type != "" && e.Message.Type != "message" || e.Message.StopReason != nil {
				return invalid()
			}
			if c := bytes.TrimSpace(e.Message.Content); len(c) > 0 && !bytes.Equal(c, []byte("[]")) {
				return errors.New("unsupported initial Anthropic content")
			}
			started = true
			var err error
			usage, err = mergeAnthropicUsage(usage, e.Message.Usage)
			return err
		case "content_block_start":
			if !started || deltaPhase || finished || stopped || active != -1 || e.Index == nil || *e.Index != nextIndex || nextIndex >= anthropicMaxBlocks || e.Block == nil {
				return invalid()
			}
			active = *e.Index
			nextIndex++
			blockType = e.Block.Type
			switch blockType {
			case "text":
				if e.Block.Text == nil || len(e.Block.Input) > 0 || e.Block.ID != "" || e.Block.Name != "" {
					return invalid()
				}
				if c := bytes.TrimSpace(e.Block.Citations); len(c) > 0 && !bytes.Equal(c, []byte("[]")) && !bytes.Equal(c, []byte("null")) {
					return errors.New("Anthropic citations are unsupported")
				}
				return appendText(*e.Block.Text)
			case "tool_use":
				if !anthropicID(e.Block.ID) || ids[e.Block.ID] || !anthropicToolName.MatchString(e.Block.Name) || !anthropicObject(e.Block.Input) || e.Block.Text != nil {
					return invalid()
				}
				ids[e.Block.ID] = true
				call = core.Call{Index: len(calls), ID: e.Block.ID, Type: "function"}
				call.Function.Name = e.Block.Name
				initialInput = e.Block.Input
				arguments.Reset()
				return nil
			case "thinking":
				thinking.Reset()
				signature.Reset()
				if e.Block.Thinking != nil {
					thinking.WriteString(*e.Block.Thinking)
				}
				if e.Block.Signature != nil {
					signature.WriteString(*e.Block.Signature)
				}
				return nil
			case "redacted_thinking":
				if e.Block.Data == nil {
					return invalid()
				}
				redacted = *e.Block.Data
				return nil
			default:
				return errors.New("unsupported Anthropic content block")
			}
		case "content_block_delta":
			if !started || finished || stopped || e.Index == nil || *e.Index != active || active < 0 || e.Delta == nil {
				return invalid()
			}
			switch e.Delta.Type {
			case "text_delta":
				if blockType != "text" || e.Delta.Text == nil || e.Delta.PartialJSON != nil {
					return invalid()
				}
				return appendText(*e.Delta.Text)
			case "input_json_delta":
				if blockType != "tool_use" || e.Delta.PartialJSON == nil || e.Delta.Text != nil {
					return invalid()
				}
				var initial map[string]json.RawMessage
				if json.Unmarshal(initialInput, &initial) != nil || len(initial) != 0 {
					return invalid()
				}
				if !utf8.ValidString(*e.Delta.PartialJSON) || arguments.Len()+len(*e.Delta.PartialJSON) > anthropicMaxText {
					return errors.New("Anthropic tool arguments exceed limit")
				}
				arguments.WriteString(*e.Delta.PartialJSON)
				return nil
			case "thinking_delta":
				if blockType != "thinking" || e.Delta.Thinking == nil || thinking.Len()+len(*e.Delta.Thinking) > anthropicMaxText {
					return invalid()
				}
				thinking.WriteString(*e.Delta.Thinking)
				if reasoning != nil && *e.Delta.Thinking != "" {
					reasoning(*e.Delta.Thinking)
				}
				return nil
			case "signature_delta":
				if blockType != "thinking" || e.Delta.Signature == nil || signature.Len()+len(*e.Delta.Signature) > 64<<10 {
					return invalid()
				}
				signature.WriteString(*e.Delta.Signature)
				return nil
			default:
				return errors.New("unsupported Anthropic content delta")
			}
		case "content_block_stop":
			if !started || finished || stopped || e.Index == nil || *e.Index != active || active < 0 {
				return invalid()
			}
			switch blockType {
			case "thinking":
				b, _ := json.Marshal(map[string]string{"type": "thinking", "thinking": thinking.String(), "signature": signature.String()})
				reply.Message.AnthropicThinking = append(reply.Message.AnthropicThinking, b)
			case "redacted_thinking":
				b, _ := json.Marshal(map[string]string{"type": "redacted_thinking", "data": redacted})
				reply.Message.AnthropicThinking = append(reply.Message.AnthropicThinking, b)
			}
			if blockType == "tool_use" {
				raw := initialInput
				if arguments.Len() > 0 {
					raw = []byte(arguments.String())
				}
				if !anthropicObject(raw) {
					invalidToolInput = true
				}
				call.Function.Arguments = string(raw)
				calls = append(calls, call)
			}
			active = -1
			blockType = ""
			return nil
		case "message_delta":
			if !started || stopped || active != -1 || e.Delta == nil || e.Delta.Type != "" || e.Delta.Text != nil || e.Delta.PartialJSON != nil || finished && e.Delta.StopReason != nil {
				return invalid()
			}
			deltaPhase = true
			var err error
			usage, err = mergeAnthropicUsage(usage, e.Usage)
			if err != nil {
				return err
			}
			if e.Delta.StopReason == nil {
				if e.Usage == nil {
					return invalid()
				}
				return nil
			}
			finished = true
			switch *e.Delta.StopReason {
			case "end_turn", "stop_sequence":
				reply.Finish = "stop"
			case "tool_use":
				reply.Finish = "tool_calls"
			case "max_tokens":
				reply.Finish = "length"
			case "refusal":
				reply.Finish = "content_filter"
			default:
				return errors.New("unsupported Anthropic finish reason")
			}
			return nil
		case "message_stop":
			if !started || !finished || stopped || active != -1 {
				return invalid()
			}
			stopped = true
			return nil
		default:
			// Unknown top-level event kinds are permitted by the Messages protocol.
			// Known content blocks/deltas remain strict to prevent silent data loss.
			return nil
		}
	}
	limited := &io.LimitedReader{R: r, N: maxResponse + 1}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 4096), anthropicMaxEvent+1)
	var event strings.Builder
	eventName := ""
	hasData := false
	consume := func() error {
		if !hasData {
			event.Reset()
			eventName = ""
			return nil
		}
		err := consumeEvent(event.String(), eventName)
		event.Reset()
		eventName = ""
		hasData = false
		return err
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := consume(); err != nil {
				return reply, err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, found := strings.Cut(line, ":")
		if !found {
			value = ""
		}
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "data":
			if hasData {
				event.WriteByte('\n')
			}
			if event.Len()+len(value) > anthropicMaxEvent {
				return reply, errors.New("Anthropic SSE event exceeds limit")
			}
			event.WriteString(value)
			hasData = true
		case "event":
			if len(value) > 128 || !utf8.ValidString(value) {
				return reply, errors.New("invalid Anthropic SSE event name")
			}
			eventName = value
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return reply, context.Canceled
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return reply, context.DeadlineExceeded
		}
		return reply, errors.New("Anthropic stream read failed")
	}
	if limited.N == 0 {
		return reply, errors.New("Anthropic stream exceeds byte limit")
	}
	if hasData {
		if err := consume(); err != nil {
			return reply, err
		}
	}
	if !started || !finished || !stopped || active != -1 {
		return reply, errors.New("interrupted Anthropic stream")
	}
	reply.Usage, _ = anthropicCoreUsage(usage)
	if reply.Finish == "length" || reply.Finish == "content_filter" {
		return reply, &FinishError{Reason: reply.Finish}
	}
	if invalidToolInput {
		return reply, errors.New("invalid Anthropic tool input")
	}
	if (len(calls) > 0) != (reply.Finish == "tool_calls") {
		return reply, errors.New("Anthropic finish reason/tool-call mismatch")
	}
	reply.Message.Content = text.String()
	reply.Message.ToolCalls = calls
	return reply, nil
}
