package runtime

import (
	"context"
	"encoding/json"
	"example.com/independent-harness/internal/testutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/snapshots"
)

func formatterFixture(t *testing.T, root, body string) string {
	t.Helper()
	file := filepath.Join(root, "fixture formatter")
	if err := os.WriteFile(file, []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return file
}
func TestPostEditFormatterAdmissionAndFinalSnapshot(t *testing.T) {
	testutil.RequireUnixShell(t)
	for _, mode := range []string{"allow", "denied", "headless", "failed"} {
		t.Run(mode, func(t *testing.T) {
			a, root := testAgent(t)
			a.Config.Snapshots = true
			body := "printf 'FORMATTED\\n' > \"$1\""
			if mode == "failed" {
				body += "\nexit 7"
			}
			program := formatterFixture(t, root, body)
			a.Config.Formatters = map[string]core.FormatterConfig{"fixture": {Command: program, Args: []string{"$FILE"}, Extensions: []string{".txt"}}}
			a.Grants["create_file"] = true
			if mode == "allow" || mode == "failed" {
				a.Grants["format:fixture"] = true
			}
			if mode == "denied" {
				a.Approve = func(context.Context, core.Permission) (core.Decision, error) { return core.Decision{}, nil }
			}
			turn := 0
			var saved map[string]any
			a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
				turn++
				if turn == 1 {
					return answer(toolCall("create", "create_file", map[string]any{"path": "note.txt", "content": "raw"})), nil
				}
				saved = toolResult(t, h[len(h)-1])
				return answer(), nil
			})
			status, err := a.Run(context.Background(), "create and format")
			if status != "completed" || err != nil {
				t.Fatal(status, err)
			}
			expected := "raw"
			if mode == "allow" || mode == "failed" {
				expected = "FORMATTED\n"
			}
			b, err := os.ReadFile(filepath.Join(root, "note.txt"))
			if err != nil || string(b) != expected {
				t.Fatal(string(b), err)
			}
			if saved["snapshot_id"] == nil {
				t.Fatal(saved)
			}
			reports := saved["formatters"].([]any)
			report := reports[0].(map[string]any)
			want := map[string]string{"allow": "completed", "denied": "denied", "headless": "skipped", "failed": "failed"}[mode]
			if report["status"] != want {
				t.Fatal(report)
			}
			m, err := snapshots.Open(root, a.snapshotDirectory())
			if err != nil {
				t.Fatal(err)
			}
			defer m.Close()
			rs, err := m.List()
			if err != nil || len(rs) != 1 || rs[0].Status != "recorded" {
				t.Fatal(rs, err)
			}
			digest, err := a.Tools.FileDigest("note.txt")
			if err != nil || saved["sha256"] != digest["sha256"] || rs[0].After[0].SHA256 != digest["sha256"] {
				t.Fatal(saved, digest, rs, err)
			}
			if _, err = m.Undo(context.Background(), rs[0].ID); err != nil {
				t.Fatal(err)
			}
			if _, err = os.Stat(filepath.Join(root, "note.txt")); !os.IsNotExist(err) {
				t.Fatal("undo did not remove created file", err)
			}
			if _, err = m.Redo(context.Background(), rs[0].ID); err != nil {
				t.Fatal(err)
			}
			b, _ = os.ReadFile(filepath.Join(root, "note.txt"))
			if string(b) != expected {
				t.Fatal(string(b))
			}
		})
	}
}
func TestFormatterCancellationRetainsBothCheckpoints(t *testing.T) {
	testutil.RequireUnixShell(t)
	a, root := testAgent(t)
	a.Config.Snapshots = true
	program := formatterFixture(t, root, "printf '%s' $$ > formatter-pid\nprintf 'changed' > \"$1\"\nsleep 30")
	a.Config.Formatters = map[string]core.FormatterConfig{"fixture": {Command: program, Args: []string{"$FILE"}, Extensions: []string{".txt"}}}
	a.Grants["create_file"] = true
	a.Grants["format:fixture"] = true
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return answer(toolCall("write", "create_file", map[string]any{"path": "note.txt", "content": "raw"})), nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	var status string
	var runErr error
	go func() { status, runErr = a.Run(ctx, "cancel formatter"); close(done) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(root, "formatter-pid")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			<-done
			t.Fatal("formatter did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("formatter cleanup hung")
	}
	if status != "cancelled" || runErr == nil || len(a.Store.Pending()) != 2 {
		t.Fatal(status, runErr, a.Store.Pending())
	}
	m, err := snapshots.Open(root, a.snapshotDirectory())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	pending, err := m.Pending()
	if err != nil || len(pending) != 1 || pending[0].Status != "captured" {
		t.Fatal(pending, err)
	}
	if _, err = m.Capture(context.Background(), []string{"note.txt"}); err == nil {
		t.Fatal("uncertain checkpoint did not block")
	}
}
func TestSnapshotToolsAdmissionAndPlanBoundary(t *testing.T) {
	a, root := testAgent(t)
	a.Config.Snapshots = true
	if err := os.WriteFile(filepath.Join(root, "note"), []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := snapshots.Open(root, a.snapshotDirectory())
	if err != nil {
		t.Fatal(err)
	}
	id, err := m.Capture(context.Background(), []string{"note"})
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "note"), []byte("after"), 0600)
	if _, err = m.Seal(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	m.Close()
	for _, name := range []string{"snapshot_list", "snapshot_diff", "snapshot_undo", "snapshot_redo"} {
		args := map[string]any{"id": id}
		if name == "snapshot_list" {
			args = map[string]any{}
		}
		call := toolCall("test", name, args)
		_, side, err := a.prepare(name, json.RawMessage(call.Function.Arguments))
		if err != nil || side != (name == "snapshot_undo" || name == "snapshot_redo") {
			t.Fatal(name, side, err)
		}
	}
	a.Config.AgentMode = "plan"
	for _, name := range []string{"snapshot_undo", "snapshot_redo", "format:fixture"} {
		if a.checkMode(name) == nil {
			t.Fatal("plan allowed", name)
		}
	}
	for _, name := range []string{"snapshot_list", "snapshot_diff", "formatter_status"} {
		if a.checkMode(name) != nil {
			t.Fatal("plan denied read", name)
		}
	}
	if r := a.execute(context.Background(), toolCall("undo", "snapshot_undo", map[string]any{"id": id})); !r.Error {
		t.Fatal("direct execute bypassed plan")
	}
}
func TestPatchCheckpointIncludesMoveAndDelete(t *testing.T) {
	a, root := testAgent(t)
	a.Config.Snapshots = true
	a.Grants["apply_patch"] = true
	os.WriteFile(filepath.Join(root, "from"), []byte("old\n"), 0600)
	digest, _ := a.Tools.FileDigest("from")
	raw := map[string]any{"patchText": "*** Begin Patch\n*** Update File: from\n*** Move to: to\n@@\n-old\n+new\n*** Add File: added\n+hello\n*** End Patch", "expected_sha256": map[string]string{"from": digest["sha256"].(string)}}
	result := a.execute(context.Background(), toolCall("patch", "apply_patch", raw))
	if result.Error {
		t.Fatal(result)
	}
	m, err := snapshots.Open(root, a.snapshotDirectory())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	records, err := m.List()
	if err != nil || len(records) != 1 || len(records[0].Before) != 3 {
		t.Fatal(records, err)
	}
	if _, err = m.Undo(context.Background(), records[0].ID); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "from"))
	if string(b) != "old\n" {
		t.Fatal(string(b))
	}
	for _, p := range []string{"to", "added"} {
		if _, err = os.Stat(filepath.Join(root, p)); !os.IsNotExist(err) {
			t.Fatal(p, err)
		}
	}
}

