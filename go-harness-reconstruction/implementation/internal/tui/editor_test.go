package tui

import (
	"reflect"
	"testing"
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
