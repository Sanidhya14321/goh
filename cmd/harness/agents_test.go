package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestWriteAgents(t *testing.T) {
	c := core.Config{Agents: map[string]core.AgentProfile{
		"review": {Description: "Reviews\x1b[2J code", Mode: "subagent", Profile: "fast"},
		"docs":   {Description: "Writes docs"},
	}}
	var b bytes.Buffer
	writeAgents(&b, c, []core.SkippedAgentFile{{Path: "a.md", Reason: "bad mode"}})
	out := b.String()
	for _, want := range []string{"build ", "plan ", "docs       primary   Writes docs", "review     subagent", "(model profile fast)", "skipped a.md: bad mode"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b") || strings.Index(out, "docs") > strings.Index(out, "review") {
		t.Fatalf("unsafe or unsorted:\n%q", out)
	}
}
