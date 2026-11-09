package tui

import (
	"testing"

	agent "github.com/Sanidhya14321/goh/internal/runtime"
)

func TestRevertFromCountsLaterIdenticalPromptsAndRefusesWhileRunning(t *testing.T) {
	prompts := []string{"again", "other", "again", "again"}
	m := &model{a: &agent.Agent{}, status: "ready"}
	if m.revertFrom(prompts, 0) == nil || m.navigation != "revert" || m.revertPrompt != "again" || m.revertOccurrence != 2 {
		t.Fatal(m.navigation, m.revertPrompt, m.revertOccurrence)
	}
	m = &model{a: &agent.Agent{}, status: "ready"}
	if m.revertFrom(prompts, 3) == nil || m.revertOccurrence != 0 {
		t.Fatal("latest occurrence", m.revertOccurrence)
	}
	m = &model{a: &agent.Agent{}, status: "running", active: true}
	if m.revertFrom(prompts, 1) != nil || m.navigation != "" {
		t.Fatal("revert allowed during a run")
	}
	m = &model{}
	if m.revertFrom(prompts, 1) != nil {
		t.Fatal("revert without a session")
	}
}
