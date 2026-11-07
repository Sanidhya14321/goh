package tui

import (
	"strings"
	"testing"
)

func TestFailoverLine(t *testing.T) {
	got := failoverLine(map[string]any{"purpose": "compaction", "from": "default", "to": "backup", "reason": "rate_limit", "state": "attempting"})
	if !strings.Contains(got, "default failed (rate_limit) before responding; retrying context compaction with fallback profile backup") {
		t.Fatal(got)
	}
	if got := failoverLine(map[string]any{"to": "small", "state": "skipped", "detail": "conversation exceeds fallback context window"}); !strings.Contains(got, "skipped fallback small") {
		t.Fatal(got)
	}
}
