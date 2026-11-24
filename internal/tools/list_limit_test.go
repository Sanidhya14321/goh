package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// A limited listing stops walking once it has more files than the limit.
func TestListFilesStopsAtTheLimit(t *testing.T) {
	e, root := engine(t)
	for i := 0; i < 30; i++ {
		dir := filepath.Join(root, fmt.Sprintf("d%02d", i))
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x\n"), 0o644)
	}
	r := e.Execute(context.Background(), "list_files", json.RawMessage(`{"limit":5}`))
	var out struct {
		Files     []string `json:"files"`
		Truncated bool     `json:"truncated"`
	}
	if r.Error || json.Unmarshal([]byte(r.Content), &out) != nil {
		t.Fatalf("list_files: %s", r.Content)
	}
	if len(out.Files) != 5 || !out.Truncated || out.Files[0] != "d00/x.go" || out.Files[4] != "d04/x.go" {
		t.Fatalf("got %v truncated=%v", out.Files, out.Truncated)
	}
	r = e.Execute(context.Background(), "list_files", json.RawMessage(`{"limit":30}`))
	json.Unmarshal([]byte(r.Content), &out)
	if len(out.Files) != 30 || out.Truncated {
		t.Fatalf("exactly the limit: %d files, truncated=%v", len(out.Files), out.Truncated)
	}
}
