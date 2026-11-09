package tui

import (
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
)

func TestCompletionOffersAgentsBeforeFiles(t *testing.T) {
	cfg := core.Config{Agents: map[string]core.AgentProfile{"gendocs": {Mode: "subagent"}}}
	m := &model{a: &agent.Agent{Config: cfg, Store: &session.Store{ID: "fixture"}}}
	files := []projectFile{{path: "general.go", basename: "general.go", folded: "general.go"}}
	got := m.completionMatches(files, "gen")
	if len(got) != 3 || got[0] != "gendocs" || got[1] != "general" || got[2] != "general.go" {
		t.Fatal(got)
	}
	if got := m.completionMatches(files, "zzz"); len(got) != 0 {
		t.Fatal(got)
	}
}
