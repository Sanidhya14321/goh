package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/independent-harness/internal/testutil"
)

func TestCollectStats(t *testing.T) {
	dir := t.TempDir()
	projA, projB := testutil.TempDir(t), testutil.TempDir(t)
	add := func(project, model string, prompt, completion int, age time.Duration) {
		s, err := New(dir, project)
		if err != nil {
			t.Fatal(err)
		}
		s.Append("message", map[string]string{"role": "user", "content": "hi"})
		s.Append("usage", map[string]any{"model": model, "prompt_tokens": prompt, "completion_tokens": completion, "total_tokens": prompt + completion})
		s.Append("usage", map[string]any{"model": model, "prompt_tokens": -5, "completion_tokens": 1, "total_tokens": 1}) // ignored
		id := s.ID
		s.Close()
		when := time.Now().Add(-age)
		os.Chtimes(filepath.Join(dir, id+".jsonl"), when, when)
	}
	add(projA, "big", 100, 50, time.Hour)
	add(projA, "small", 10, 5, 48*time.Hour)
	add(projB, "big", 1000, 1, time.Hour)
	all, err := CollectStats(dir, "", time.Time{})
	if err != nil || all.Sessions != 3 || all.Prompts != 3 || len(all.Models) != 2 || all.Models[0].Model != "big" || all.Models[0].Total != 1151 || all.Models[0].Replies != 2 {
		t.Fatalf("%+v %v", all, err)
	}
	a, _ := CollectStats(dir, projA, time.Now().Add(-24*time.Hour))
	if a.Sessions != 1 || len(a.Models) != 1 || a.Models[0].Total != 150 {
		t.Fatalf("filters wrong: %+v", a)
	}
}
