package tui

import (
	"strings"

	"example.com/independent-harness/internal/session"
)

// renameSession handles "/rename title": it saves a title for the open
// session in its metadata sidecar (the journal is untouched). An empty title
// restores the default derived from the first prompt.
func (m *model) renameSession(arg string) {
	if m.a == nil || m.a.Store == nil || m.a.Config.SessionDir == "" {
		m.log += "\n[notice] No open session to rename.\n"
		return
	}
	title := session.CleanTitle(strings.TrimSpace(arg))
	if err := session.SetTitle(m.a.Config.SessionDir, m.a.Store.ID, title); err != nil {
		m.log += "\n[notice] Rename failed: " + Safe(err.Error()) + "\n"
		return
	}
	m.sessionTitle = Safe(title)
	if title == "" {
		m.log += "\n[notice] Session title reset to the first prompt.\n"
		return
	}
	m.log += "\n[notice] Session renamed to “" + Safe(title) + "”.\n"
}
