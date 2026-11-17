package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
	"github.com/Sanidhya14321/goh/internal/tools"
)

// A provider that refuses a request as larger than its stated per-request
// cap (Groq's free tier: "Limit 8000, Requested …") gets the same turn
// again, shrunk to fit, instead of a failed run.
func TestRequestCapShrinksAndResends(t *testing.T) {
	a, root := testAgent(t)
	a.Config.ContextWindow = 131072
	// Like go.dev/dl: mostly checksums, about 2.3 bytes a token, while
	// goh's own tools and instructions are about 5.
	page := strings.Repeat("go1.25.4.linux-amd64.tar.gz 9fa5ffeda4170de60f67f3aa0f824e426421ba724c21e133c1e35d6159ca1bec\n", 600)
	if err := os.WriteFile(filepath.Join(root, "page.txt"), []byte(page), 0o600); err != nil {
		t.Fatal(err)
	}
	calls, anchor := 0, 0
	var resent []core.Message
	a.Provider = modelFunc(func(_ context.Context, history []core.Message, defs []core.Tool, _ func(string)) (core.Reply, error) {
		calls++
		switch calls {
		case 1:
			anchor, _ = requestBytes(history, defs)
			return core.Reply{Message: core.Message{Role: "assistant", ToolCalls: []core.Call{toolCall("c1", "read_file", map[string]any{"path": "page.txt"})}}, Finish: "tool_calls", Usage: &core.Usage{Prompt: anchor / 5}}, nil
		case 2:
			size, _ := requestBytes(history, defs)
			// Requested counts the output reserve too.
			return core.Reply{}, &provider.HTTPError{Status: 413, Kind: "request_too_large", Limit: 8000, Requested: anchor/5 + (size-anchor)*10/23 + 4096}
		default:
			resent = history
			return core.Reply{Message: core.Message{Role: "assistant", Content: "summarized"}, Finish: "stop"}, nil
		}
	})
	learned := false
	a.Emit = func(e core.Event) {
		if e.Type == "request_cap_learned" {
			learned = true
		}
	}
	status, err := a.Run(context.Background(), "summarize page.txt")
	if status != "completed" || err != nil || calls != 3 {
		t.Fatal(status, err, calls)
	}
	if a.Config.ContextWindow != 8000 || a.Config.OutputBudget != 2000 {
		t.Fatal("cap not applied", a.Config.ContextWindow, a.Config.OutputBudget)
	}
	if size := a.estimate(resent, a.localDefinitions()); size > 8000-2000 {
		t.Fatal("resent request still exceeds the cap", size)
	}
	compacted := false
	for _, r := range a.Store.Records() {
		if r.Type == "context_compacted" {
			compacted = true
		}
	}
	if !compacted || !learned {
		t.Fatal("the oversized result was not shrunk, or the cap not reported", compacted, learned)
	}
}

// A 413 without a stated cap, or a second refusal after adapting twice,
// fails the run as before: there is nothing to learn from it.
func TestRequestCapWithoutNumbersFails(t *testing.T) {
	a, _ := testAgent(t)
	calls := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		calls++
		return core.Reply{}, &provider.HTTPError{Status: 413, Kind: "request_too_large"}
	})
	if status, err := a.Run(context.Background(), "hi"); status != "failed" || err == nil || calls != 1 {
		t.Fatal(status, err, calls)
	}
	calls = 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		calls++
		return core.Reply{}, &provider.HTTPError{Status: 413, Kind: "request_too_large", Limit: 900000, Requested: 1000000}
	})
	if status, err := a.Run(context.Background(), "hi"); status != "failed" || err == nil || calls != 1+maxCapAdapts {
		t.Fatal(status, err, calls)
	}
}

// Prompt-token counts the provider reports replace the one-token-per-byte
// guess: the smallest counted request anchors the fixed part (instructions
// and tools), and content beyond it uses its own learned ratio.
func TestUsageCalibratesEstimate(t *testing.T) {
	a, _ := testAgent(t)
	defs := a.localDefinitions() // turn requests carry the tools
	fixed := []core.Message{{Role: "user", Content: strings.Repeat("dense tool schema text ", 400)}}
	page := append(append([]core.Message(nil), fixed...), core.Message{Role: "tool", Content: strings.Repeat("9fa5ffeda4170de6 ", 1200)})
	before := a.estimate(fixed, defs)
	small, _ := requestBytes(fixed, defs)
	a.calibrate(small, small/5)
	if after := a.estimate(fixed, defs); after >= before/3 || after < small/5 {
		t.Fatal("anchor not used", before, after)
	}
	large, _ := requestBytes(page, defs)
	a.calibrate(large, small/5+(large-small)*10/23) // checksums: 2.3 bytes a token
	want := small/5 + (large-small)*10/23
	if got := a.estimate(page, defs); got < want || got > want*13/10 {
		t.Fatal("content ratio not learned", got, want)
	}
	// The fixed part keeps its own count: a dense page does not inflate it.
	if got := a.estimate(fixed, defs); got > small/5*12/10 {
		t.Fatal("fixed part inflated", got)
	}
	a.calibrate(large, large*2) // never below one byte a token
	if a.ratio() != 1 {
		t.Fatal(a.ratio())
	}
}

