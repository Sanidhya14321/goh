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
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCalls  []Call `json:"tool_calls,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}
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
	Prompt     int `json:"prompt_tokens"`
	Completion int `json:"completion_tokens"`
	Total      int `json:"total_tokens"`
}
type Reply struct {
	Message Message
	Finish  string
	Usage   *Usage
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
