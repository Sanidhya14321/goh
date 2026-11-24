package tui

import (
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
)

// sessionLog is a realistic transcript: prompts, tool calls with results,
// an edit, a shell check and a Markdown reply with inline code and a block.
func sessionLog(turns int) string {
	turn := "You: Update `note.txt` to beta and verify it\n" +
		"\n▸ read_file  note.txt\n✓ note.txt\nalpha\n" +
		"\n▸ edit_file  note.txt\n✓ note.txt · 5 bytes\n" +
		"\n▸ shell  go test ./...\n✓ exit 0\nok  \texample/pkg\t0.12s\n" +
		"\nUpdated **note.txt** from `alpha` to `beta` and ran the tests; the [docs](https://example.com) say so.\n\n" +
		"```go\nfunc ready() bool {\n\treturn true\n}\n```\n\n- the edit was approved\n- the check printed `ok`\n" +
		verbatimMark + "gpt-6.1-sol · 4.2s · 6,000 in · 200 out\n\n"
	return strings.Repeat(turn, turns)
}

func benchModel() *model {
	return &model{a: &agent.Agent{Config: core.Config{Model: "gpt-6.1-sol", ContextWindow: 272000}, Store: &session.Store{ID: "fixture", Project: "/tmp/project"}},
		width: 120, height: 36, log: sessionLog(200)}
}

// BenchmarkSessionFrame is one redraw with nothing new (typing, a timer tick).
func BenchmarkSessionFrame(b *testing.B) {
	m := benchModel()
	m.View()
	b.ReportAllocs()
	for b.Loop() {
		m.View()
	}
}

// BenchmarkSessionReflow is one redraw after new text arrived (streaming).
func BenchmarkSessionReflow(b *testing.B) {
	m := benchModel()
	b.ReportAllocs()
	for b.Loop() {
		m.wrappedRows = nil
		m.View()
	}
}
