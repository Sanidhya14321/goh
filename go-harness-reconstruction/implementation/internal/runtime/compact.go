package runtime

import (
	"context"
	"errors"
	"strings"
	"time"
)

// CompactWith is Compact with a user focus ("keep the API decisions") that
// is added to the summarizer instructions for this compaction only.
func (a *Agent) CompactWith(ctx context.Context, focus string) (string, error) {
	if r := []rune(strings.TrimSpace(focus)); len(r) > 500 {
		focus = string(r[:500])
	}
	a.compactFocus = strings.TrimSpace(focus)
	defer func() { a.compactFocus = "" }()
	return a.Compact(ctx)
}

// Compact performs an explicit context operation over the same durable session.
// It neither sends a new user task nor starts executable integrations.
func (a *Agent) Compact(ctx context.Context) (status string, err error) {
	a.seq = 0
	a.runID = ID()
	status = "failed"
	started := time.Now()
	a.event("run_started", map[string]any{"operation": "compact", "model": a.Config.Model, "project": a.Store.Project})
	defer func() { status, err = a.endRun(ctx, status, err, started, false) }()
	if len(a.Store.Pending()) > 0 {
		a.event("recovery_required", map[string]any{"pending": a.Store.Pending()})
		return "failed", errors.New("uncertain tool effects require acknowledgment before compaction")
	}
	if len(a.History) <= 1 {
		return "failed", errors.New("session has no conversation to compact")
	}
	// Local tools reserve their future schemas. Explicit compaction does not launch MCP.
	defs := a.localDefinitions()
	if e := a.compactWithMode(ctx, defs, true); e != nil {
		return "budget_exhausted", e
	}
	return "completed", nil
}
