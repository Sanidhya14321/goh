package ignore

import (
	"strings"
	"testing"
)

func matcher(files map[string]string) *Matcher {
	m := &Matcher{}
	// Root first, then deeper files, as a directory walk would add them.
	for _, base := range []string{"", "pkg", "pkg/sub"} {
		if body, ok := files[base]; ok {
			m.Add(base, strings.NewReader(body))
		}
	}
	return m
}

func TestGitignoreSemantics(t *testing.T) {
	m := matcher(map[string]string{
		"":    "# comment\n\n*.log\n!keep.log\nbuild/\n/root-only.txt\ndocs/*.tmp\n**/cache\na/**/z\nspaced\\ \n",
		"pkg": "generated.go\n/local\n",
	})
	cases := []struct {
		path  string
		dir   bool
		want  bool
		about string
	}{
		{"app.log", false, true, "glob"},
		{"deep/x/app.log", false, true, "unanchored matches at any depth"},
		{"keep.log", false, false, "negation re-includes"},
		{"build", true, true, "dir-only rule on a directory"},
		{"build", false, false, "dir-only rule skips a file of that name"},
		{"src/build", true, true, "dir-only unanchored at depth"},
		{"root-only.txt", false, true, "leading slash anchors"},
		{"sub/root-only.txt", false, false, "anchored rule does not match deeper"},
		{"docs/a.tmp", false, true, "middle slash anchors"},
		{"x/docs/a.tmp", false, false, "middle-slash rule is anchored to base"},
		{"cache", true, true, "**/ prefix matches at root"},
		{"x/y/cache", true, true, "**/ prefix matches deeper"},
		{"a/z", false, true, "/**/ matches zero segments"},
		{"a/b/c/z", false, true, "/**/ matches several segments"},
		{"spaced ", false, true, "escaped trailing space kept"},
		{"pkg/generated.go", false, true, "nested file rule"},
		{"generated.go", false, false, "nested rule does not apply outside its dir"},
		{"pkg/local", false, true, "nested anchored rule"},
		{"pkg/x/local", false, false, "nested anchored rule is relative to pkg"},
		{"main.go", false, false, "unmatched"},
	}
	for _, c := range cases {
		if got := m.Ignored(c.path, c.dir); got != c.want {
			t.Errorf("%s (%s): got %v want %v", c.path, c.about, got, c.want)
		}
	}
}

func TestDeeperFileOverridesRoot(t *testing.T) {
	m := matcher(map[string]string{"": "*.gen\n", "pkg": "!keep.gen\n"})
	if !m.Ignored("pkg/other.gen", false) || m.Ignored("pkg/keep.gen", false) {
		t.Fatal("deeper negation should win for its subtree only")
	}
	if !m.Ignored("keep.gen", false) {
		t.Fatal("root rule must still apply outside pkg")
	}
}
