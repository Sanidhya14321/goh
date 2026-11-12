package tui

import (
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
)

// setEffort handles "/effort [level|off]" for this session
// only; the saved configuration is unchanged.
func (m *model) setEffort(arg string) {
	if m.a == nil {
		return
	}
	arg = strings.ToLower(strings.TrimSpace(arg))
	if arg == "" {
		current := m.a.Config.ReasoningEffort
		if current == "" {
			current = "off (provider default)"
		}
		family := "openai"
		if m.a.Config.EffectiveProtocol() == "anthropic_messages" {
			family = "anthropic"
		}
		m.log += "\n[notice] Reasoning effort: " + current + ". Use /effort " + strings.Join(core.ReasoningEfforts[family], "|") + "|off.\n"
		return
	}
	if m.active {
		m.log += "\n[notice] Reasoning effort can change after this run finishes.\n"
		return
	}
	if m.a.Remote != nil {
		m.log += "\n[notice] /effort is not available while attached; the server's configuration applies.\n"
		return
	}
	if arg == "off" {
		arg = ""
	}
	if err := m.a.SetReasoningEffort(arg); err != nil {
		m.log += "\n[notice] " + Safe(err.Error()) + "\n"
		return
	}
	if arg == "" {
		arg = "off"
	}
	m.status = "Reasoning effort " + arg + " for this session"
}
