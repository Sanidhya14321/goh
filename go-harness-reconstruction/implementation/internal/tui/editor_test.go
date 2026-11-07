package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"example.com/independent-harness/internal/core"
)

func TestSplitWords(t *testing.T) {
	for in, want := range map[string][]string{
		"vim":                  {"vim"},
		"code --wait":          {"code", "--wait"},
		`"/Apps/My Editor" -n`: {"/Apps/My Editor", "-n"},
		`emacs -nw  '+set x'`:  {"emacs", "-nw", "+set x"},
		`nano a\ b`:            {"nano", "a b"},
	} {
		got, err := splitWords(in)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("%q: %q %v", in, got, err)
		}
	}
	if _, err := splitWords(`vim "unterminated`); err == nil {
		t.Fatal("accepted unterminated quote")
	}
}

func TestNormalizeEdited(t *testing.T) {
	for in, want := range map[string]string{
		"one line\n":   "one line",
		"one line\r\n": "one line",
		"two\nlines\n": "two\nlines\n",
		"no newline":   "no newline",
	} {
		if got := normalizeEdited(in); got != want {
			t.Fatalf("%q -> %q", in, got)
		}
	}
}

func TestEditorRequiresConfiguration(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	if _, err := editorCommand("/tmp/x"); err == nil {
		t.Fatal("expected a configuration error")
	}
	t.Setenv("EDITOR", "nano -w")
	t.Setenv("VISUAL", "code --wait")
	cmd, err := editorCommand("/tmp/x")
	if err != nil || !reflect.DeepEqual(cmd.Args, []string{"code", "--wait", "/tmp/x"}) {
		t.Fatal(cmd.Args, err)
	}
}

func TestRemappedKeysDriveActions(t *testing.T) {
	c := core.Defaults()
	c.Keybinds = map[string]string{"command_palette": "f5", "sidebar_toggle": "<leader>b"}
	m := &model{width: 40, height: 10, keys: keymapFor(&c)}
	m.Update(tea.KeyPressMsg{Code: tea.KeyF5})
	if !m.palette {
		t.Fatal("remapped palette key did nothing")
	}
	m.palette = false
	m.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	if !m.leader || !strings.Contains(m.status, "b sidebar toggle") {
		t.Fatal("leader hint missing", m.status)
	}
	m.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	if !m.hideSidebar || m.leader {
		t.Fatal("leader binding did not toggle the sidebar")
	}
	m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	if m.palette {
		t.Fatal("ctrl+p must no longer open the palette after remapping")
	}
}
