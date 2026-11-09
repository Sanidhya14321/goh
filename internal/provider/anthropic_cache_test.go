package provider

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestEncodeAnthropicPromptCacheBreakpoints(t *testing.T) {
	tools := []core.Tool{
		core.NewTool("read_file", "Read", map[string]any{"type": "object"}),
		core.NewTool("grep", "Search", map[string]any{"type": "object"}),
	}
	messages := []core.Message{{Role: "system", Content: "stable rules"}, {Role: "system", Content: "project guidance"}, {Role: "user", Content: "inspect"},
		{Role: "assistant", Content: "checking", ToolCalls: []core.Call{anthroCall("tool_a", "read_file", `{"path":"a"}`)}},
		{Role: "tool", ToolCallID: "tool_a", Content: `{"error":false,"content":"text"}`}}
	b, err := EncodeAnthropic(anthroConfig(), messages, tools)
	if err != nil {
		t.Fatal(err)
	}
	var req struct {
		Tools    []map[string]any `json:"tools"`
		System   []map[string]any `json:"system"`
		Messages []struct {
			Content []map[string]any `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(b, &req); err != nil {
		t.Fatal(err)
	}
	marked := func(m map[string]any) bool { _, ok := m["cache_control"]; return ok }
	if marked(req.Tools[0]) || !marked(req.Tools[1]) || marked(req.System[0]) || !marked(req.System[1]) {
		t.Fatalf("tool/system breakpoints wrong: %s", b)
	}
	last := req.Messages[len(req.Messages)-1].Content
	if lb := last[len(last)-1]; !marked(lb) || lb["type"] != "tool_result" {
		t.Fatalf("newest block not marked: %s", b)
	}
	if n := strings.Count(string(b), `"cache_control"`); n != 3 {
		t.Fatalf("%d breakpoints, want 3 (API maximum is 4)", n)
	}
	cfg := anthroConfig()
	cfg.DisablePromptCache = true
	if b, err = EncodeAnthropic(cfg, messages, tools); err != nil || strings.Contains(string(b), "cache_control") {
		t.Fatalf("disable_prompt_cache ignored: %s %v", b, err)
	}
}
