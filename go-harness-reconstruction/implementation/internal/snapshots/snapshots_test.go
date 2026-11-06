package snapshots

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func setup(t *testing.T) (*Manager, string, string) {
	t.Helper()
	base := t.TempDir()
	project := filepath.Join(base, "project")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(base, "snapshots")
	m, err := Open(project, directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	return m, project, directory
}
func put(t *testing.T, root, path, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0640); err != nil {
		t.Fatal(err)
	}
}
func get(t *testing.T, root, path string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func capture(t *testing.T, m *Manager, paths ...string) string {
	t.Helper()
	id, err := m.Capture(context.Background(), paths)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func seal(t *testing.T, m *Manager, id string) Record {
	t.Helper()
	r, err := m.Seal(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestSnapshotUndoRedoDurableAndIndexUnchanged(t *testing.T) {
	m, p, d := setup(t)
	put(t, p, "old", "one\n")
	put(t, p, "deleted", "gone\n")
	git := func(args ...string) []byte {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = p
		b, e := c.CombinedOutput()
		if e != nil {
			t.Fatalf("git: %v: %s", e, b)
		}
		return b
	}
	git("init", "-q")
	git("add", "--", "old", "deleted")
	index := git("ls-files", "--stage")
	id := capture(t, m, "old", "created", "deleted")
	put(t, p, "old", "two\n")
	put(t, p, "created", "new\n")
	if err := os.Remove(filepath.Join(p, "deleted")); err != nil {
		t.Fatal(err)
	}
	r := seal(t, m, id)
	if r.Status != "recorded" || len(r.After) != 3 {
		t.Fatalf("record: %+v", r)
	}
	changes, err := m.Diff(id)
	if err != nil || len(changes) != 3 {
		t.Fatalf("diff: %+v %v", changes, err)
	}
	if !strings.Contains(changes[2].Preview, "-one\n+two\n") {
		t.Fatalf("preview: %+v", changes)
	}
	put(t, p, "unrelated", "stay")
	r, err = m.Undo(context.Background(), id)
	if err != nil || r.Status != "undone" {
		t.Fatalf("undo: %+v %v", r, err)
	}
	if get(t, p, "old") != "one\n" || get(t, p, "deleted") != "gone\n" || get(t, p, "unrelated") != "stay" {
		t.Fatal("undo changed unexpected bytes")
	}
	if _, err = os.Stat(filepath.Join(p, "created")); !os.IsNotExist(err) {
		t.Fatal("created file not removed")
	}
	st, _ := os.Stat(filepath.Join(p, "old"))
	if st.Mode().Perm() != 0640 {
		t.Fatalf("mode: %o", st.Mode().Perm())
	}
	m.Close()
	m, err = Open(p, d)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	r, err = m.Redo(context.Background(), id)
	if err != nil || r.Status != "recorded" {
		t.Fatalf("redo: %+v %v", r, err)
	}
	if get(t, p, "old") != "two\n" || get(t, p, "created") != "new\n" {
		t.Fatal("redo lost bytes")
	}
	if _, err = os.Stat(filepath.Join(p, "deleted")); !os.IsNotExist(err) {
		t.Fatal("deleted file restored on redo")
	}
	if string(index) != string(git("ls-files", "--stage")) {
		t.Fatal("user index modified")
	}
}

func TestSnapshotPrevalidationConflictsAndModes(t *testing.T) {
	m, p, _ := setup(t)
	put(t, p, "a", "A")
	put(t, p, "b", "B")
	id := capture(t, m, "a", "b")
	put(t, p, "a", "a")
	put(t, p, "b", "b")
	seal(t, m, id)
	put(t, p, "b", "external")
	if _, e := m.Undo(context.Background(), id); !errors.Is(e, ErrConflict) {
		t.Fatalf("expected conflict: %v", e)
	}
	if get(t, p, "a") != "a" {
		t.Fatal("published before complete prevalidation")
	}
	put(t, p, "b", "b")
	if e := os.Chmod(filepath.Join(p, "b"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := m.Undo(context.Background(), id); !errors.Is(e, ErrConflict) {
		t.Fatalf("expected mode conflict: %v", e)
	}
	r, _ := m.Read(id)
	if r.Status != "recorded" {
		t.Fatalf("prevalidation changed status: %s", r.Status)
	}
}

// This context cancels monotonically at a deterministic admission checkpoint,
// after one file has been published and its progress synchronized.
type countingContext struct {
	n, at int
	done  chan struct{}
}

func (c *countingContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *countingContext) Done() <-chan struct{}       { return c.done }
func (c *countingContext) Value(any) any               { return nil }
func (c *countingContext) Err() error {
	c.n++
	if c.n >= c.at {
		select {
		case <-c.done:
		default:
			close(c.done)
		}
		return context.Canceled
	}
	return nil
}

func TestSnapshotPartialApplicationNeverReplayed(t *testing.T) {
	m, p, d := setup(t)
	put(t, p, "a", "A")
	put(t, p, "b", "B")
	id := capture(t, m, "a", "b")
	put(t, p, "a", "a")
	put(t, p, "b", "b")
	seal(t, m, id)
	c := &countingContext{at: 6, done: make(chan struct{})}
	r, err := m.Undo(c, id)
	if !errors.Is(err, ErrUncertain) || r.Status != "applying_undo" || len(r.Applied) != 1 || r.Applied[0] != "a" {
		t.Fatalf("expected partial effect: %+v %v calls=%d", r, err, c.n)
	}
	if get(t, p, "a") != "A" || get(t, p, "b") != "b" {
		t.Fatal("unexpected partial state")
	}
	m.Close()
	m, err = Open(p, d)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if _, err = m.Capture(context.Background(), []string{"a"}); !errors.Is(err, ErrPending) {
		t.Fatalf("capture bypassed pending: %v", err)
	}
	if _, err = m.Undo(context.Background(), id); !errors.Is(err, ErrPending) {
		t.Fatalf("undo replayed partial: %v", err)
	}
	if _, err = m.Redo(context.Background(), id); !errors.Is(err, ErrPending) {
		t.Fatalf("redo replayed partial: %v", err)
	}
	if _, err = m.Seal(context.Background(), id); err == nil {
		t.Fatal("sealed interrupted apply")
	}
	if err = m.Acknowledge(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	r, _ = m.Read(id)
	if r.Status != "acknowledged" || len(r.Applied) != 1 {
		t.Fatalf("lost uncertainty evidence: %+v", r)
	}
	if get(t, p, "a") != "A" || get(t, p, "b") != "b" {
		t.Fatal("recovery replayed effects")
	}
	if _, err = m.Undo(context.Background(), id); err == nil {
		t.Fatal("acknowledged record eligible for undo")
	}
}

func TestSnapshotCapturedRecoveryAndAbort(t *testing.T) {
	m, p, d := setup(t)
	put(t, p, "a", "A")
	id := capture(t, m, "a")
	m.Close()
	var err error
	m, err = Open(p, d)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	pending, err := m.Pending()
	if err != nil || len(pending) != 1 || pending[0].ID != id {
		t.Fatalf("pending: %+v %v", pending, err)
	}
	put(t, p, "a", "different")
	if err = m.Abort(context.Background(), id); !errors.Is(err, ErrConflict) {
		t.Fatalf("abort lost mutation: %v", err)
	}
	put(t, p, "a", "A")
	if err = m.Abort(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if _, err = m.Capture(context.Background(), []string{"a"}); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotPathsAndSpecialFiles(t *testing.T) {
	m, p, d := setup(t)
	put(t, p, "a", "A")
	outside := filepath.Join(t.TempDir(), "outside")
	put(t, filepath.Dir(outside), "outside", "safe")
	if err := os.Symlink(outside, filepath.Join(p, "link")); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(p, "pipe"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../outside", outside, ".git/index", "link", "pipe", "missing/child", ".", ""} {
		if _, err := m.Capture(context.Background(), []string{path}); err == nil {
			t.Errorf("accepted %q", path)
		}
	}
	if _, err := Open(p, filepath.Join(p, "snapshots")); err == nil {
		t.Fatal("allowed in-project store")
	}
	if _, err := Open(p, filepath.Dir(p)); err == nil {
		t.Fatal("allowed project ancestor store")
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(p, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(p, filepath.Join(alias, "new-store")); err == nil {
		t.Fatal("allowed aliased in-project store")
	}
	if _, err := os.Stat(filepath.Join(p, "new-store")); !os.IsNotExist(err) {
		t.Fatal("created aliased directory before validation")
	}
	if _, err := Open(p, d); err == nil {
		t.Fatal("allowed second writer")
	}
	if get(t, filepath.Dir(outside), "outside") != "safe" {
		t.Fatal("outside content changed")
	}
}

func TestSnapshotIntegrityAndBounds(t *testing.T) {
	m, p, d := setup(t)
	put(t, p, "a", "A")
	id := capture(t, m, "a")
	put(t, p, "a", "a")
	r := seal(t, m, id)
	if err := os.WriteFile(filepath.Join(d, "blobs", r.Before[0].SHA256), []byte("X"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Undo(context.Background(), id); err == nil {
		t.Fatal("accepted corrupt blob")
	}
	if get(t, p, "a") != "a" {
		t.Fatal("published corrupt blob")
	}
	put(t, p, "big", strings.Repeat("x", MaxFile+1))
	if _, err := m.Capture(context.Background(), []string{"big"}); err == nil {
		t.Fatal("oversized capture")
	}
	paths := make([]string, MaxPaths+1)
	for i := range paths {
		paths[i] = "a"
	}
	if _, err := m.Capture(context.Background(), paths); err == nil {
		t.Fatal("oversized paths")
	}
	for _, id := range []string{"../outside", "", "ffffffffffffffffffffffff.json"} {
		if _, err := m.Read(id); err == nil {
			t.Fatalf("invalid ID %q", id)
		}
	}
}

func TestSnapshotBinaryAndBoundedDiff(t *testing.T) {
	m, p, _ := setup(t)
	put(t, p, "binary", "a\x00b")
	put(t, p, "text", strings.Repeat("😀", 20000))
	id := capture(t, m, "binary", "text")
	put(t, p, "binary", "c\x00d")
	put(t, p, "text", strings.Repeat("λ", 50000))
	seal(t, m, id)
	changes, err := m.Diff(id)
	if err != nil || len(changes) != 2 {
		t.Fatalf("diff: %+v %v", changes, err)
	}
	if !changes[0].Binary || changes[0].Preview != "" || !changes[1].Truncated || len(changes[1].Preview) > 32<<10 {
		t.Fatal("invalid bounded diff")
	}
	if _, err = m.Undo(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if get(t, p, "binary") != "a\x00b" {
		t.Fatal("binary bytes not restored")
	}
}

func TestSnapshotSealFailureRetainsPending(t *testing.T) {
	m, p, _ := setup(t)
	put(t, p, "a", "A")
	id := capture(t, m, "a")
	if err := os.Remove(filepath.Join(p, "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("nowhere", filepath.Join(p, "a")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Seal(context.Background(), id); err == nil {
		t.Fatal("sealed symlink")
	}
	pending, _ := m.Pending()
	if len(pending) != 1 || pending[0].Status != "captured" {
		t.Fatal("failed seal erased pending")
	}
}
