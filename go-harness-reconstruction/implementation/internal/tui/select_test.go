package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

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

func TestChooserRenameAndDelete(t *testing.T) {
	dir, project := t.TempDir(), t.TempDir()
	var ids []string
	for i := 0; i < 2; i++ {
		s, err := session.New(dir, project)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, s.ID)
		s.Close()
		time.Sleep(1100 * time.Millisecond) // distinct, ordered IDs
	}
	infos, _ := session.List(dir)
	m := &chooser{sessions: infos, dir: dir, current: ids[0], project: infos[0].Project}
	sort.Slice(m.sessions, func(i, j int) bool { return m.sessions[i].ID > m.sessions[j].ID })
	m.index = 1 // row 0 is "Start a new session"; row 1 is the newest (ids[1])

	m.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	for _, r := range "Renamed" {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if meta, _ := session.ReadMeta(dir, ids[1]); meta.Title != "Renamed" || m.summaries[ids[1]].Title != "Renamed" {
		t.Fatal("rename not saved", meta, m.notice)
	}

	m.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if _, err := os.Stat(filepath.Join(dir, ids[1]+".jsonl")); err != nil || !strings.Contains(m.notice, "again") {
		t.Fatal("first Ctrl+D must only ask for confirmation", m.notice)
	}
	m.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if _, err := os.Stat(filepath.Join(dir, "trash", ids[1]+".jsonl")); err != nil {
		t.Fatal("second Ctrl+D did not trash the session", m.notice)
	}

	m.index = 1 // now the current session (ids[0])
	m.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	m.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if _, err := os.Stat(filepath.Join(dir, ids[0]+".jsonl")); err != nil || !strings.Contains(m.notice, "current session") {
		t.Fatal("the current session must not be deleted", m.notice)
	}
}
