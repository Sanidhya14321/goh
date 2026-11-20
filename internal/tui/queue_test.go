package tui

import (
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"

	tea "charm.land/bubbletea/v2"
)

func typeText(m *model, text string) {
	for _, r := range text {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func TestQueueWhileActive(t *testing.T) {
	m := &model{width: 80, height: 20, active: true, cancel: func() {}}
	typeText(m, "next step")
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(m.queue) != 1 || m.queue[0] != "next step" || m.input != "" || !strings.Contains(m.log, "[queued] next step") {
		t.Fatal("prompt not queued", m.queue, m.input)
	}
	typeText(m, "/fork")
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(m.queue) != 1 || m.input != "/fork" {
		t.Fatal("slash commands must not be queued and the draft must stay", m.queue, m.input)
	}
	m.input, m.inputCursor = "", 0
	for i := 0; i < maxQueued; i++ {
		typeText(m, "p")
		m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	}
	if len(m.queue) != maxQueued || m.input != "p" || !strings.Contains(m.status, "Queue full") {
		t.Fatal("queue limit not enforced", len(m.queue), m.input, m.status)
	}
	// Dialog shortcuts stay unavailable during a run.
	m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	if m.palette {
		t.Fatal("palette opened during an active run")
	}
}

func TestQueuedPromptsHeldAfterFailure(t *testing.T) {
	m := &model{width: 80, height: 20, active: true, cancel: func() {}, queue: []string{"one", "two"}}
	typeText(m, "draft")
	_, cmd := m.Update(doneMsg{status: "cancelled"})
	if cmd != nil || len(m.queue) != 2 || m.input != "draft" || m.queueHeld == "" || !strings.Contains(m.log, "held because the run ended with status cancelled") {
		t.Fatalf("queued prompts must be held in order with the draft kept: queue %q input %q", m.queue, m.input)
	}
	if m.active {
		t.Fatal("run still marked active")
	}
	// ↑ on an empty composer takes the newest back; the order is kept.
	m.input, m.inputCursor = "", 0
	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.input != "two" || len(m.queue) != 1 || m.queue[0] != "one" {
		t.Fatalf("unqueue: input %q queue %q", m.input, m.queue)
	}
}

// The vertical slice: a run whose configured check failed shows a
// persistent failure and holds the queue even though its status is
// completed; doneMsg does not overwrite the run's own result.
func TestFailedVerificationHoldsQueue(t *testing.T) {
	m := &model{width: 80, height: 20, active: true, cancel: func() {}, queue: []string{"follow up"}}
	m.events = make(chan tea.Msg, 1)
	code := 1
	m.Update((core.Event{Type: "run_finished", RunID: "r1", Data: map[string]any{
		"status": "completed", "outcome": "failed_verification",
		"verification_detail": map[string]any{"change_tracking": "known", "evidence": map[string]any{"source": "configured", "command": "go test ./...", "exit_code": code, "passed": false}},
	}}))
	_, cmd := m.Update(doneMsg{status: "completed"})
	if cmd != nil {
		t.Fatal("a failed verification must not send the queued prompt")
	}
	if m.status != "Verification failed: go test ./... (exit 1)" || statusColor(m) != red {
		t.Fatalf("status %q", m.status)
	}
	if len(m.queue) != 1 || !strings.Contains(m.log, "Verification failed: go test ./... (exit 1). The changes were kept") || !strings.Contains(m.log, "held because the configured verification failed") {
		t.Fatalf("log %q queue %q", m.log, m.queue)
	}
}

func TestCompletedRunSchedulesNextQueuedPrompt(t *testing.T) {
	m := &model{width: 80, height: 20, active: true, cancel: func() {}, queue: []string{"one"}}
	_, cmd := m.Update(doneMsg{status: "completed"})
	if cmd == nil {
		t.Fatal("completed run should schedule the queued prompt")
	}
	if _, ok := cmd().(queuedSubmitMsg); !ok {
		t.Fatal("unexpected follow-up message")
	}
}

// Reopening a session shows its latest verification result, tied to that
// run; the next run starts without it.
func TestReopenShowsLastVerification(t *testing.T) {
	c := core.Defaults()
	s, err := session.New(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.Append("run_started", map[string]any{})
	s.Append("run_finished", map[string]any{"status": "completed", "outcome": "failed_verification", "run_id": "r9",
		"verification_detail": map[string]any{"evidence": map[string]any{"source": "configured", "command": "make check", "exit_code": 2}}})
	a, err := agent.New(c, s)
	if err != nil {
		t.Fatal(err)
	}
	m := &model{a: a, width: 80, height: 24, status: "ready", events: make(chan tea.Msg, 4)}
	m.restoreLastRun()
	if m.lastRun == nil || m.lastRun.RunID != "r9" || m.status != "Last run: Verification failed: make check (exit 2)" || !strings.Contains(m.log, "The changes were kept") {
		t.Fatalf("status %q run %+v", m.status, m.lastRun)
	}
	m.Update(core.Event{Type: "run_started", RunID: "r10"})
	if m.lastRun != nil {
		t.Fatal("new turn inherited the previous run's result")
	}
}

// Review: a subagent's retry retracts only its own abandoned text.
func TestChildRetryRetractsOnlyItsText(t *testing.T) {
	m := &model{width: 80, height: 24, active: true, events: make(chan tea.Msg, 8)}
	child := func(task, kind string, data any) core.Event {
		return core.Event{Type: "subagent_event", Data: map[string]any{"task_id": task, "event": core.Event{Type: kind, Data: data}}}
	}
	m.log = "parent text\n"
	m.Update(child("a", "assistant_updated", map[string]any{"text": "BROKEN-A"}))
	m.Update(child("b", "assistant_updated", map[string]any{"text": "KEEP-B"}))
	m.Update(child("a", "assistant_updated", map[string]any{"text": " more"}))
	m.Update(child("a", "stream_retry", map[string]any{"attempt": 1}))
	m.Update(child("a", "assistant_updated", map[string]any{"text": "GOOD-A"}))
	if strings.Contains(m.log, "BROKEN-A") || strings.Contains(m.log, " more") || !strings.Contains(m.log, "KEEP-B") || !strings.Contains(m.log, "GOOD-A") || !strings.Contains(m.log, "parent text") {
		t.Fatalf("log %q", m.log)
	}
}

// Review: /find reaches turns that are only in the journal.
func TestFindSearchesEarlierJournalTurns(t *testing.T) {
	s, err := session.New(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.Append("message", core.Message{Role: "user", Content: "please rename the zebra handler"})
	s.Append("message", core.Message{Role: "assistant", Content: "Renamed the zebra handler."})
	a, err := agent.New(core.Defaults(), s)
	if err != nil {
		t.Fatal(err)
	}
	m := &model{a: a, width: 80, height: 24, log: "recent turn only\n"}
	m.startFind("zebra")
	if m.find == nil || len(m.find.matches) != 2 || !strings.Contains(m.log, "turn 1 · You: please rename the zebra handler") {
		t.Fatalf("find %+v log %q", m.find, m.log)
	}
}

// Review: at a tiny height a picker still shows the option Enter applies.
func TestPickerTinyHeightShowsSelection(t *testing.T) {
	m := &optionPicker{title: "Theme", options: []Option{{Value: "a", Label: "Alpha"}, {Value: "b", Label: "Bravo"}}, width: 72, height: 3}
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if v := m.View().Content; !strings.Contains(v, "Bravo") {
		t.Fatalf("selected option not visible:\n%s", v)
	}
}

// Follow-up review P1: /find over multiline Unicode history neither panics
// nor loses a match beyond the excerpt cap.
func TestFindUnicodeAndLongLines(t *testing.T) {
	s, err := session.New(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// "İ" lowercases to a longer byte sequence; "ẞ" folds to "ß".
	s.Append("message", core.Message{Role: "user", Content: "İİİİ first line\nİSTANBUL Straße ẞ zebra here\nlast"})
	long := strings.Repeat("filler words ", 60) + "NEEDLE-AT-END tail"
	s.Append("message", core.Message{Role: "assistant", Content: long})
	a, err := agent.New(core.Defaults(), s)
	if err != nil {
		t.Fatal(err)
	}
	m := &model{a: a, width: 80, height: 24, log: "recent\n"}
	m.startFind("ZEBRA")
	if m.find == nil || !strings.Contains(m.log, "İSTANBUL Straße ẞ zebra here") {
		t.Fatalf("unicode match: %q", m.log)
	}
	m.find = nil
	m.startFind("needle-at-end")
	if m.find == nil || len(m.find.matches) != 1 || !strings.Contains(m.log, "NEEDLE-AT-END") {
		t.Fatalf("late match lost: find %+v", m.find)
	}
	if at, end := foldIndex("İİ zebra", "ZEBRA"); at < 0 || "İİ zebra"[at:end] != "zebra" {
		t.Fatal(at, end)
	}
}

// Follow-up review: a child's retry removes its own text at the place it
// was written, not an equal string elsewhere (a queued prompt's preview).
func TestChildRetryRemovesOnlyItsOwnSpan(t *testing.T) {
	m := &model{width: 80, height: 24, active: true, events: make(chan tea.Msg, 8)}
	child := func(task, kind string, data any) core.Event {
		return core.Event{Type: "subagent_event", Data: map[string]any{"task_id": task, "event": core.Event{Type: kind, Data: data}}}
	}
	m.Update(child("a", "assistant_updated", map[string]any{"text": "fix it"}))
	m.log += "\n[queued] fix it\n" // the same words in the user's queued prompt
	m.Update(child("b", "assistant_updated", map[string]any{"text": "B text"}))
	m.Update(child("a", "stream_retry", map[string]any{"attempt": 1}))
	if !strings.Contains(m.log, "[queued] fix it") || strings.HasPrefix(m.log, "fix it") || !strings.Contains(m.log, "B text") {
		t.Fatalf("log %q", m.log)
	}
	// The other child's span still points at its own text.
	m.Update(child("b", "stream_retry", map[string]any{"attempt": 1}))
	if strings.Contains(m.log, "B text") || !strings.Contains(m.log, "[queued] fix it") {
		t.Fatalf("log %q", m.log)
	}
}

// Follow-up review: held prompts survive leaving the same chat for a picker
// and coming back; another session does not get them.
func TestHeldQueueSurvivesSameChatNavigation(t *testing.T) {
	parkQueue("s1", []string{"one", "two"}, "the configured verification failed")
	if q, _ := unparkQueue("s2"); q != nil {
		t.Fatal("another session got the held prompts")
	}
	parkQueue("s1", []string{"one", "two"}, "the configured verification failed")
	q, reason := unparkQueue("s1")
	if len(q) != 2 || q[0] != "one" || reason == "" {
		t.Fatal(q, reason)
	}
	if q, _ := unparkQueue("s1"); q != nil {
		t.Fatal("held prompts restored twice")
	}
}