func TestSummaryEstimateSkipsAnchor(t *testing.T) {
	a, _ := testAgent(t)
	defs := a.localDefinitions()
	turn := []core.Message{{Role: "user", Content: "hi"}}
	size, _ := requestBytes(turn, defs)
	a.calibrate(size, size/5) // instructions and tools: about five bytes a token
	// A summary request has no tools: its bytes are content, not the anchor.
	summary := []core.Message{{Role: "user", Content: strings.Repeat("9fa5ffeda4170de6 ", 1200)}}
	if got, want := a.estimate(summary, nil), a.contentTokens(len(strings.Repeat("9fa5ffeda4170de6 ", 1200))); got < want {
		t.Fatal(got, want)
	}
}

// The oversized result can sit in the newest turn group, which masking
// always keeps; it is excerpted there too (found live: a second fetch after
// an earlier lookup ended the run with "compaction did not reduce context").
func TestRequestCapShrinksPreservedGroup(t *testing.T) {
	a, root := testAgent(t)
	a.Config.ContextWindow = 131072
	os.WriteFile(filepath.Join(root, "notes.txt"), []byte(strings.Repeat("an earlier small lookup\n", 40)), 0o600)
	os.WriteFile(filepath.Join(root, "page.txt"), []byte(strings.Repeat("go1.25.4.linux-amd64.tar.gz 9fa5ffeda4170de60f67f3aa0f824e426421ba724c21e133c1e35d6159ca1bec\n", 600)), 0o600)
	calls, anchor := 0, 0
	a.Provider = modelFunc(func(_ context.Context, history []core.Message, defs []core.Tool, _ func(string)) (core.Reply, error) {
		calls++
		size, _ := requestBytes(history, defs)
		switch calls {
		case 1:
			anchor = size
			return core.Reply{Message: core.Message{Role: "assistant", ToolCalls: []core.Call{toolCall("c1", "read_file", map[string]any{"path": "notes.txt"})}}, Finish: "tool_calls", Usage: &core.Usage{Prompt: size / 5}}, nil
		case 2:
			return core.Reply{Message: core.Message{Role: "assistant", ToolCalls: []core.Call{toolCall("c2", "read_file", map[string]any{"path": "page.txt"})}}, Finish: "tool_calls", Usage: &core.Usage{Prompt: size / 4}}, nil
		case 3:
			return core.Reply{}, &provider.HTTPError{Status: 413, Kind: "request_too_large", Limit: 8000, Requested: anchor/5 + (size-anchor)*10/23 + 4096}
		default:
			if a.estimate(history, defs) > 8000-2000 {
				t.Error("resent request still exceeds the cap")
			}
			return core.Reply{Message: core.Message{Role: "assistant", Content: "done"}, Finish: "stop"}, nil
		}
	})
	if status, err := a.Run(context.Background(), "compare notes.txt with page.txt"); status != "completed" || err != nil || calls != 4 {
		t.Fatal(status, err, calls)
	}
}

// An excerpt of an oversized result keeps its head only while recent; once
// older turns are masked it becomes a plain receipt (with its artifact id),
// so excerpts cannot pile up in a small window.
func TestOlderExcerptBecomesReceipt(t *testing.T) {
	big, _ := json.Marshal(tools.Result{Content: `{"artifact_id":"fetch-7.log","content":"` + strings.Repeat("x", 6000) + `"}`})
	history := []core.Message{{Role: "system"}, {Role: "user", Content: "go"},
		{Role: "assistant", ToolCalls: []core.Call{toolCall("f1", "webfetch", map[string]any{"url": "https://go.dev/dl/"})}},
		{Role: "tool", ToolCallID: "f1", Content: string(big)}}
	excerpted, n := shrinkLargeResults(history, 2048)
	if n != 1 || !strings.Contains(excerpted[3].Content, `"head"`) {
		t.Fatal(n, excerpted[3].Content)
	}
	masked, n := maskObservations(excerpted, len(excerpted))
	var r struct {
		ArtifactID string  `json:"artifact_id"`
		Note       string  `json:"note"`
		Head       *string `json:"head"`
	}
	if json.Unmarshal([]byte(masked[3].Content), &r); n != 1 || r.Head != nil || r.ArtifactID != "fetch-7.log" || strings.Contains(r.Note, "only the beginning") || len(masked[3].Content) > 600 {
		t.Fatal(n, masked[3].Content)
	}
}