func TestSnapshotUndoRuntimePermissionAndConcurrentReads(t *testing.T) {
	for _, mode := range []string{"headless", "denied", "granted", "conflict"} {
		t.Run(mode, func(t *testing.T) {
			a, root := testAgent(t)
			a.Config.Snapshots = true
			path := filepath.Join(root, "note")
			os.WriteFile(path, []byte("before"), 0600)
			m, err := snapshots.Open(root, a.snapshotDirectory())
			if err != nil {
				t.Fatal(err)
			}
			id, err := m.Capture(context.Background(), []string{"note"})
			if err != nil {
				t.Fatal(err)
			}
			os.WriteFile(path, []byte("after"), 0600)
			if _, err = m.Seal(context.Background(), id); err != nil {
				t.Fatal(err)
			}
			m.Close()
			if mode == "granted" || mode == "conflict" {
				a.Grants["snapshot_undo"] = true
			}
			if mode == "denied" {
				a.Approve = func(context.Context, core.Permission) (core.Decision, error) { return core.Decision{}, nil }
			}
			if mode == "conflict" {
				os.WriteFile(path, []byte("external"), 0600)
			}
			turn := 0
			a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
				turn++
				if turn == 1 {
					return answer(toolCall("list", "snapshot_list", map[string]any{}), toolCall("diff", "snapshot_diff", map[string]any{"id": id})), nil
				}
				if turn == 2 {
					// These adjacent read calls run concurrently; the manager's writer lock must
					// not cause one of the reads to fail nondeterministically.
					for _, msg := range h[len(h)-2:] {
						toolResult(t, msg)
					}
					return answer(toolCall("undo", "snapshot_undo", map[string]any{"id": id})), nil
				}
				return answer(), nil
			})
			status, err := a.Run(context.Background(), "inspect and undo")
			if mode == "headless" {
				if status != "needs_permission" || err == nil {
					t.Fatal(status, err)
				}
			} else if status != "completed" || err != nil {
				t.Fatal(status, err)
			}
			want := "after"
			if mode == "granted" {
				want = "before"
			}
			if mode == "conflict" {
				want = "external"
			}
			b, _ := os.ReadFile(path)
			if string(b) != want {
				t.Fatal("unapproved or stale undo", string(b), want)
			}
		})
	}
}
