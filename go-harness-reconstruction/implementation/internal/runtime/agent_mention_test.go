package runtime

import (
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestAgentMentions(t *testing.T) {
	a := &Agent{Config: core.Config{Agents: map[string]core.AgentProfile{"docs": {Mode: "subagent"}}}}
	got, found := a.withAgentMentions("@general find the bug, then @docs. Also @unknown and @shot.png")
	if strings.Join(found, ",") != "docs,general" || !strings.Contains(got, "mentioned the docs, general subagents") {
		t.Fatal(found, got)
	}
	if got, found := a.withAgentMentions("plain email me@general.com"); found != nil || got != "plain email me@general.com" {
		t.Fatal(found, got)
	}
	child := &Agent{depth: 1}
	if _, found := child.withAgentMentions("@general x"); found != nil {
		t.Fatal("child agent should not get delegation notes")
	}
}
