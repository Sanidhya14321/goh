package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestPreviewWithActiveWriterAndTornTail(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Append("message", core.Message{Role: "user", Content: "Fix the parser\nand check Unicode 界"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Append("message", core.Message{Role: "assistant", Content: "Parser verified"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Append("run_finished", map[string]string{"status": "completed"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, store.ID+".jsonl")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := Preview(dir, store.ID)
	if err != nil || preview.Title != "Fix the parser and check Unicode 界" || preview.Assistant != "Parser verified" || preview.Status != "completed" || preview.Limited {
		t.Fatal("active journal preview failed", preview, err)
	}
	if err := store.Append("run_started", map[string]string{"model": "fixture"}); err != nil {
		t.Fatal(err)
	}
	preview, err = Preview(dir, store.ID)
	if err != nil || preview.Status != "unfinished run" {
		t.Fatal("preview represented an unfinished run as completed", preview, err)
	}
	writer, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	writer.WriteString("{\"version\":")
	writer.Close()
	torn, _ := os.ReadFile(path)
	preview, err = Preview(dir, store.ID)
	after, _ := os.ReadFile(path)
	if err != nil || !preview.Limited || string(after) != string(torn) || len(after) <= len(before) {
		t.Fatal("preview repaired or discarded an incomplete tail", err)
	}
}

func TestPreviewLimitsAndSymlinkRefusal(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id := store.ID
	store.Append("message", core.Message{Role: "user", Content: "Bounded preview"})
	store.Append("message", core.Message{Role: "assistant", Content: strings.Repeat("x", previewBytes*2)})
	store.Close()
	before, _ := os.ReadFile(filepath.Join(dir, id+".jsonl"))
	preview, err := Preview(dir, id)
	after, _ := os.ReadFile(filepath.Join(dir, id+".jsonl"))
	if err != nil || !preview.Limited || preview.Title != "Bounded preview" || string(before) != string(after) {
		t.Fatal("bounded read failed", preview, err)
	}
	if _, err := Preview(dir, "../escape"); err == nil {
		t.Fatal("accepted traversal")
	}
	if err := os.Symlink(filepath.Join(dir, id+".jsonl"), filepath.Join(dir, "linked.jsonl")); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(dir, "linked"); err == nil {
		t.Fatal("followed preview symlink")
	}
}

func BenchmarkSessionPreview(b *testing.B) {
	dir := b.TempDir()
	store, err := New(dir, b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	id := store.ID
	if err := store.Append("message", core.Message{Role: "user", Content: "Inspect a large journal"}); err != nil {
		b.Fatal(err)
	}
	if err := store.Append("message", core.Message{Role: "assistant", Content: strings.Repeat("x", 1<<20)}); err != nil {
		b.Fatal(err)
	}
	store.Close()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Preview(dir, id); err != nil {
			b.Fatal(err)
		}
	}
}
