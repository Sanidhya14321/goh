package runtime

import (
	"context"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
)

const titlePrompt = "Write a short title (at most 6 words) for the coding request below. Reply with the title only: no quotes, no punctuation at the end."

// maybeAutoTitle names a top-level session after its first completed run
// when auto_title is on and the user has not chosen a title. It is
// best-effort: any failure leaves the default title, and the request sees
// only the first prompt (no tools, files or later turns).
func (a *Agent) maybeAutoTitle(ctx context.Context) {
	if !a.Config.AutoTitle || a.depth > 0 || a.Config.SessionDir == "" || ctx.Err() != nil {
		return
	}
	if a.Config.MaxRunTokens > 0 && a.runTokens >= a.Config.MaxRunTokens {
		return // the run's token allowance is spent; a title is optional
	}
	users := 0
	for _, m := range a.History {
		if m.Role == "user" && !m.Automatic {
			users++
		}
	}
	// The title reads only what the user typed, never text added by @file
	// or @agent expansion or by a custom command template.
	first := a.titleSource
	if users != 1 || strings.TrimSpace(first) == "" {
		return
	}
	if meta, err := session.ReadMeta(a.Config.SessionDir, a.Store.ID); err != nil || meta.Title != "" {
		return
	}
	if r := []rune(first); len(r) > 2000 {
		first = string(r[:2000])
	}
	model, cfg, err := a.modelForRole("title")
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req := []core.Message{{Role: "system", Content: titlePrompt}, {Role: "user", Content: first}}
	reply, err := a.completeWithFailover(ctx, "title", model, cfg, req, nil, nil)
	if err != nil || len(reply.Message.ToolCalls) > 0 {
		return
	}
	title := strings.Trim(session.CleanTitle(strings.SplitN(reply.Message.Content, "\n", 2)[0]), "\"'`. ")
	if r := []rune(title); len(r) > 80 {
		title = string(r[:80])
	}
	if title == "" || session.SetTitle(a.Config.SessionDir, a.Store.ID, title) != nil {
		return
	}
	a.event("session_titled", map[string]any{"title": title})
}
