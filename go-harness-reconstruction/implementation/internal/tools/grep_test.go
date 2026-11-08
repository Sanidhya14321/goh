package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrepIncludeAndFileScope(t *testing.T) {
	dir := t.TempDir()
	for p, s := range map[string]string{"root.txt": "needle root\n", "src/one.go": "needle go\nNEEDLE upper\n", "src/two.ts": "needle ts\n", ".hidden": "needle hidden\n", "binary.dat": "\x00needle binary\n"} {
		if e := os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(dir, p), []byte(s), 0600); e != nil {
			t.Fatal(e)
		}
	}
	e, x := New(dir, t.TempDir())
	if x != nil {
		t.Fatal(x)
	}
	for _, tc := range []struct {
		args  string
		count int
		paths []string
	}{{`{"pattern":"needle","include":"*.go"}`, 1, []string{"src/one.go"}}, {`{"pattern":"(?i)NEEDLE","path":"src"}`, 3, []string{"src/one.go", "src/one.go", "src/two.ts"}}, {`{"pattern":"needle","include":"*.{go,ts}"}`, 2, []string{"src/one.go", "src/two.ts"}}, {`{"pattern":"needle","path":"src/one.go"}`, 1, []string{"src/one.go"}}, {`{"pattern":"never-present"}`, 0, nil}} {
		r := e.Execute(context.Background(), "grep", json.RawMessage(tc.args))
		if r.Error {
			t.Fatal(r.Content)
		}
		var got struct {
			Matches []struct {
				Path string
				Line int
				Text string
			}
		}
		if x = json.Unmarshal([]byte(r.Content), &got); x != nil {
			t.Fatal(x)
		}
		if len(got.Matches) != tc.count {
			t.Fatalf("%s %+v", tc.args, got)
		}
		for i, m := range got.Matches {
			if m.Path != tc.paths[i] || m.Line < 1 {
				t.Fatal(got)
			}
		}
	}
	for _, bad := range []string{`{"pattern":"["}`, `{"pattern":""}`, `{"pattern":"needle","path":"../"}`, `{"pattern":"needle","include":"["}`, `{"pattern":"needle","include":null}`} {
		if r := e.Execute(context.Background(), "grep", json.RawMessage(bad)); !r.Error {
			t.Fatalf("accepted %s", bad)
		}
	}
}
func TestGrepResultCapAndCancellation(t *testing.T) {
	dir := t.TempDir()
	if e := os.WriteFile(filepath.Join(dir, "many.txt"), []byte(strings.Repeat("needle\n", 150)), 0600); e != nil {
		t.Fatal(e)
	}
	e, x := New(dir, t.TempDir())
	if x != nil {
		t.Fatal(x)
	}
	r := e.Execute(context.Background(), "grep", json.RawMessage(`{"pattern":"needle"}`))
	if r.Error {
		t.Fatal(r)
	}
	var got struct {
		Matches   []any
		Truncated bool
	}
	json.Unmarshal([]byte(r.Content), &got)
	if len(got.Matches) != 100 || !got.Truncated {
		t.Fatal(got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if r = e.Execute(ctx, "grep", json.RawMessage(`{"pattern":"needle"}`)); !r.Error {
		t.Fatal("cancelled grep succeeded")
	}
}

func TestGrepRespectsIgnoreFiles(t *testing.T) {
	dir := t.TempDir()
	for p, s := range map[string]string{
		".gitignore": "dist/\n*.log\n", "keep.go": "needle\n", "app.log": "needle\n",
		"dist/out.js": "needle\n", "src/.ignore": "gen.go\n", "src/gen.go": "needle\n", "src/real.go": "needle\n",
	} {
		os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0700)
		os.WriteFile(filepath.Join(dir, p), []byte(s), 0600)
	}
	e, _ := New(dir, t.TempDir())
	paths := func(args string) []string {
		r := e.Execute(context.Background(), "grep", json.RawMessage(args))
		if r.Error {
			t.Fatal(r.Content)
		}
		var got struct{ Matches []struct{ Path string } }
		json.Unmarshal([]byte(r.Content), &got)
		var out []string
		for _, m := range got.Matches {
			out = append(out, m.Path)
		}
		return out
	}
	if got := strings.Join(paths(`{"pattern":"needle"}`), ","); got != "keep.go,src/real.go" {
		t.Fatalf("ignored files searched: %s", got)
	}
	// An explicitly requested ignored directory is still searched.
	if got := strings.Join(paths(`{"pattern":"needle","path":"dist"}`), ","); got != "dist/out.js" {
		t.Fatalf("explicit path not searched: %s", got)
	}
}
