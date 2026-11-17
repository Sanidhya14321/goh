package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
)

// A whole run under an 8k per-request cap (found live on Groq's free tier):
// a large page, several small lookups that fill the window until older turns
// are summarized, then the same page fetched again. After the one refusal
// that states the cap, no request exceeds it and the run completes.
func TestRequestCapLongRun(t *testing.T) {
	a, root := testAgent(t)
	a.Config.ContextWindow = 131072
	os.WriteFile(filepath.Join(root, "page.txt"), []byte(strings.Repeat("go1.25.4.linux-amd64.tar.gz 9fa5ffeda4170de60f67f3aa0f824e426421ba724c21e133c1e35d6159ca1bec\n", 600)), 0o600)
	calls, anchor := 0, 0
	reasoning := strings.Repeat("I should look at the page. ", 20)
	refusals, summaries := 0, 0
	a.Provider = modelFunc(func(_ context.Context, history []core.Message, defs []core.Tool, _ func(string)) (core.Reply, error) {
		if strings.HasPrefix(history[0].Content, "Summarize this coding session") {
			summaries++
			return core.Reply{Message: core.Message{Role: "assistant", Content: "Fetched go.dev/dl; grepped for releases."}, Finish: "stop"}, nil
		}
		calls++
		size, _ := requestBytes(history, defs)
		real := anchor/5 + (size-anchor)*10/23
		if calls == 1 {
			anchor = size
			real = size / 5
		}
		if real+a.Config.OutputBudget > 8000 {
			refusals++
			return core.Reply{}, &provider.HTTPError{Status: 413, Kind: "request_too_large", Limit: 8000, Requested: real + a.Config.OutputBudget}
		}
		u := &core.Usage{Prompt: real}
		msg := func(c core.Call) (core.Reply, error) {
			return core.Reply{Message: core.Message{Role: "assistant", Content: reasoning, ToolCalls: []core.Call{c}}, Finish: "tool_calls", Usage: u}, nil
		}
		switch calls {
		case 1:
			return msg(toolCall("c1", "read_file", map[string]any{"path": "page.txt"}))
		default:
			if calls < 8 {
				return msg(toolCall("c"+string(rune('0'+calls)), "grep", map[string]any{"pattern": "Released" + string(rune(0x30+calls)), "path": "."}))
			}
			if calls == 8 {
				return msg(toolCall("c8", "read_file", map[string]any{"path": "page.txt"}))
			}
			return core.Reply{Message: core.Message{Role: "assistant", Content: "done"}, Finish: "stop", Usage: u}, nil
		}
	})
	status, err := a.Run(context.Background(), "latest go?")
	if status != "completed" || err != nil || refusals != 1 || summaries == 0 {
		t.Fatal(status, err, "refusals", refusals, "summaries", summaries)
	}
}
