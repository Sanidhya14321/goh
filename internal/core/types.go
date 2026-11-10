// Package core defines the wire contracts shared by the agent and its adapters.
package core

import (
	"encoding/json"
	"time"
)

type Call struct {
	Index    int    `json:"-"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}
type Message struct {
	Role               string            `json:"role"`
	Content            string            `json:"content"`
	ToolCalls          []Call            `json:"tool_calls,omitempty"`
	ToolCallID         string            `json:"tool_call_id,omitempty"`
	ResponsesReasoning []json.RawMessage `json:"responses_reasoning,omitempty"`
	ResponsesRoute     string            `json:"responses_route,omitempty"`
	// AnthropicThinking holds an assistant turn's thinking blocks exactly as
	// received, replayed with that turn while extended thinking is on.
	AnthropicThinking []json.RawMessage `json:"anthropic_thinking,omitempty"`
	// Images attached to a user message (base64), sent as image parts.
	Images []Image `json:"images,omitempty"`
	// Automatic marks a user-role message GoHarness wrote itself (checks,
	// repair requests, pinned instructions). It is not a user turn for
	// undo, revert, timeline or statistics, and never sent to providers.
	Automatic bool `json:"automatic,omitempty"`
}

// Image is an attached picture; Data is standard base64 without a prefix.
type Image struct {
	Path      string `json:"path"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// DataURL returns the image as a data: URL.
func (i Image) DataURL() string { return "data:" + i.MediaType + ";base64," + i.Data }

type Tool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

func NewTool(name, description string, schema map[string]any) Tool {
	var t Tool
	t.Type = "function"
	t.Function.Name = name
	t.Function.Description = description
	t.Function.Parameters = schema
	return t
}

type Usage struct {
	// Prompt counts all input tokens, including cache reads and writes.
	Prompt     int `json:"prompt_tokens"`
	Completion int `json:"completion_tokens"`
	Total      int `json:"total_tokens"`
	// CacheRead and CacheWrite are the parts of Prompt served from or
	// written to the provider's prompt cache, when reported.
	CacheRead  int `json:"cache_read_tokens,omitempty"`
	CacheWrite int `json:"cache_write_tokens,omitempty"`
}
type Reply struct {
	Message Message
	Finish  string
	Usage   *Usage
	// Model is the model that actually served the reply (a fallback route
	// may differ from the configured one); empty means the configured one.
	Model string
}
type Event struct {
	Version   int       `json:"version"`
	SessionID string    `json:"session_id"`
	RunID     string    `json:"run_id"`
	Sequence  int       `json:"sequence"`
	Time      time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Data      any       `json:"data"`
}
type Permission struct {
	Tool      string          `json:"tool"`
	Preview   string          `json:"preview"`
	Arguments json.RawMessage `json:"arguments"`
}
type Decision struct {
	Allow   bool
	Session bool
}

// IntegrationFailure names one optional integration that could not start.
type IntegrationFailure struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

// MCPPrompt describes a prompt template an MCP server offers. Users invoke it
// as /mcp:<server>:<name>; Arguments lists required names first, optional
// ones with a trailing "?".
type MCPPrompt struct {
	Server, Name, Description string
	Arguments                 []string
}
