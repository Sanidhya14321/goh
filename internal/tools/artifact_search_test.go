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

// Paging a search with next_offset lists every match exactly once and always
// advances, even when one match line is longer than the limit or a short
// match follows one that did not fit.
func TestArtifactSearchPagesEachMatchOnce(t *testing.T) {
	e, err := New(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var page strings.Builder
	want := 0
	for i := range 60 {
		switch i % 3 {
		case 0:
			page.WriteString("hit " + strings.Repeat("long", 70) + "\n") // cut to 300 bytes, wider than limit 120
		case 1:
			page.WriteString("hit short\n")
		default:
			page.WriteString("other\n")
			continue
		}
		want++
	}
	id, _ := e.SaveArtifact("fetch", page.String())
	seen := map[string]int{}
	offset := 0.0
	for calls := 0; ; calls++ {
		if calls > want+5 {
			t.Fatal("search did not advance")
		}
		raw, _ := json.Marshal(map[string]any{"id": id, "pattern": "^hit", "limit": 120, "offset": offset})
		out := decodeEfficiency(t, e.Execute(context.Background(), "read_artifact", raw))
		for _, line := range strings.Split(strings.TrimSuffix(out["content"].(string), "\n"), "\n") {
			if at, _, ok := strings.Cut(line, ": "); ok {
				seen[at]++
			}
		}
		next := out["next_offset"].(float64)
		if out["eof"] == true {
			break
		}
		if next <= offset {
			t.Fatalf("next_offset %v did not advance from %v", next, offset)
		}
		offset = next
	}
	if len(seen) != want {
		t.Fatalf("listed %d distinct matches, want %d", len(seen), want)
	}
	for at, n := range seen {
		if n != 1 {
			t.Fatalf("match at %s listed %d times", at, n)
		}
	}
}

// Paging text by next_offset keeps every character whole.
func TestArtifactTextPagesKeepCharactersWhole(t *testing.T) {
	e, err := New(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("aé界😀", 400)
	id, _ := e.SaveArtifact("fetch", text)
	for _, limit := range []int{1, 2, 3, 7, 100} {
		var got strings.Builder
		offset := 0.0
		for calls := 0; ; calls++ {
			if calls > len(text)+2 {
				t.Fatalf("limit %d: paging did not finish", limit)
			}
			raw, _ := json.Marshal(map[string]any{"id": id, "offset": offset, "limit": limit})
			out := decodeEfficiency(t, e.Execute(context.Background(), "read_artifact", raw))
			if out["utf8_repaired"] == true {
				t.Fatalf("limit %d: a page split a character at %v", limit, offset)
			}
			got.WriteString(out["content"].(string))
			if out["eof"] == true {
				break
			}
			if next := out["next_offset"].(float64); next <= offset {
				t.Fatalf("limit %d: stalled at %v", limit, offset)
			} else {
				offset = next
			}
		}
		if got.String() != text {
			t.Fatalf("limit %d: paged text differs", limit)
		}
	}
}
