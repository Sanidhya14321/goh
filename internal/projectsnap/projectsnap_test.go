package projectsnap

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, name, body string) {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		return "<missing>"
	}
	return string(b)
}

func TestTrackDiffRestoreLeavesUserRepoAlone(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	ctx := context.Background()
	project := t.TempDir()
	git := func(a ...string) string {
		cmd := exec.Command("git", a...)
		cmd.Dir = project
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(a, err, string(out))
		}
		return string(out)
	}
	git("init", "-q")
	write(t, project, ".gitignore", "build/\n")
	write(t, project, "a.txt", "one\n")
	write(t, project, "src/b.txt", "bee\n")
	git("add", ".")
	git("commit", "-qm", "base")
	userHead := git("rev-parse", "HEAD")
	write(t, project, "a.txt", "one-edited-but-uncommitted\n")

	s, err := Open(t.TempDir(), project)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.Track(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// What a shell command might do: edit, delete, create, build output.
	write(t, project, "a.txt", "changed by shell\n")
	os.Remove(filepath.Join(project, "src/b.txt"))
	write(t, project, "new/c.txt", "created\n")
	write(t, project, "build/out.bin", "ignored artifact\n")
	second, err := s.Track(ctx)
	if err != nil {
		t.Fatal(err)
	}
	changes, err := s.Diff(ctx, first, second)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, c := range changes {
		got = append(got, c.Status+" "+c.Path)
	}
	if strings.Join(got, ",") != "M a.txt,A new/c.txt,D src/b.txt" {
		t.Fatalf("diff %v", got)
	}
	if patch, _, err := s.Patch(ctx, first, second, 1<<20); err != nil || !strings.Contains(patch, "+changed by shell") {
		t.Fatal(patch, err)
	}

	before, _, err := s.Restore(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if read(t, project, "a.txt") != "one-edited-but-uncommitted\n" || read(t, project, "src/b.txt") != "bee\n" || read(t, project, "new/c.txt") != "<missing>" {
		t.Fatal("restore did not return to the first snapshot")
	}
	if _, err := os.Stat(filepath.Join(project, "new")); err == nil {
		t.Fatal("empty directory left behind")
	}
	if read(t, project, "build/out.bin") != "ignored artifact\n" {
		t.Fatal("ignored file was touched")
	}
	// The restore is undoable.
	if _, _, err := s.Restore(ctx, before); err != nil || read(t, project, "a.txt") != "changed by shell\n" || read(t, project, "new/c.txt") != "created\n" {
		t.Fatal("redo failed", err)
	}
	// The user's repository is untouched: same HEAD, same staged state.
	if git("rev-parse", "HEAD") != userHead || strings.TrimSpace(git("diff", "--cached", "--name-only")) != "" {
		t.Fatal("user repository index or HEAD changed")
	}
	if _, _, err := s.Restore(ctx, "not-a-tree"); err == nil {
		t.Fatal("accepted bad id")
	}
	if _, _, err := s.Restore(ctx, strings.Repeat("a", 40)); err == nil {
		t.Fatal("accepted unknown tree")
	}
	if _, err := Open(filepath.Join(project, "inside"), project); err == nil {
		t.Fatal("allowed storage inside the project")
	}
}

func TestRestoreRefusesToDeleteUnrecordedFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	ctx := context.Background()
	project := t.TempDir()
	write(t, project, ".gitignore", "*.local\n")
	write(t, project, "cfg", "old\n")
	s, err := Open(t.TempDir(), project)
	if err != nil {
		t.Fatal(err)
	}
	old, err := s.Track(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(project, "cfg")); err != nil {
		t.Fatal(err)
	}
	write(t, project, "cfg/tracked.txt", "t\n")
	write(t, project, "cfg/secret.local", "keep me\n")
	if _, _, err := s.Restore(ctx, old); err == nil || !strings.Contains(err.Error(), "secret.local") {
		t.Fatalf("restore err = %v, want refusal naming the ignored file", err)
	}
	if got := read(t, project, "cfg/secret.local"); got != "keep me\n" {
		t.Fatalf("ignored file = %q", got)
	}
	// Without the ignored file, the restore goes ahead.
	if err := os.Remove(filepath.Join(project, "cfg/secret.local")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Restore(ctx, old); err != nil {
		t.Fatal(err)
	}
	if got := read(t, project, "cfg"); got != "old\n" {
		t.Fatalf("cfg = %q", got)
	}
}

func TestLinkedParentDetectsSymlinkedFolders(t *testing.T) {
	project, outside := t.TempDir(), t.TempDir()
	os.MkdirAll(filepath.Join(project, "real", "deep"), 0o755)
	if err := os.Symlink(outside, filepath.Join(project, "sub")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if !linkedParent(project, filepath.Join(project, "sub", "a.txt")) {
		t.Fatal("symlinked parent not detected")
	}
	if linkedParent(project, filepath.Join(project, "real", "deep", "a.txt")) {
		t.Fatal("plain folders reported as links")
	}
}

// Review P2: two sessions capturing and restoring one project never share a
// mutable index, so their operations cannot interleave into a wrong tree.
func TestConcurrentCaptureAndRestore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	ctx := context.Background()
	project, data := t.TempDir(), t.TempDir()
	write(t, project, "a.txt", "one\n")
	s1, err := Open(data, project)
	if err != nil {
		t.Fatal(err)
	}
	s2, _ := Open(data, project)
	first, err := s1.Track(ctx)
	if err != nil {
		t.Fatal(err)
	}
	write(t, project, "b.txt", "two\n")
	errs := make(chan error, 20)
	for i := 0; i < 10; i++ {
		go func() { _, err := s1.Track(ctx); errs <- err }()
		go func() { _, err := s2.Track(ctx); errs <- err }()
	}
	for i := 0; i < 20; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := s2.Restore(ctx, first); err != nil {
		t.Fatal(err)
	}
	if read(t, project, "b.txt") != "<missing>" || read(t, project, "a.txt") != "one\n" {
		t.Fatal("restore did not reach the first snapshot")
	}
	if again, _ := s1.Track(ctx); again != first {
		t.Fatalf("tree after restore %s, want %s", again, first)
	}
}
