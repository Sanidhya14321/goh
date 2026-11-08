package runtime

import (
	"context"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestCompactionThreshold(t *testing.T) {
	run := func(threshold int) bool {
		a := contextAgent(t)
		a.Config.ContextWindow, a.Config.OutputBudget, a.Config.CompactionThreshold = 100000, 2000, threshold
		off := false
		a.Config.CompactionMasking = &off
		a.History = append(a.History, core.Message{Role: "user", Content: "task"})
		for i := 0; i < 10; i++ {
			call := core.Call{ID: ID(), Type: "function"}
			call.Function.Name, call.Function.Arguments = "read_file", `{"path":"x"}`
			a.History = append(a.History, core.Message{Role: "assistant", ToolCalls: []core.Call{call}}, core.Message{Role: "tool", ToolCallID: call.ID, Content: strings.Repeat("a", 6000)})
		}
		m := &summaryModel{summary: "summary"}
		a.Provider = m
		if err := a.compact(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		return m.seen != nil
	}
	if run(0) {
		t.Fatal("about 60% full should not compact at the default 75%")
	}
	if !run(55) {
		t.Fatal("about 60% full should compact at a 55% threshold")
	}
	if (core.Config{CompactionThreshold: 40}).CompactAt() != 40 || (core.Config{}).CompactAt() != 75 {
		t.Fatal("CompactAt")
	}
}
