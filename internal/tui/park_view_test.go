package tui

import (
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

// Leaving the chat for a picker and coming back shows the same screen:
// transcript, scroll position and display toggles. A changed journal or
// another session rebuilds instead.
func TestChatScreenSurvivesNavigation(t *testing.T) {
	cfg := core.Defaults()
	cfg.SessionDir, cfg.Model = t.TempDir(), "fixture"
	store, err := session.New(cfg.SessionDir, testutil.TempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	a, err := agent.New(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Append("message", core.Message{Role: "user", Content: "earlier prompt"}); err != nil {
		t.Fatal(err)
	}
	left := &model{a: a, log: "\nYou: earlier prompt\nlive transcript line\n", scrollOffset: 7, details: detailsPreviews, hideThinking: true, timestamps: true, hideSidebar: true}
	left.parkView()

	back := &model{a: a}
	if !back.unparkView() {
		t.Fatal("the parked screen was not restored")
	}
	if back.log != left.log || back.scrollOffset != 7 || back.details != detailsPreviews || !back.hideThinking || !back.timestamps || !back.hideSidebar {
		t.Fatalf("screen state not restored: %+v", back)
	}
	if again := (&model{a: a}); again.unparkView() {
		t.Fatal("a parked screen is used once")
	}

	left.parkView()
	if err := store.Append("message", core.Message{Role: "assistant", Content: "written elsewhere"}); err != nil {
		t.Fatal(err)
	}
	if changed := (&model{a: a}); changed.unparkView() || strings.Contains(changed.log, "live transcript") {
		t.Fatal("a changed journal must rebuild the transcript")
	}

	left.parkView()
	other, err := session.New(cfg.SessionDir, testutil.TempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	b, _ := agent.New(cfg, other)
	if (&model{a: b}).unparkView() {
		t.Fatal("another session must not show this screen")
	}
}

// A rate limit is named as one, and clears once the retry gets through.
func TestRateLimitRetryStatus(t *testing.T) {
	m := &model{width: 80, height: 24, active: true}
	m.Update(core.Event{Type: "provider_retry", Data: core.RetryNotice{Attempt: 2, Max: 3, DelayMS: 28000, Reason: "provider HTTP 429 (rate_limit)"}})
	if !strings.HasPrefix(m.status, "Rate limited by the provider") || !strings.Contains(m.status, "waiting 28s") {
		t.Fatal(m.status)
	}
	m.Update(core.Event{Type: "assistant_updated", Data: map[string]any{"text": "ok"}})
	if m.status != "running" {
		t.Fatal("status not cleared after the retry got through:", m.status)
	}
	if got := retryStatus(core.RetryNotice{Attempt: 2, Max: 3, DelayMS: 1500, Reason: "provider HTTP 503 (server)"}); !strings.HasPrefix(got, "Provider busy") {
		t.Fatal(got)
	}
}
