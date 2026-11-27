package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func gitRepo(t *testing.T) (git, root string) {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	root = filepath.Join(t.TempDir(), "repo")
	os.MkdirAll(root, 0o755)
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.name", "t"}, {"config", "user.email", "t@example.com"}, {"config", "commit.gpgsign", "false"}} {
		if out, err := exec.Command(git, append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	os.WriteFile(filepath.Join(root, "calc.txt"), []byte("1+1=3\n"), 0o644)
	os.WriteFile(filepath.Join(root, "keep.txt"), []byte("keep\n"), 0o644)
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte("build/\n"), 0o644)
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-qm", "init"}} {
		if out, err := exec.Command(git, append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	root, _ = filepath.EvalSymlinks(root)
	t.Cleanup(func() { os.RemoveAll(root + "-watch") })
	return git, root
}

func TestWatchFixTreeAndPatch(t *testing.T) {
	git, root := gitRepo(t)
	// Uncommitted, deleted, untracked and ignored files in the working copy.
	os.WriteFile(filepath.Join(root, "calc.txt"), []byte("1+1=3\n2+2=5\n"), 0o644)
	os.Remove(filepath.Join(root, "keep.txt"))
	os.MkdirAll(filepath.Join(root, "new"), 0o755)
	os.WriteFile(filepath.Join(root, "new", "notes.txt"), []byte("draft\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "build"), 0o755)
	os.WriteFile(filepath.Join(root, "build", "out.bin"), []byte("x"), 0o644)

	dir, base, err := fixTree(git, root)
	if err != nil {
		t.Fatal(err)
	}
	read := func(p string) string { b, _ := os.ReadFile(filepath.Join(dir, p)); return string(b) }
	if read("calc.txt") != "1+1=3\n2+2=5\n" || read("new/notes.txt") != "draft\n" {
		t.Fatalf("working copy not copied: %q %q", read("calc.txt"), read("new/notes.txt"))
	}
	if _, err := os.Stat(filepath.Join(dir, "keep.txt")); err == nil {
		t.Fatal("deleted file came back")
	}
	if _, err := os.Stat(filepath.Join(dir, "build")); err == nil {
		t.Fatal("ignored files were copied")
	}
	if patch, _, err := fixPatch(git, dir, base); err != nil || len(patch) != 0 {
		t.Fatalf("untouched copy has a patch: %q %v", patch, err)
	}

	// The "fix": edit a file and add one.
	os.WriteFile(filepath.Join(dir, "calc.txt"), []byte("1+1=2\n2+2=4\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "new", "test.txt"), []byte("ok\n"), 0o644)
	patch, stat, err := fixPatch(git, dir, base)
	if err != nil || !strings.Contains(stat, "calc.txt") || !strings.Contains(stat, "new/test.txt") {
		t.Fatalf("%v\n%s", err, stat)
	}
	before, _ := os.ReadFile(filepath.Join(root, "calc.txt"))
	if string(before) != "1+1=3\n2+2=5\n" {
		t.Fatal("the working copy changed before applying")
	}
	w := &watcher{git: git, root: root, out: &bytes.Buffer{}}
	file, err := w.savePatch(patch)
	if err != nil || !strings.Contains(file, filepath.Join(".git", "goh-watch")) {
		t.Fatal(file, err)
	}
	w.patch = file
	if !w.apply() {
		t.Fatal(w.out)
	}
	after, _ := os.ReadFile(filepath.Join(root, "calc.txt"))
	added, _ := os.ReadFile(filepath.Join(root, "new", "test.txt"))
	if string(after) != "1+1=2\n2+2=4\n" || string(added) != "ok\n" {
		t.Fatalf("%q %q", after, added)
	}
	// A second copy starts over from the new working copy.
	if _, base2, err := fixTree(git, root); err != nil || base2 == base {
		t.Fatal(err)
	}
	if strings.TrimSpace(readFile(t, filepath.Join(dir, "calc.txt"))) != "1+1=2\n2+2=4" {
		t.Fatal("second copy is stale")
	}
	if _, err := os.Stat(filepath.Join(dir, "new", "test.txt")); err != nil {
		t.Fatal("second copy misses the applied file")
	}

	// An applied patch whose files changed since is refused, not forced.
	w.patch = file
	if w.apply() {
		t.Fatal("re-applied a stale patch")
	}
}

func readFile(t *testing.T, p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestWatchRefusesForeignFolder(t *testing.T) {
	git, root := gitRepo(t)
	os.MkdirAll(root+"-watch", 0o755)
	if _, _, err := fixTree(git, root); err == nil || !strings.Contains(err.Error(), "not a worktree") {
		t.Fatal(err)
	}
}

func TestWatchFingerprint(t *testing.T) {
	git, root := gitRepo(t)
	w := &watcher{git: git, root: root}
	a := w.fingerprint()
	os.WriteFile(filepath.Join(root, "calc.txt"), []byte("1+1=2\n"), 0o644)
	b := w.fingerprint()
	// A second edit leaves git status unchanged; size or time changes.
	os.WriteFile(filepath.Join(root, "calc.txt"), []byte("1+1=22\n"), 0o644)
	c := w.fingerprint()
	os.WriteFile(filepath.Join(root, "build.log"), nil, 0o644)
	d := w.fingerprint()
	os.MkdirAll(filepath.Join(root, "build"), 0o755)
	os.WriteFile(filepath.Join(root, "build", "x"), nil, 0o644)
	e := w.fingerprint()
	if a == b || b == c || c == d || d != e {
		t.Fatalf("%s %s %s %s %s", a, b, c, d, e)
	}
}

func TestWatchTests(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	dir := t.TempDir()
	ok := runTests(context.Background(), dir, "echo fine", time.Minute)
	bad := runTests(context.Background(), dir, "echo broken; exit 3", time.Minute)
	slow := runTests(context.Background(), dir, "sleep 5", 200*time.Millisecond)
	if !ok.passed || bad.passed || !strings.Contains(bad.output, "broken") || slow.passed || !strings.Contains(slow.err, "timed out") || slow.seconds > 3 {
		t.Fatalf("%+v %+v %+v", ok, bad, slow)
	}
	p := fixPrompt("make test", bad)
	if !strings.Contains(p, "`make test` fails") || !strings.Contains(p, "broken") {
		t.Fatal(p)
	}
	var tb tailBuffer
	tb.limit = 10
	tb.Write([]byte("line1\nline2\nline3\n"))
	if s := tb.String(); s != "…\nline3\n" {
		t.Fatalf("%q", s)
	}
}

func TestWatchDetectAndArguments(t *testing.T) {
	dir := t.TempDir()
	if detectTestCommand(dir) != "" {
		t.Fatal("detected in an empty folder")
	}
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"jest"}}`), 0o644)
	if detectTestCommand(dir) != "npm test" {
		t.Fatal(detectTestCommand(dir))
	}
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644)
	if detectTestCommand(dir) != "go test ./..." {
		t.Fatal(detectTestCommand(dir))
	}
	for _, bad := range [][]string{{"extra"}, {"--quiet", "1ms"}, {"--max-fixes", "0"}, {"--", "--project", "/x"}, {"--", "--prompt=x"}} {
		if err := runWatch(bad, nil, &bytes.Buffer{}); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
	if !hasAccessFlag([]string{"--access=auto"}) || hasAccessFlag([]string{"--allow", "shell"}) {
		t.Fatal("access flag detection")
	}
}
