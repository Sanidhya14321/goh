package tui

import "strings"

// setEffort handles "/effort [minimal|low|medium|high|off]" for this session
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
		m.log += "\n[notice] Reasoning effort: " + current + ". Use /effort minimal|low|medium|high|off.\n"
		return
	}
	if m.active {
		m.log += "\n[notice] Reasoning effort can change after this run finishes.\n"
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
