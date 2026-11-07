package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestFileCompletionProjectScopeAndDraft(t *testing.T) {
	project, outside := t.TempDir(), t.TempDir()
	for _, name := range []string{"docs/界 notes.md", "docs/notes.md", ".git/config", "node_modules/pkg/notes.md", "vendor/pkg/notes.md", ".env"} {
		file := filepath.Join(project, name)
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("local fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(project, "outside")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(project, "docs/notes.md"), filepath.Join(project, "alias.md")); err != nil {
		t.Fatal(err)
	}
	index := scanProjectFiles(project, nil)
	if got := fileMatches(index.files, ""); index.failed || index.limited ||
		!slices.Equal(got, []string{"docs/notes.md", "docs/界 notes.md"}) {
		t.Fatalf("unexpected file index: %v", got)
	}
	m := &model{width: 30, height: 10, input: "Check  please", inputCursor: 6, fileIndex: &index}
	m.Update(tea.KeyPressMsg{Code: '@', Text: "@"})
	m.Update(tea.PasteMsg{Content: "界n"})
	if m.fileCompletion == nil || !slices.Equal(m.fileCompletion.matches, []string{"docs/界 notes.md"}) {
		t.Fatal("Unicode fuzzy query did not match")
	}
	lines := strings.Split(m.View().Content, "\n")
	if len(lines) > m.height {
		t.Fatal("completion exceeded terminal height")
	}
	for _, line := range lines {
		if ansi.StringWidth(line) > m.width {
			t.Fatal("completion exceeded terminal width")
		}
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	want := `Check @"docs/界 notes.md" please`
	if m.input != want || m.inputCursor != utf8.RuneCountInString(`Check @"docs/界 notes.md"`) || m.active {
		t.Fatalf("completion changed surrounding draft or sent it: %q", m.input)
	}

	m.input, m.inputCursor = "mail a", 6
	m.Update(tea.KeyPressMsg{Code: '@', Text: "@"})
	if m.fileCompletion != nil || m.input != "mail a@" {
		t.Fatal("email address triggered file completion")
	}
	m.input, m.inputCursor = "Check ", 6
	m.Update(tea.KeyPressMsg{Code: '@', Text: "@"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.fileCompletion != nil || m.input != "Check @" {
		t.Fatal("dismissing the picker changed the draft")
	}
}

func TestFileScanRespectsGitignore(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(".gitignore", "*.log\nbuild/\n!keep.log\n")
	write("pkg/.gitignore", "generated.go\n")
	for _, f := range []string{"main.go", "app.log", "keep.log", "build/out.bin", "pkg/lib.go", "pkg/generated.go", "generated.go"} {
		write(f, "x")
	}
	got := map[string]bool{}
	for _, f := range scanProjectFiles(root, nil).files {
		got[f.path] = true
	}
	for _, want := range []string{"main.go", "keep.log", "pkg/lib.go", "generated.go"} {
		if !got[want] {
			t.Errorf("missing %s: %v", want, got)
		}
	}
	for _, hidden := range []string{"app.log", "build/out.bin", "pkg/generated.go"} {
		if got[hidden] {
			t.Errorf("ignored file listed: %s", hidden)
		}
	}
}

func TestFileIndexRefreshKeepsOldMatchesWhileRescanning(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "old.go"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	index := scanProjectFiles(project, nil)
	t.Chdir(project) // no agent: completion indexes the working directory
	m := &model{width: 40, height: 10, input: "@", inputCursor: 1, fileIndex: &index}

	// Fresh snapshot: reuse it without rescanning.
	if cmd := m.openFileCompletion(); cmd != nil {
		t.Fatal("fresh snapshot should not rescan")
	}
	// A finished run marks the snapshot stale: old matches stay visible and a
	// background rescan picks up the new file.
	if err := os.WriteFile(filepath.Join(project, "new.go"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m.fileIndexStale = true
	cmd := m.openFileCompletion()
	if cmd == nil || !slices.Equal(m.fileCompletion.matches, []string{"old.go"}) {
		t.Fatal("expected old matches plus a rescan", m.fileCompletion.matches)
	}
	if again := m.openFileCompletion(); again != nil {
		t.Fatal("a second rescan started while one was in flight")
	}
	m.Update(cmd())
	if got := fileMatches(m.fileIndex.files, ""); !slices.Equal(got, []string{"new.go", "old.go"}) || m.fileRescanning {
		t.Fatal("rescan result not applied", got)
	}
}
