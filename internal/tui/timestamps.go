package tui

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/session"
)

// stampMark starts a transcript line holding a turn time. Safe() replaces
// NUL in every model, tool and user string, so content cannot forge one.
const stampMark = "\x00T"

func stampLine(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return "\n" + stampMark + t.Local().Format("Mon 15:04")
}

// withStamps shows or hides turn-time lines (OpenCode's timestamp toggle).
func withStamps(log string, show bool) string {
	if !strings.Contains(log, stampMark) {
		return log
	}
	lines := strings.Split(log, "\n")
	out := lines[:0]
	for _, line := range lines {
		if rest, ok := strings.CutPrefix(line, stampMark); ok {
			if show {
				out = append(out, "  · "+rest)
			}
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// userPromptTimes returns journaled times of user messages, in order, keyed by
// content so compacted or summarized history still matches the right prompt.
// Records from journals older than record timestamps have none.
func userPromptTimes(records []session.Record) map[string][]time.Time {
	times := map[string][]time.Time{}
	for _, r := range records {
		if r.Type != "message" || r.Time <= 0 {
			continue
		}
		var msg struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		if json.Unmarshal(r.Data, &msg) == nil && msg.Role == "user" {
			times[msg.Content] = append(times[msg.Content], time.UnixMilli(r.Time))
		}
	}
	return times
}

func popTime(times map[string][]time.Time, content string) time.Time {
	q := times[content]
	if len(q) == 0 {
		return time.Time{}
	}
	times[content] = q[1:]
	return q[0]
}

func (m *model) toggleTimestamps() {
	m.timestamps = !m.timestamps
	m.status = "Timestamps hidden"
	if m.timestamps {
		m.status = "Timestamps shown"
	}
}
