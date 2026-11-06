package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func engine(t *testing.T) (*Engine, string) {
	t.Helper()
	root := t.TempDir()
	e, err := New(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return e, root
}
func call(t *testing.T, e *Engine, name string, a any) Result {
	t.Helper()
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	return e.Execute(context.Background(), name, b)
}
func object(t *testing.T, r Result) map[string]any {
	t.Helper()
	if r.Error {
		t.Fatal(r.Content)
	}
	v := map[string]any{}
	if err := json.Unmarshal([]byte(r.Content), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func TestEditPreservesBytesPermissionsAndRejectsStale(t *testing.T) {
	e, root := engine(t)
	before := "first\r\n日本語 😀\r\nlast\r\n"
	p := filepath.Join(root, "note.txt")
	if err := os.WriteFile(p, []byte(before), 0751); err != nil {
		t.Fatal(err)
	}
	got := object(t, call(t, e, "read_file", map[string]any{"path": "note.txt", "start_line": 2, "end_line": 2}))
	if got["content"] != "日本語 😀\r" || got["sha256"] != hash([]byte(before)) {
		t.Fatal(got)
	}
	change := map[string]any{"path": "note.txt", "old_text": "日本語 😀", "new_text": "changed", "expected_sha256": got["sha256"]}
	preview, side, err := e.Prepare("edit_file", mustJSON(t, change))
	if err != nil || !side || !strings.Contains(preview, p) || !strings.Contains(preview, "+changed\r") {
		t.Fatal(preview, side, err)
	}
	object(t, call(t, e, "edit_file", change))
	after, err := os.ReadFile(p)
	if err != nil || string(after) != "first\r\nchanged\r\nlast\r\n" {
		t.Fatal(string(after), err)
	}
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0751 {
		t.Fatal(st.Mode())
	}
	if !call(t, e, "edit_file", change).Error {
		t.Fatal("stale edit accepted")
	}
	still, _ := os.ReadFile(p)
	if string(still) != string(after) {
		t.Fatal("stale edit mutated file")
	}
}
func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestAmbiguityAndCreate(t *testing.T) {
	e, root := engine(t)
	p := filepath.Join(root, "a")
	os.WriteFile(p, []byte("x x"), 0644)
	a := map[string]any{"path": "a", "old_text": "x", "new_text": "y", "expected_sha256": hash([]byte("x x"))}
	if !call(t, e, "edit_file", a).Error {
		t.Fatal("ambiguous edit accepted")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "x x" {
		t.Fatal("changed on failure")
	}
	a["replace_all"] = true
	object(t, call(t, e, "edit_file", a))
	b, _ = os.ReadFile(p)
	if string(b) != "y y" {
		t.Fatal(string(b))
	}
	create := map[string]any{"path": "new", "content": "complete"}
	object(t, call(t, e, "create_file", create))
	if !call(t, e, "create_file", create).Error {
		t.Fatal("overwrote existing file")
	}
	entries, _ := os.ReadDir(root)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".harness-") {
			t.Fatal("staging file leaked")
		}
	}
}
func TestPrepareDoesNotChangeFilesAndExecuteRechecks(t *testing.T) {
	e, root := engine(t)
	os.WriteFile(filepath.Join(root, "a"), []byte("old"), 0644)
	a := mustJSON(t, map[string]any{"path": "a", "content": "new", "expected_sha256": hash([]byte("old"))})
	if _, _, err := e.Prepare("write_file", a); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "a"))
	if string(b) != "old" {
		t.Fatal("prepare changed file")
	}
	os.WriteFile(filepath.Join(root, "a"), []byte("external"), 0644)
	if !e.Execute(context.Background(), "write_file", a).Error {
		t.Fatal("missed intervening change")
	}
}
func TestPathAndSpecialFileBoundaries(t *testing.T) {
	e, root := engine(t)
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "secret"), []byte("secret"), 0600)
	os.Symlink(outside, filepath.Join(root, "escape"))
	for _, p := range []string{"../secret", filepath.Join(outside, "secret"), "escape/secret"} {
		if !call(t, e, "read_file", map[string]any{"path": p}).Error {
			t.Fatalf("accepted %s", p)
		}
	}
	os.WriteFile(filepath.Join(root, "binary"), []byte{0, 'x'}, 0644)
	if !call(t, e, "read_file", map[string]any{"path": "binary"}).Error {
		t.Fatal("accepted binary")
	}
	if err := syscall.Mkfifo(filepath.Join(root, "fifo"), 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan Result, 1)
	go func() { done <- call(t, e, "read_file", map[string]any{"path": "fifo"}) }()
	select {
	case r := <-done:
		if !r.Error {
			t.Fatal("accepted FIFO")
		}
	case <-time.After(time.Second):
		t.Fatal("FIFO blocked")
	}
}
func TestInventorySearchLimitsAndCancellation(t *testing.T) {
	e, root := engine(t)
	os.Mkdir(filepath.Join(root, "node_modules"), 0755)
	os.WriteFile(filepath.Join(root, "node_modules", "hidden"), []byte("needle"), 0644)
	os.WriteFile(filepath.Join(root, "a"), []byte("needle\nneedle\n"), 0644)
	os.WriteFile(filepath.Join(root, "b"), []byte("other"), 0644)
	v := object(t, call(t, e, "list_files", map[string]any{"limit": 1}))
	if v["truncated"] != true || len(v["files"].([]any)) != 1 {
		t.Fatal(v)
	}
	v = object(t, call(t, e, "search", map[string]any{"pattern": "needle", "limit": 1}))
	if v["truncated"] != true || len(v["matches"].([]any)) != 1 {
		t.Fatal(v)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !e.Execute(ctx, "list_files", []byte(`{}`)).Error {
		t.Fatal("ignored cancellation")
	}
	for _, raw := range []string{`null`, `[]`, `{"path":null}`, `{"path":"a","extra":true}`, `{"path":"a","start_line":1.5}`, `{"path":"a"} {}`} {
		if !e.Execute(context.Background(), "read_file", []byte(raw)).Error {
			t.Fatal("accepted", raw)
		}
	}
}
func TestShellExitAndBoundedOutput(t *testing.T) {
	e, _ := engine(t)
	v := object(t, call(t, e, "shell", map[string]any{"command": "printf out; printf err >&2"}))
	if v["exit_code"] != float64(0) || !strings.Contains(v["output"].(string), "out") || !strings.Contains(v["output"].(string), "err") {
		t.Fatal(v)
	}
	st, err := os.Stat(v["artifact"].(string))
	if err != nil || st.Mode().Perm() != 0600 {
		t.Fatal(st, err)
	}
	r := call(t, e, "shell", map[string]any{"command": "printf fail >&2; exit 7"})
	if !r.Error {
		t.Fatal("nonzero not error")
	}
	var failed map[string]any
	json.Unmarshal([]byte(r.Content), &failed)
	if failed["exit_code"] != float64(7) {
		t.Fatal(failed)
	}
	v = object(t, call(t, e, "shell", map[string]any{"command": "head -c 2097152 /dev/zero"}))
	if v["output_bytes"] != float64(2097152) || v["retained_bytes"] != float64(maxArtifact) || v["truncated"] != true {
		t.Fatal(v)
	}
	if len(v["output"].(string)) > MaxInline/2 {
		t.Fatal("unbounded preview")
	}
}
func TestShellTimeoutAndChildCleanup(t *testing.T) {
	e, root := engine(t)
	start := time.Now()
	r := call(t, e, "shell", map[string]any{"command": "trap '' TERM; sleep 30 & wait", "timeout_seconds": 1})
	elapsed := time.Since(start)
	if !r.Error || elapsed > 5*time.Second {
		t.Fatal(r, elapsed)
	}
	var v map[string]any
	json.Unmarshal([]byte(r.Content), &v)
	if v["status"] != "timed_out" {
		t.Fatal(v)
	}
	marker := filepath.Join(root, "escaped")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan Result, 1)
	go func() {
		done <- e.Execute(ctx, "shell", mustJSON(t, map[string]any{"command": fmt.Sprintf("(sleep 1; touch %s) & wait", marker)}))
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case r = <-done:
		if !r.Error {
			t.Fatal("cancel success")
		}
	case <-time.After(4 * time.Second):
		t.Fatal("cancel did not settle")
	}
	time.Sleep(time.Second)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("child survived cancellation")
	}
}

