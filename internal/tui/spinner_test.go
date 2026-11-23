package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Sanidhya14321/goh/internal/core"
)

func TestHumanizers(t *testing.T) {
	for _, c := range []struct {
		got, want string
	}{
		{humanTokens(0), ""}, {humanTokens(512), "512 tokens"}, {humanTokens(12345), "12.3k tokens"},
		{humanTokens(2_500_000), "2.50M tokens"}, {humanDuration(0), ""}, {humanDuration(333), "333 ms"},
		{humanDuration(4200), "4.2s"}, {humanDuration(125_000), "2m 05s"},
	} {
		if c.got != c.want {
			t.Errorf("got %q want %q", c.got, c.want)
		}
	}
}

func TestStatusLineStates(t *testing.T) {
	m := &model{status: "completed", durationMS: 1500}
	if got := m.statusLine(); got != "completed  ·  1.5s" {
		t.Fatalf("idle: %q", got)
	}
	m = &model{active: true, pending: &permissionMsg{}}
	if got := m.statusLine(); got != "● Waiting for your approval" {
		t.Fatalf("approval: %q", got)
	}
}

func TestRelativizePreview(t *testing.T) {
	got := relativizePreview("Command: ls\nDirectory: /p/x\nTimeout: 5s", "/p/x")
	if got != "Command: ls\nDirectory: . (project root)\nTimeout: 5s" {
		t.Fatalf("%q", got)
	}
}

func TestRelativizePreviewLeavesCommandsAndContentExact(t *testing.T) {
	in := "Command: cat /repo/file\nDirectory: /repo/sub\nTimeout: 5s"
	if got := relativizePreview(in, "/repo"); got != "Command: cat /repo/file\nDirectory: sub\nTimeout: 5s" {
		t.Fatalf("%q", got)
	}
	diff := "--- /repo/a.go\n+++ /repo/a.go\n-x := \"/repo/b\"\n+x := \"/repo/c\""
	if got := relativizePreview(diff, "/repo"); got != "--- a.go\n+++ a.go\n-x := \"/repo/b\"\n+x := \"/repo/c\"" {
		t.Fatalf("%q", got)
	}
	patch := "File 1: MOVE /repo/a.go -> /repo/b.go\nFile 1 before (-):\n-/repo/x"
	if got := relativizePreview(patch, "/repo"); got != "File 1: MOVE a.go -> b.go\nFile 1 before (-):\n-/repo/x" {
		t.Fatalf("%q", got)
	}
}

func TestRelativizePreviewLeavesHeaderLookalikesInBodies(t *testing.T) {
	heredoc := "Command: cat <<EOF\nDirectory: /repo/x\n--- /repo/y\nFile 1: ADD /repo/z\nEOF\nDirectory: /repo\nTimeout: 5s"
	want := "Command: cat <<EOF\nDirectory: /repo/x\n--- /repo/y\nFile 1: ADD /repo/z\nEOF\nDirectory: . (project root)\nTimeout: 5s"
	if got := relativizePreview(heredoc, "/repo"); got != want {
		t.Fatalf("%q", got)
	}
	diff := "--- /repo/a\n+++ /repo/a\n--- /repo/b\n+++ /repo/c\n-Directory: /repo/d"
	if got := relativizePreview(diff, "/repo"); got != "--- a\n+++ a\n--- /repo/b\n+++ /repo/c\n-Directory: /repo/d" {
		t.Fatalf("%q", got)
	}
	patch := "Apply patch (all files):\nFile 1: UPDATE /repo/a\n\nFile 1 before (-):\n--- /repo/q\nFile 2: ADD /repo/r"
	if got := relativizePreview(patch, "/repo"); got != "Apply patch (all files):\nFile 1: UPDATE a\n\nFile 1 before (-):\n--- /repo/q\nFile 2: ADD /repo/r" {
		t.Fatalf("%q", got)
	}
}

func TestShortPathKeepsTail(t *testing.T) {
	if got := shortPath("/very/long/path/to/project", 10); got != "…o/project" {
		t.Fatalf("%q", got)
	}
}

