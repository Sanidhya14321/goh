package session

import (
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// Messages GoHarness writes itself (checks, repair requests, pinned
// instructions) are not user turns for undo, fork, timeline or statistics.
func TestAutomaticMessagesAreNotUserTurns(t *testing.T) {
	s, err := New(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, m := range []core.Message{
		{Role: "user", Content: "real request"},
		{Role: "assistant", Content: "done"},
		{Role: "user", Content: "[Automatic check] run the tests", Automatic: true},
		{Role: "assistant", Content: "tests pass"},
	} {
		if err := s.Append("message", m); err != nil {
			t.Fatal(err)
		}
	}
	turns := s.RecentUserTurns(10)
	if len(turns) != 1 || turns[0].Prompt != "real request" {
		t.Fatalf("turns %+v", turns)
	}
}
