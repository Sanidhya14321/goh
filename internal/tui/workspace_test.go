package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func typeKeys(m tea.Model, keys ...string) {
	for _, k := range keys {
		switch k {
		case "enter":
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		case "tab":
			m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		case "left":
			m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
		case "down":
			m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		case "esc":
			m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		default:
			for _, r := range k {
				m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
			}
		}
	}
}

// Browse into a folder, open a project inside it, answer the trust question.
func TestWorkspaceBrowserOpensAndTrusts(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"code/app", "code/tool", ".hidden", "notes"} {
		os.MkdirAll(filepath.Join(root, d), 0700)
	}
	trusted := map[string]bool{}
	m := &workspacePicker{width: 80, height: 24, c: WorkspaceChooser{
		Start:   root,
		Check:   func(p string) error { return map[bool]error{true: errors.New("holds keys")}[p == root] },
		Trusted: func(p string) bool { return trusted[p] },
	}}
	m.browse(root)
	view := m.View().Content
	if strings.Contains(view, ".hidden") || !strings.Contains(view, "code"+string(filepath.Separator)) || strings.Contains(view, "Open "+root) {
		t.Fatal(view)
	}
	if !strings.Contains(view, "can't be a project") {
		t.Fatal("refused start folder not explained")
	}
	typeKeys(m, "code", "tab")
	if m.dir != filepath.Join(root, "code") {
		t.Fatal(m.dir)
	}
	typeKeys(m, "tool", "enter")
	if m.confirm != filepath.Join(root, "code", "tool") || !strings.Contains(m.View().Content, "Trust this folder?") {
		t.Fatal("trust not asked", m.confirm)
	}
	typeKeys(m, "esc")
	if m.confirm != "" || m.chosen != "" {
		t.Fatal("Esc must go back")
	}
	typeKeys(m, "esc", "app", "enter", "y") // Esc clears the filter
	if m.chosen != filepath.Join(root, "code", "app") {
		t.Fatal(m.chosen)
	}
	// A trusted folder opens without the question; ← returns to the parent.
	trusted[filepath.Join(root, "notes")] = true
	m.chosen = ""
	typeKeys(m, "esc", "left")
	if m.dir != root {
		t.Fatal(m.dir)
	}
	typeKeys(m, "notes", "enter")
	if m.chosen != filepath.Join(root, "notes") || m.confirm != "" {
		t.Fatal("trusted folder asked again")
	}
	// A typed path completes against the folders in its parent.
	m.chosen = ""
	m.browse(root)
	typeKeys(m, root+string(filepath.Separator)+"co")
	if rows := m.rows(); len(rows) != 1 || rows[0].path != filepath.Join(root, "code") {
		t.Fatal(rows)
	}
	// The refused folder itself cannot be opened.
	if _, _ = m.open(root); m.notice == "" || m.chosen != "" {
		t.Fatal("refused folder opened")
	}
}
