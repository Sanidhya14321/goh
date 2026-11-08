package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeAgent(t *testing.T, path, text string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o700)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestProjectAgentFiles(t *testing.T) {
	p := t.TempDir()
	writeAgent(t, filepath.Join(p, ".opencode/agent/reviewer.md"), "---\ndescription: Reviews diffs\nmode: plan\nmodel: other/model\n---\nReview carefully.\n")
	writeAgent(t, filepath.Join(p, ".claude/agents/fixer.md"), "Fix bugs.")
	writeAgent(t, filepath.Join(p, ".claude/agents/reviewer.md"), "shadowed")
	writeAgent(t, filepath.Join(p, ".harness/agents/plan.md"), "reserved")
	writeAgent(t, filepath.Join(p, ".harness/agents/empty.md"), "---\ndescription: x\n---\n")
	writeAgent(t, filepath.Join(p, ".harness/agents/mine.md"), "from repo")
	c := Defaults()
	c.Agents = map[string]AgentProfile{"mine": {Description: "configured", Instructions: "from config", Mode: "build"}}
	c, skipped := c.WithProjectAgents(p)
	if r := c.Agents["reviewer"]; r.Mode != "plan" || r.Description != "Reviews diffs" || r.Instructions != "Review carefully." || r.Profile != "" {
		t.Fatalf("%+v", r)
	}
	if c.Agents["fixer"].Mode != "build" || c.Agents["mine"].Instructions != "from config" {
		t.Fatal(c.Agents)
	}
	if _, ok := c.Agents["plan"]; ok {
		t.Fatal("reserved name loaded")
	}
	if len(skipped) != 3 {
		t.Fatal(skipped)
	}
	resolved, err := c.Resolve("", "reviewer")
	if err != nil || resolved.AgentMode != "plan" || !strings.Contains(resolved.AgentInstructions, "Review carefully.") {
		t.Fatalf("%+v %v", resolved.AgentMode, err)
	}
}
