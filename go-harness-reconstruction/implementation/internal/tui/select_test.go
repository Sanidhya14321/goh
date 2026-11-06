package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"example.com/independent-harness/internal/session"
	"github.com/charmbracelet/x/ansi"
)

func TestSessionBrowserSearchScopeAndSelection(t *testing.T) {
	m := &chooser{project: "/one", current: "recent", width: 120, height: 30,
		sessions:  []session.Info{{ID: "recent", Project: "/one"}, {ID: "other", Project: "/two"}},
		summaries: map[string]session.PreviewInfo{"recent": {Title: "Fix parser 界"}, "other": {Title: "Review API"}},
	}
	m.Update(tea.PasteMsg{Content: "parser"})
	if got := m.matches(); len(got) != 1 || got[0].ID != "recent" {
		t.Fatal("title search failed", got)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil || !m.chosen || m.selected != "recent" {
		t.Fatal("wrong session selected")
	}
	m.chosen = false
	m.query = "API"
	if len(m.matches()) != 0 {
		t.Fatal("workspace filter leaked a different project")
	}
	m.Update(tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl})
	if got := m.matches(); len(got) != 1 || got[0].ID != "other" {
		t.Fatal("all-workspace search failed", got)
	}
	for _, size := range [][2]int{{24, 10}, {80, 24}, {120, 30}} {
		m.width, m.height = size[0], size[1]
		rows := strings.Split(m.View().Content, "\n")
		if len(rows) > m.height {
			t.Fatal("browser exceeded terminal height")
		}
		for _, row := range rows {
			if ansi.StringWidth(row) > m.width {
				t.Fatal("browser exceeded terminal width")
			}
		}
	}
	m.query = "missing"
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.chosen {
		t.Fatal("empty search created a session")
	}
}