func TestRunNoticesShowBesideSpinner(t *testing.T) {
	m := &model{active: true, status: "Queue full (8 prompts); the draft is kept"}
	if got := m.statusLine(); !strings.Contains(got, "Queue full") {
		t.Fatalf("%q", got)
	}
	m.status = "running"
	if got := m.statusLine(); strings.Contains(got, "running") {
		t.Fatalf("%q", got)
	}
}

func TestRetryRemovesDiscardedPartialReply(t *testing.T) {
	m := &model{events: make(chan tea.Msg, 4), log: "You: hi\n"}
	m.Update(core.Event{Type: "assistant_updated", Data: map[string]any{"text": "HALF AN ANS"}})
	m.Update(core.Event{Type: "stream_retry", Data: map[string]any{"attempt": 1}})
	m.Update(core.Event{Type: "assistant_updated", Data: map[string]any{"text": "Full answer."}})
	if strings.Contains(m.log, "HALF AN ANS") || !strings.Contains(m.log, "Full answer.") || !strings.Contains(m.log, "discarded") {
		t.Fatalf("%q", m.log)
	}
}

func TestRetryRemovesPartialReplyFollowedByReasoning(t *testing.T) {
	m := &model{events: make(chan tea.Msg, 8), log: "You: hi\n"}
	m.Update(core.Event{Type: "assistant_updated", Data: map[string]any{"text": "ABANDONED TEXT"}})
	m.Update(core.Event{Type: "reasoning_updated", Data: map[string]any{"text": "thinking more"}})
	m.Update(core.Event{Type: "stream_retry", Data: map[string]any{"attempt": 1}})
	m.Update(core.Event{Type: "assistant_updated", Data: map[string]any{"text": "FIRST"}})
	m.Update(core.Event{Type: "assistant_updated", Data: map[string]any{"text": " AND LAST"}})
	if strings.Contains(m.log, "ABANDONED") || !strings.Contains(m.log, "FIRST AND LAST") {
		t.Fatalf("%q", m.log)
	}
	m.Update(core.Event{Type: "assistant_updated", Data: map[string]any{"text": "\nDRAFT"}})
	m.Update(core.Event{Type: "assistant_reset", Data: map[string]any{}})
	m.Update(core.Event{Type: "assistant_updated", Data: map[string]any{"text": "FINAL"}})
	if strings.Contains(m.log, "DRAFT") || !strings.HasSuffix(m.log, "FINAL") {
		t.Fatalf("%q", m.log)
	}
}

// The running timer counts in tenths of a second from the start, not in
// whole-second jumps that begin at 1s.
func TestLiveTimerShowsTenths(t *testing.T) {
	for _, c := range []struct {
		ms   int64
		want string
	}{{0, "0.0s"}, {340, "0.3s"}, {2_390, "2.3s"}, {59_990, "59.9s"}, {61_000, "1m 01s"}} {
		if got := liveDuration(c.ms); got != c.want {
			t.Errorf("liveDuration(%d) = %q, want %q", c.ms, got, c.want)
		}
	}
	m := &model{active: true, runStarted: time.Now().Add(-2350 * time.Millisecond)}
	if got := m.statusLine(); !strings.Contains(got, "  ·  2.3s") && !strings.Contains(got, "  ·  2.4s") {
		t.Fatalf("running: %q", got)
	}
	m.runStarted = time.Now().Add(-400 * time.Millisecond)
	if got := m.statusLine(); !strings.Contains(got, "  ·  0.4s") && !strings.Contains(got, "  ·  0.5s") {
		t.Fatalf("first second hidden: %q", got)
	}
}

// Green means a passing or unchanged run; unverified changes are amber,
// failed verification red.
func TestOutcomeColors(t *testing.T) {
	for outcome, want := range map[string]string{"verified": green, "no_changes": green, "unverified": accent, "failed_verification": red} {
		r := &runResult{Status: "completed", Outcome: outcome}
		m := &model{lastRun: r, status: r.label()}
		if got := statusColor(m); got != want {
			t.Errorf("%s: color %q, want %q", outcome, got, want)
		}
	}
}
