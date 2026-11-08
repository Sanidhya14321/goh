package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"example.com/independent-harness/internal/core"
)

func TestProjectCommandFileExpandsAndConfigWins(t *testing.T) {
	base := contextAgent(t)
	dir := filepath.Join(base.Store.Project, ".opencode", "command")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "review.md"), []byte("---\ndescription: Review\n---\nReview $1 please"), 0o600)
	os.WriteFile(filepath.Join(dir, "mine.md"), []byte("from repo"), 0o600)
	cfg := base.Config
	cfg.Commands = map[string]core.CommandConfig{"mine": {Template: "from config"}}
	a, err := New(cfg, base.Store)
	if err != nil {
		t.Fatal(err)
	}
	if a.CommandSet()["review"].Description != "Review" || a.CommandSet()["mine"].Template != "from config" {
		t.Fatal(a.CommandSet())
	}
	m := &summaryModel{summary: "ok"}
	a.Provider = m
	if status, err := a.Run(context.Background(), "/review parser.go"); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	last := ""
	for _, msg := range m.seen {
		if msg.Role == "user" {
			last = msg.Content
		}
	}
	if last != "Review parser.go please" {
		t.Fatalf("model saw %q", last)
	}
}
