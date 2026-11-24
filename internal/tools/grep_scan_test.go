package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Matches come in listing order with the first 100 kept, however the files
// are spread over folders and workers; a cancelled search fails.
func TestGrepKeepsListingOrder(t *testing.T) {
	e, root := engine(t)
	for d := 0; d < 12; d++ {
		for f := 0; f < 40; f++ {
			p := filepath.Join(root, fmt.Sprintf("d%02d", d), fmt.Sprintf("f%02d.txt", f))
			os.MkdirAll(filepath.Dir(p), 0o755)
			os.WriteFile(p, []byte("x\nneedle\n"), 0o644)
		}
	}
	r := e.Execute(context.Background(), "grep", json.RawMessage(`{"pattern":"needle"}`))
	var out struct {
		Matches []struct {
			Path string `json:"path"`
			Line int    `json:"line"`
		} `json:"matches"`
		Truncated bool `json:"truncated"`
	}
	if r.Error || json.Unmarshal([]byte(r.Content), &out) != nil {
		t.Fatalf("grep: %s", r.Content)
	}
	if len(out.Matches) != 100 || !out.Truncated {
		t.Fatalf("%d matches, truncated=%v", len(out.Matches), out.Truncated)
	}
	for i, m := range out.Matches {
		if want := fmt.Sprintf("d%02d/f%02d.txt", i/40, i%40); m.Path != want || m.Line != 2 {
			t.Fatalf("match %d is %s:%d, want %s:2", i, m.Path, m.Line, want)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if r := e.Execute(ctx, "grep", json.RawMessage(`{"pattern":"needle"}`)); !r.Error {
		t.Fatalf("cancelled grep succeeded: %.80s", r.Content)
	}
}