func FuzzToolArguments(f *testing.F) {
	for _, seed := range []string{`{}`, `null`, `{"path":"a","old_text":"x","new_text":"y","expected_sha256":"abc"}`, `{"path":"../escape"}`, `{"path":null}`, `{"replace_all":true}`, "\xff"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		_, err := decode("edit_file", json.RawMessage(raw))
		if err == nil && (!json.Valid([]byte(raw)) || len(raw) > 1<<20) {
			t.Fatal("accepted invalid argument envelope")
		}
	})
}

// This context models cancellation immediately after Execute's first context
// check, while its admission/validation work is still in progress.
type cancelAfterFirstCheck struct {
	context.Context
	cancel context.CancelFunc
	once   sync.Once
}

func (c *cancelAfterFirstCheck) Err() error {
	previous := c.Context.Err()
	c.once.Do(c.cancel)
	return previous
}
func TestCancellationDuringAdmissionPreventsNewSideEffects(t *testing.T) {
	for _, name := range []string{"create_file", "shell"} {
		t.Run(name, func(t *testing.T) {
			e, root := engine(t)
			base, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx := &cancelAfterFirstCheck{Context: base, cancel: cancel}
			args := map[string]any{"path": "cancelled-effect", "content": "effect"}
			if name == "shell" {
				args = map[string]any{"command": "touch cancelled-effect"}
			}
			res := e.Execute(ctx, name, mustJSON(t, args))
			if !res.Error {
				t.Fatal("cancelled operation reported success")
			}
			if _, err := os.Stat(filepath.Join(root, "cancelled-effect")); !os.IsNotExist(err) {
				t.Fatal("side effect started after cancellation was observable")
			}
		})
	}
}
