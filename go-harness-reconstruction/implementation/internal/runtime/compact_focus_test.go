package runtime

import (
	"context"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestCompactWithFocus(t *testing.T) {
	a := contextAgent(t)
	a.History = append(a.History, core.Message{Role: "user", Content: "task"})
	readGroups(a, 6, 6000)
	m := &summaryModel{summary: "Focused summary."}
	a.Provider = m
	if status, err := a.CompactWith(context.Background(), " the API decisions "); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	if len(m.seen) == 0 || !strings.HasSuffix(m.seen[0].Content, "focus on: the API decisions") {
		t.Fatalf("%+v", m.seen)
	}
	if a.compactFocus != "" {
		t.Fatal("focus leaked into later compactions")
	}
}
