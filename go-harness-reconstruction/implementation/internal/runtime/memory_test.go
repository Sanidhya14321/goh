package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/memory"
)

func TestProjectMemoryRequiresOptInAndExplicitRecall(t *testing.T) {
	a := contextAgent(t)
	if err := os.WriteFile(filepath.Join(a.Store.Project, "parser.go"), []byte("package parser\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ledger, err := memory.New(a.Config.SessionDir, a.Store.Project)
	if err != nil {
		t.Fatal(err)
	}
	const note = "Parser fixture uses a project-specific grammar."
	if _, err := ledger.Add(note, []string{"parser.go"}); err != nil {
		t.Fatal(err)
	}
	raw := json.RawMessage(`{"query":"parser grammar"}`)
	if _, _, err := a.prepare("memory_recall", raw); err == nil {
		t.Fatal("recall enabled by default")
	}
	a.Config.Memory = true
	a.Config.AgentMode = "plan"
	if _, sideEffect, err := a.prepare("memory_recall", raw); err != nil || sideEffect {
		t.Fatalf("plan mode recall rejected: %v", err)
	}
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, history []core.Message, defs []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		if turn == 1 {
			for _, m := range history {
				if strings.Contains(m.Content, note) {
					t.Fatal("memory was injected before a recall")
				}
			}
			advertised := false
			for _, tool := range defs {
				advertised = advertised || tool.Function.Name == "memory_recall"
			}
			if !advertised {
				t.Fatal("recall missing from plan mode tools")
			}
			call := core.Call{ID: "memory-test", Type: "function"}
			call.Function.Name, call.Function.Arguments = "memory_recall", string(raw)
			return core.Reply{Message: core.Message{Role: "assistant", ToolCalls: []core.Call{call}}}, nil
		}
		if last := history[len(history)-1]; last.Role != "tool" || !strings.Contains(last.Content, note) {
			t.Fatal("missing explicit recall result")
		}
		return core.Reply{Message: core.Message{Role: "assistant", Content: "done"}}, nil
	})
	if status, err := a.Run(context.Background(), "inspect parser"); err != nil || status != "completed" || turn != 2 {
		t.Fatal(status, err, turn)
	}
}
