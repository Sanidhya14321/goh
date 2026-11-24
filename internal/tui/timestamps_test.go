package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/testutil"
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

// A reopened chat stamps each prompt with its own record's time. Before,
// times were matched by text across the whole journal, so when an earlier
// identical prompt fell outside the shown tail, the recent one got its time.
func TestRepeatedPromptKeepsItsOwnTime(t *testing.T) {
	store, err := session.New(t.TempDir(), testutil.TempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Append("message", core.Message{Role: "user", Content: "run the tests"}); err != nil {
		t.Fatal(err)
	}
	for i := range 80 { // pushes the first prompt out of the shown tail
		if err := store.Append("message", core.Message{Role: "assistant", Content: strings.Repeat("output ", 600) + fmt.Sprint(i)}); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(5 * time.Millisecond)
	if err := store.Append("message", core.Message{Role: "user", Content: "run the tests"}); err != nil {
		t.Fatal(err)
	}
	last, _ := store.Last("message")
	a := &agent.Agent{Store: store}
	entries := screenHistory(a)
	prompts := 0
	for _, e := range entries {
		if e.message.Role == "user" {
			prompts++
			if e.at.UnixMilli() != last.Time {
				t.Fatalf("prompt shown with %d, its record has %d", e.at.UnixMilli(), last.Time)
			}
		}
	}
	if prompts != 1 {
		t.Fatalf("the earlier prompt should be outside the tail: %d prompts", prompts)
	}
}
