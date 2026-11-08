package tui

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"example.com/independent-harness/internal/session"
)

func TestTimestampLinesToggleAndCannotBeForged(t *testing.T) {
	at := time.Date(2026, 11, 8, 14, 3, 0, 0, time.Local)
	log := stampLine(at) + "\nYou: hi\n" + Safe("\x00Tforged 99:99") + "\n"
	hidden := withStamps(log, false)
	if strings.Contains(hidden, "14:03") || strings.Contains(hidden, stampMark) {
		t.Fatal("hidden view shows a time", hidden)
	}
	shown := withStamps(log, true)
	if !strings.Contains(shown, "· Sun 14:03") || strings.Contains(shown, stampMark) {
		t.Fatal("shown view", shown)
	}
	if strings.Contains(shown, "· forged") {
		t.Fatal("content forged a timestamp line")
	}
	if stampLine(time.Time{}) != "" {
		t.Fatal("unknown time should add nothing")
	}
	m := &model{}
	m.toggleTimestamps()
	if !m.timestamps || m.status != "Timestamps shown" {
		t.Fatal(m.status)
	}
}

func TestPromptTimesMatchByContentInOrder(t *testing.T) {
	rec := func(role, content string, at int64) session.Record {
		b, _ := json.Marshal(map[string]string{"role": role, "content": content})
		v := 2
		if at == 0 {
			v = 1
		}
		return session.Record{Version: v, Type: "message", Data: b, Time: at}
	}
	times := userPromptTimes([]session.Record{
		rec("user", "again", 1000), rec("assistant", "x", 1500), rec("user", "old", 0), rec("user", "again", 2000),
	})
	if popTime(times, "again").UnixMilli() != 1000 || popTime(times, "again").UnixMilli() != 2000 {
		t.Fatal("repeated prompts out of order")
	}
	if !popTime(times, "again").IsZero() || !popTime(times, "old").IsZero() || !popTime(times, "x").IsZero() {
		t.Fatal("expected no time for exhausted, untimed or assistant entries")
	}
}
