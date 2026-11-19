package tui

import (
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/sandbox"
)

// sessionAccess is the access level chosen for one chat session (Shift+Tab,
// the approval shortcut or /access "This session only"). It is bound to that
// session's journal: rebuilding the same chat (another agent, a picker
// closed, settings) keeps it, while a new session, another session or
// another workspace starts from the saved folder levels again.
var sessionAccess struct {
	level   core.AccessLevel
	session string
}

// SessionAccessFor is the level chosen for session, "" when none was or it
// was chosen for another session.
func SessionAccessFor(session string) core.AccessLevel {
	if session == "" || session != sessionAccess.session {
		return ""
	}
	return sessionAccess.level
}

// SetSessionAccessLevel records a level chosen outside the chat screen
// (the /access picker) for session; "" clears it.
func SetSessionAccessLevel(level core.AccessLevel, session string) {
	sessionAccess.level, sessionAccess.session = level, session
	if level == "" {
		sessionAccess.session = ""
	}
}

// accessLevel is the level that applies at the project root now.
func (m *model) accessLevel() core.AccessLevel {
	if m.a == nil {
		return core.AccessAsk
	}
	return m.a.ProjectAccess()
}

// setAccess applies level to the project for this session and says what
// it means.
func (m *model) setAccess(level core.AccessLevel) {
	if m.a == nil {
		return
	}
	if m.attached() {
		m.status = "Attached: the server's own access levels apply; this terminal cannot change them"
		return
	}
	m.a.SetSessionAccess(level)
	if m.a.Store != nil {
		SetSessionAccessLevel(level, m.a.Store.ID)
	}
	m.status = "Access: " + level.Label() + " for this session · " + accessNote(level)
}

// accessNote explains a level, including where this system cannot sandbox.
func accessNote(level core.AccessLevel) string {
	note := level.Describe()
	if level == core.AccessAuto && !sandbox.Supported() {
		note = "Edits files without asking; commands still ask (no sandbox on this system)"
	}
	return note + " · /access saves a level for a folder"
}

// cycleAccess moves to the next level (Shift+Tab).
func (m *model) cycleAccess() {
	m.setAccess(m.accessLevel().Next())
}

// coveringLevel is the lowest level that lets requests like p run without
// asking, for the approval shortcut; ok is false when none should be
// offered (network, MCP, or commands where there is no sandbox: those need
// full access, which is only chosen deliberately).
func coveringLevel(p core.Permission) (core.AccessLevel, bool) {
	switch strings.TrimPrefix(p.Tool, "child:") {
	case "create_file", "write_file", "edit_file", "apply_patch":
		return core.AccessEdits, true
	case "shell", "job_start":
		if sandbox.Supported() {
			return core.AccessAuto, true
		}
	}
	return "", false
}

// approvalAllShortcut is the "a" entry of the approval line, "" when the
// request has none.
func (m *model) approvalAllShortcut(p core.Permission) string {
	level, ok := coveringLevel(p)
	if !ok || m.accessLevel().Rank() >= level.Rank() {
		return ""
	}
	if level == core.AccessEdits {
		return "a all edits"
	}
	return "a all, sandboxed"
}

// accessLabel is the footer's access entry.
func (m *model) accessLabel() string {
	if m.attached() {
		return "Access: server's" // the server's own levels apply, not this terminal's
	}
	level := m.accessLevel()
	if level == core.AccessAsk {
		return "Access: Ask"
	}
	return "Access: " + level.Label()
}

// approvalAll joins the approval shortcut into an approval line.
func approvalAll(shortcut, sep string) string {
	if shortcut == "" {
		return ""
	}
	return sep + shortcut
}

// attached reports whether this chat runs on a GoHarness server (goh attach).
func (m *model) attached() bool { return m.a != nil && m.a.Remote != nil }

// approvalChoices is the one decision row for a pending approval: only the
// decisions the runtime behind this chat accepts. An attached chat sends a
// once-only decision to the server, so it offers y and n alone.
func (m *model) approvalChoices(sep string) string {
	if m.pending == nil {
		return ""
	}
	if m.attached() {
		return "y allow once on the server" + sep + "n / Esc deny"
	}
	return "y once" + sep + "s " + sessionScopeLabel(m.pending.request) + approvalAll(m.approvalAllShortcut(m.pending.request), sep) + sep + "n / Esc deny"
}
