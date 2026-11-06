package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func globPaths(t *testing.T, r Result) ([]string, map[string]any) {
	t.Helper()
	v := object(t, r)
	files := []string{}
	for _, p := range v["files"].([]any) {
		files = append(files, p.(string))
	}
	return files, v
}
func TestGlobPatternsAndDirectoryScope(t *testing.T) {
	e, root := engine(t)
	names := []string{"root.go", "src/a.go", "src/nested/z.ts", ".hidden.go", "src/.secret.ts", "ignored.go", "node_modules/vendor.go", ".git/config", ".harness/session", "literal{brace}.txt", "literal[bang.txt"}
	for _, name := range names {
		p := filepath.Join(root, name)
		os.MkdirAll(filepath.Dir(p), 0755)
		if err := os.WriteFile(p, []byte("fixture"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored.go\n"), 0644)
	cases := []struct {
		pattern, base string
		want          []string
	}{
		{"*.go", "", []string{".hidden.go", "ignored.go", "root.go", "src/a.go"}},
		{"**/*.go", "", []string{".hidden.go", "ignored.go", "root.go", "src/a.go"}},
		{"**/*.{go,ts}", "", []string{".hidden.go", "ignored.go", "root.go", "src/.secret.ts", "src/a.go", "src/nested/z.ts"}},
		{"**/*.{go,{ts,rs}}", "src", []string{"src/.secret.ts", "src/a.go", "src/nested/z.ts"}},
		{"src/?[!b-z].go", "", []string{}},
		{"src/[a-c].go", "", []string{"src/a.go"}},
		{"./**/*.go", "src", []string{"src/a.go"}},
		{"missing.*", "", []string{}},
		{`literal\{brace\}.txt`, "", []string{"literal{brace}.txt"}},
		{`literal\[bang.txt`, "", []string{"literal[bang.txt"}},
	}
	for _, tc := range cases {
		t.Run(tc.pattern+tc.base, func(t *testing.T) {
			a := map[string]any{"pattern": tc.pattern}
			if tc.base != "" {
				a["path"] = tc.base
			}
			preview, side, err := e.Prepare("glob", mustJSON(t, a))
			if err != nil || side || !strings.Contains(preview, root) {
				t.Fatal(preview, side, err)
			}
			got, v := globPaths(t, call(t, e, "glob", a))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatal(got, tc.want)
			}
			if v["truncated"] != false {
				t.Fatal(v)
			}
		})
	}
}
func TestGlobRejectsUnsafeBasesAndMalformedPatterns(t *testing.T) {
	e, root := engine(t)
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "secret.go"), []byte("secret"), 0600)
	os.Symlink(outside, filepath.Join(root, "outside"))
	os.WriteFile(filepath.Join(root, "file.go"), []byte("file"), 0644)
	for _, base := range []string{"../", outside, "outside", "file.go", "missing"} {
		if !call(t, e, "glob", map[string]any{"pattern": "**/*", "path": base}).Error {
			t.Fatal("accepted base", base)
		}
	}
	for _, pattern := range []string{"", "[unclosed", "{unclosed", "{a}", "a}", "../*.go", "/root/*", "a//b", "!**/*.go", strings.Repeat("x", globPatternBytes+1), strings.Repeat("{a,b}", 7)} {
		if !call(t, e, "glob", map[string]any{"pattern": pattern}).Error {
			t.Fatal("accepted malformed pattern", pattern)
		}
	}
	for _, raw := range []string{`{"pattern":null}`, `{"path":"."}`, `{"pattern":"*","limit":1}`, `{"pattern":"*","path":null}`} {
		if !e.Execute(context.Background(), "glob", json.RawMessage(raw)).Error {
			t.Fatal("accepted arguments", raw)
		}
	}
	got, _ := globPaths(t, call(t, e, "glob", map[string]any{"pattern": "**/*"}))
	if !reflect.DeepEqual(got, []string{"file.go"}) {
		t.Fatal("symlink traversed", got)
	}
}
func TestGlobResultLimitOrderingAndCancellation(t *testing.T) {
	e, root := engine(t)
	os.Mkdir(filepath.Join(root, "a"), 0755)
	os.WriteFile(filepath.Join(root, "a", "inner.go"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(root, "a.go"), []byte("x"), 0644)
	for i := 0; i < 110; i++ {
		os.WriteFile(filepath.Join(root, fmt.Sprintf("z%03d.go", i)), []byte("x"), 0644)
	}
	got, v := globPaths(t, call(t, e, "glob", map[string]any{"pattern": "**/*.go"}))
	if len(got) != 100 || got[0] != "a.go" || got[1] != "a/inner.go" || v["truncated"] != true {
		t.Fatal(got, v)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !e.Execute(ctx, "glob", json.RawMessage(`{"pattern":"**/*"}`)).Error {
		t.Fatal("ignored cancellation")
	}
}
func TestGlobExactlyOneHundredIsNotTruncated(t *testing.T) {
	e, root := engine(t)
	for i := 0; i < 100; i++ {
		os.WriteFile(filepath.Join(root, fmt.Sprintf("%03d.go", i)), []byte("x"), 0644)
	}
	got, v := globPaths(t, call(t, e, "glob", map[string]any{"pattern": "*.go"}))
	if len(got) != 100 || v["truncated"] != false {
		t.Fatal(v)
	}
}
func FuzzGlobPattern(f *testing.F) {
	for _, s := range []string{"*.go", "**/*.{go,ts}", "{a,{b,c}}/*.txt", "[!a-z]", "[", "../*", "a/**/b/**/c"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, pattern string) {
		p, err := compileGlob(pattern)
		if err == nil {
			_ = p.matches("src/nested/file.go")
		}
	})
}

func FuzzGlobPatterns(f *testing.F) {
	for _, seed := range []string{"**/*.{go,ts}", "*.go", "[unclosed", "{a,{b,c}}/**/*.go", "**/**/x", "../*", "[!a-z]*"} {
		f.Add(seed, "src/α.go")
	}
	f.Fuzz(func(t *testing.T, pattern, candidate string) {
		if len(pattern) > 4096 || len(candidate) > 4096 {
			t.Skip()
		}
		compiled, e := compileGlob(pattern)
		if e == nil {
			compiled.matches(candidate)
		}
	})
}
