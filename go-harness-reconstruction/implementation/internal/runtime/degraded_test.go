package runtime

import "testing"

func TestDegradedIntegrationsAreReportedForStatus(t *testing.T) {
	a := contextAgent(t)
	if len(a.DegradedIntegrations()) != 0 {
		t.Fatal("fresh agent reported degraded integrations")
	}
	if err := a.integrationDegraded("mcp", "broken", "exec: not found"); err != nil {
		t.Fatal(err)
	}
	got := a.DegradedIntegrations()
	if got["MCP broken"] != "exec: not found" {
		t.Fatal(got)
	}
	got["MCP broken"] = "mutated"
	if a.DegradedIntegrations()["MCP broken"] != "exec: not found" {
		t.Fatal("caller mutated agent state")
	}
}
