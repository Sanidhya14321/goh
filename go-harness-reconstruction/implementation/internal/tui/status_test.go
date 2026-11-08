package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/core"
	agent "example.com/independent-harness/internal/runtime"
	"example.com/independent-harness/internal/session"
	"github.com/charmbracelet/x/ansi"
)

func TestStatusSnapshotRedactsCredentialsAndKeepsDraft(t *testing.T) {
	t.Setenv("STATUS_FIXTURE_KEY", "LOCAL_SECRET")
	cfg := core.Defaults()
	cfg.Endpoint = "https://example.test/PATH_SECRET/chat?key=QUERY_SECRET"
	cfg.APIKeyEnv, cfg.Model, cfg.ContextWindow = "STATUS_FIXTURE_KEY", "fixture", 32768
	cfg.Agent, cfg.AgentMode = "plan", "plan"
	cfg.MCP = map[string]core.MCPConfig{"fixture": {Command: "unused", Env: map[string]string{"TOKEN": "INTEGRATION_SECRET"}}}
	m := &model{a: &agent.Agent{Config: cfg, Store: &session.Store{ID: "fixture", Project: "/tmp/project"}, Grants: map[string]bool{"read_file": true}},
		width: 120, height: 36, input: "unfinished draft", inputCursor: 16}
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	if cmd != nil || m.infoPanel == nil || m.active {
		t.Fatal("opening status started work or navigation")
	}
	all := strings.Join(m.infoPanel.lines, "\n")
	for _, secret := range []string{"LOCAL_SECRET", "PATH_SECRET", "QUERY_SECRET", "INTEGRATION_SECRET"} {
		if strings.Contains(all, secret) {
			t.Fatal("status reflected private configuration")
		}
	}
	for _, expected := range []string{"https://example.test", "Local key present", "Plan mode blocks mutation tools", "configured, starts on demand"} {
		if !strings.Contains(all, expected) {
			t.Fatal("missing status detail", expected)
		}
	}
	for _, size := range [][2]int{{24, 10}, {60, 18}, {120, 36}} {
		m.width, m.height = size[0], size[1]
		m.statusKey("end")
		view := m.View().Content
		lines := strings.Split(view, "\n")
		if len(lines) > m.height {
			t.Fatal("status exceeded terminal height")
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > m.width {
				t.Fatal("status exceeded terminal width")
			}
		}
	}
	m.Update(tea.PasteMsg{Content: "should not edit the draft"})
	m.statusKey("esc")
	if m.infoPanel != nil || m.input != "unfinished draft" || m.inputCursor != 16 {
		t.Fatal("closing status lost or edited the draft")
	}
}

func TestUnconfiguredStatusCanOpenProvider(t *testing.T) {
	m := &model{width: 80, height: 24, blocked: "setup required", input: "/status"}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.infoPanel == nil || m.active {
		t.Fatal("unconfigured status did not open")
	}
	_, cmd := m.statusKey("p")
	if cmd == nil || m.navigation != "provider" || m.infoPanel != nil {
		t.Fatal("provider shortcut failed")
	}
}

func BenchmarkStatusView(b *testing.B) {
	cfg := core.Defaults()
	cfg.Endpoint, cfg.Model, cfg.ContextWindow = "https://fixture.invalid/v1/chat/completions", "fixture", 32768
	m := &model{a: &agent.Agent{Config: cfg, Store: &session.Store{ID: "fixture", Project: "/tmp/project"}}, width: 120, height: 36}
	m.openStatus()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		m.View()
	}
}

func TestResumedToolResultPresentation(t *testing.T) {
	if got := historyToolText("{\"content\":\"literal\",\"path\":\"note.txt\"}"); got != "[result] note.txt\nliteral" {
		t.Fatal("legacy result lost its path", got)
	}
	if got := historyToolText("{\"content\":\"{\\\"exit_code\\\":0,\\\"output\\\":\\\"verified\\\",\\\"artifact\\\":\\\"internal-path\\\"}\",\"error\":false}"); got != "✓ exit 0\nverified" {
		t.Fatal("resume exposed the raw tool envelope", got)
	}
	if got := historyToolText("{\"content\":\"command failed\",\"error\":true}"); got != "[error] command failed" {
		t.Fatal("resume lost a tool error", got)
	}
}

func TestStatusExplainsConfigProblemsFailoverAndStoppedIntegrations(t *testing.T) {
	cfg := core.Defaults()
	cfg.Endpoint, cfg.Model, cfg.ContextWindow = "https://example.test/v1/chat/completions", "fixture", 32768
	cfg.Keybinds = map[string]string{"no_such_action": "ctrl+q"}
	cfg.MCP = map[string]core.MCPConfig{"broken": {Command: "unused"}, "fine": {Command: "unused"}}
	a := &agent.Agent{Config: cfg, Store: &session.Store{ID: "fixture", Project: "/tmp/project"}, Grants: map[string]bool{}}
	m := &model{a: a, width: 120, height: 36}
	m.openStatus()
	all := strings.Join(m.infoPanel.lines, "\n")
	for _, expected := range []string{"needs attention: keybinds: unknown action", "Failover: off", "Theme: ",
		"MCP fine · configured, starts on demand", "MCP broken · configured, starts on demand"} {
		if !strings.Contains(all, expected) {
			t.Fatalf("missing %q in:\n%s", expected, all)
		}
	}
	cfg.FallbackProfiles = []string{"backup"}
	if got := failoverStatus(cfg); !strings.Contains(got, "backup") || !strings.Contains(got, "before any response") {
		t.Fatal("failover summary", got)
	}
	if got := oneLine("a\n\tb   c"+strings.Repeat("x", 300), 20); strings.ContainsAny(got, "\n\t") || len([]rune(got)) != 21 {
		t.Fatal("oneLine", got)
	}
}

func TestStatusListsSkippedAgentFiles(t *testing.T) {
	cfg := core.Defaults()
	cfg.Endpoint, cfg.Model, cfg.ContextWindow = "https://example.test/v1/chat/completions", "fixture", 32768
	cfg.SkippedAgentFiles = []core.SkippedAgentFile{{Path: "/p/.opencode/agent/plan.md", Reason: "invalid or reserved agent name"}}
	m := &model{a: &agent.Agent{Config: cfg, Store: &session.Store{ID: "fixture", Project: "/p"}, Grants: map[string]bool{}}, width: 120, height: 36}
	m.openStatus()
	if all := strings.Join(m.infoPanel.lines, "\n"); !strings.Contains(all, "Agent file skipped") || !strings.Contains(all, "reserved agent name") {
		t.Fatal(all)
	}
}

func TestSkillDiagnosticsNoticeOnce(t *testing.T) {
	m := &model{a: &agent.Agent{Config: core.Defaults(), Store: &session.Store{ID: "fixture"}}, width: 80, height: 24}
	for i := 0; i < 2; i++ {
		m.Update(core.Event{Type: "skill_diagnostics", Data: []any{}})
	}
	if strings.Count(m.log, "skill file(s) could not be loaded") != 1 || strings.Contains(m.log, "[skill_diagnostics]") {
		t.Fatalf("%q", m.log)
	}
}
