package tui

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/formatters"
)

type statusPanel struct {
	title    string // empty for the status screen; others cannot refresh
	lines    []string
	backdrop string
	offset   int
}

func (m *model) openStatus() {
	backdrop := ""
	offset := 0
	if m.infoPanel != nil {
		backdrop = m.infoPanel.backdrop
		offset = m.infoPanel.offset
	} else {
		backdrop = m.View().Content
	}
	lines := []string{"# PROVIDER"}
	if m.a == nil {
		lines = append(lines, "Chat is waiting for setup.", "Use p to configure the provider.")
	} else {
		cfg := m.a.Config
		configuration := "Configuration valid"
		if err := cfg.Validate(); err != nil {
			configuration = "Configuration needs attention: " + oneLine(err.Error(), 160)
		}
		keyState := "No local key"
		if cfg.EffectiveProtocol() == core.ChatGPTProtocol {
			keyState = "ChatGPT sign-in required"
			if _, err := core.ChatGPTModel(); err == nil {
				keyState = "ChatGPT authorization saved locally"
			}
		} else if key, err := cfg.ProviderKey(); err != nil {
			keyState = "Local key unavailable"
		} else if key != "" {
			keyState = "Local key present"
		}
		origin := "URL needs configuration"
		if endpoint, err := cfg.ProviderEndpoint(); err == nil {
			if parsed, err := url.Parse(endpoint); err == nil {
				// Paths and queries may contain a provider's credential.
				origin = parsed.Scheme + "://" + parsed.Host
			}
		}
		lines = append(lines, configuration, "Model: "+cfg.Model, "Protocol: "+cfg.EffectiveProtocol(),
			"Origin: "+origin, keyState, failoverStatus(cfg), "", "# CONTEXT",
			fmt.Sprintf("Context window: %d tokens", cfg.ContextWindow),
			fmt.Sprintf("Output reserve: %d tokens", cfg.OutputBudget),
			fmt.Sprintf("Reported usage in this view: %d tokens", m.usage),
			fmt.Sprintf("Turn limit: %d", cfg.MaxTurns), "", "# SESSION",
			"Agent: "+cfg.Agent+" · "+cfg.AgentMode, "Profile: "+cfg.Profile,
			"Project: "+m.a.Store.Project, "Session: "+m.a.Store.ID,
			fmt.Sprintf("Memory: %t · Checkpoints: %t", cfg.Memory, cfg.Snapshots),
			"Theme: "+currentTheme,
			"", "# INTEGRATIONS", "Entries describe local configuration.")
		var entries []string
		degraded := m.a.DegradedIntegrations()
		describe := func(key string) string {
			if detail, ok := degraded[key]; ok {
				return key + " · failed and stopped: " + oneLine(detail, 120)
			}
			return key + " · configured, starts on demand"
		}
		for name := range cfg.MCP {
			entries = append(entries, describe("MCP "+name))
		}
		for name := range cfg.LSP {
			entries = append(entries, describe("LSP "+name))
		}
		sort.Strings(entries)
		lines = append(lines, entries...)
		if len(entries) == 0 {
			lines = append(lines, "No MCP or LSP servers configured")
		}
		discovered, err := formatters.Discover(cfg.Formatters, m.a.Store.Project)
		if err != nil {
			lines = append(lines, "Formatter configuration needs attention")
		} else {
			for _, entry := range discovered {
				state := "disabled"
				if entry.Enabled {
					state = "executable missing"
					if entry.Available {
						state = "executable available"
					}
				}
				lines = append(lines, "Formatter "+entry.Name+" · "+state)
			}
		}
		for _, skip := range m.a.SkippedCommands {
			lines = append(lines, "Command file skipped · "+oneLine(Safe(skip.Path), 60)+": "+skip.Reason)
		}
		lines = append(lines, "", "# PERMISSIONS")
		if cfg.AgentMode == "plan" {
			lines = append(lines, "Plan mode blocks mutation tools.")
		}
		var grants []string
		for name, granted := range m.a.Grants {
			if granted {
				grants = append(grants, name)
			}
		}
		sort.Strings(grants)
		if len(grants) == 0 {
			lines = append(lines, "No explicit session grants")
		} else {
			lines = append(lines, grants...)
		}
		lines = append(lines, "", "Provider authentication is checked when a request is sent.")
	}
	m.infoPanel = &statusPanel{lines: lines, backdrop: backdrop, offset: offset}
}

func (m *model) statusKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "ctrl+c", "ctrl+d", "enter":
		m.infoPanel = nil
	case "up":
		m.infoPanel.offset = max(0, m.infoPanel.offset-1)
	case "down":
		m.infoPanel.offset++
	case "pgup":
		m.infoPanel.offset = max(0, m.infoPanel.offset-8)
	case "pgdown":
		m.infoPanel.offset += 8
	case "home":
		m.infoPanel.offset = 0
	case "end":
		width := max(1, min(82, m.width-4)-4)
		m.infoPanel.offset = len(terminalRows(strings.Join(m.infoPanel.lines, "\n"), width))
	case "r":
		if m.infoPanel.title == "" {
			m.openStatus()
		}
	case "p", "m", "a", "s":
		action := map[string]string{"p": "provider", "m": "models", "a": "agents", "s": "settings"}[key]
		m.infoPanel = nil
		m.navigation = action
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) statusView(width, height int) tea.View {
	boxWidth := min(82, max(1, width-4))
	left := max(0, (width-boxWidth)/2)
	padding := strings.Repeat(" ", left)
	var content []transcriptRow
	for _, line := range m.infoPanel.lines {
		color := foreground
		if strings.HasPrefix(line, "# ") {
			line, color = strings.TrimPrefix(line, "# "), accent
		}
		for _, wrapped := range terminalRows(line, max(1, boxWidth-4)) {
			content = append(content, transcriptRow{text: wrapped, fg: color})
		}
	}
	visible := max(1, min(len(content), height-7))
	offset := min(m.infoPanel.offset, max(0, len(content)-visible))
	m.infoPanel.offset = offset
	top := max(0, (height-visible-5)/2)
	rows := make([]string, top)
	title, hint := "Status", "  Local snapshot · r refresh · Esc back"
	if m.infoPanel.title != "" {
		title, hint = m.infoPanel.title, "  Snapshot · Esc back"
	}
	rows = append(rows, padding+cell("  GoHarness  /  "+title, boxWidth, foreground, panel),
		padding+cell(hint, boxWidth, muted, panel))
	for _, row := range content[offset : offset+visible] {
		rows = append(rows, padding+cell("  "+row.text, boxWidth, row.fg, panel))
	}
	rows = append(rows, padding+cell("  ↑/↓ scroll · PgUp/PgDn · Home/End", boxWidth, muted, panel),
		padding+cell("  p provider · m models · a agents · s settings", boxWidth, accent, panel))
	return overlayView(m.infoPanel.backdrop, rows, left, top, width, height)
}

// failoverStatus summarizes explicit fallback profiles; none means a failed
// request is reported, never sent elsewhere.
func failoverStatus(cfg core.Config) string {
	if len(cfg.FallbackProfiles) == 0 {
		return "Failover: off (no fallback_profiles)"
	}
	return "Failover: " + strings.Join(cfg.FallbackProfiles, " → ") + " (before any response only)"
}

func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		s = string(r[:max]) + "…"
	}
	return s
}
