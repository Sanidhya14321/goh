package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// A saved page or log can be searched: matching lines come back with their
// byte offsets, so the model reads only the part it needs.
func TestArtifactPatternSearch(t *testing.T) {
	e, err := New(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	page := strings.Repeat("navigation\n", 300) + "go1.25.4 (released 2026-11-04)\n" + strings.Repeat("9fa5ffeda417 checksum\n", 300) + "go1.24.10 (released 2026-11-04)"
	id, err := e.SaveArtifact("fetch", page)
	if err != nil || !strings.HasPrefix(id, "fetch-") {
		t.Fatal(id, err)
	}
	if _, err := e.SaveArtifact("../x", "y"); err == nil {
		t.Fatal("unknown kinds are refused")
	}
	run := func(args map[string]any) Result {
		raw, _ := json.Marshal(args)
		return e.Execute(context.Background(), "read_artifact", raw)
	}
	out := decodeEfficiency(t, run(map[string]any{"id": id, "pattern": `go1\.\d+\.\d+`}))
	content := out["content"].(string)
	first := strings.Index(page, "go1.25.4")
	if out["matches"].(float64) != 2 || !strings.HasPrefix(content, itoa(first)+": go1.25.4") || !strings.Contains(content, ": go1.24.10") {
		t.Fatal(out)
	}
	// The listing is bounded by limit; next_offset resumes after it.
	out = decodeEfficiency(t, run(map[string]any{"id": id, "pattern": "checksum", "limit": 100}))
	if out["matches"].(float64) != 300 || len(out["content"].(string)) > 100 || out["next_offset"].(float64) <= float64(first) {
		t.Fatal(out)
	}
	for _, bad := range []map[string]any{{"id": id, "pattern": "("}, {"id": id, "pattern": "x", "tail": true}, {"id": id, "pattern": ""}} {
		if r := run(bad); !r.Error {
			t.Fatal("accepted", bad)
		}
	}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestArtifactIDUsedAsPathGetsHint(t *testing.T) {
	e, err := New(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"path": "fetch-432435538.log", "pattern": "go1"})
	if r := e.Execute(context.Background(), "grep", raw); !r.Error || !strings.Contains(r.Content, "use read_artifact") {
		t.Fatal(r)
	}
}
