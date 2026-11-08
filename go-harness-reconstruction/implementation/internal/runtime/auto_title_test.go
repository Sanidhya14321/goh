package runtime

import (
	"context"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/session"
)

func TestAutoTitleAfterFirstRun(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.AutoTitle = true
	var titleReq []core.Message
	a.Provider = modelFunc(func(_ context.Context, ms []core.Message, defs []core.Tool, _ func(string)) (core.Reply, error) {
		if len(ms) > 0 && ms[0].Content == titlePrompt {
			titleReq = ms
			return answerText("\"Fix flaky parser tests.\"\nextra"), nil
		}
		return answerText("done"), nil
	})
	if status, err := a.Run(context.Background(), "please fix the flaky parser tests"); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	meta, _ := session.ReadMeta(a.Config.SessionDir, a.Store.ID)
	if meta.Title != "Fix flaky parser tests" || len(titleReq) != 2 || !strings.Contains(titleReq[1].Content, "flaky parser") {
		t.Fatalf("title %q req %+v", meta.Title, titleReq)
	}
	// A second run never renames, and a user title is never replaced.
	session.SetTitle(a.Config.SessionDir, a.Store.ID, "Mine")
	titleReq = nil
	a.Run(context.Background(), "next")
	if meta, _ := session.ReadMeta(a.Config.SessionDir, a.Store.ID); meta.Title != "Mine" || titleReq != nil {
		t.Fatal(meta.Title, titleReq)
	}
}

func TestAutoTitleOffByDefault(t *testing.T) {
	a, _ := testAgent(t)
	calls := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		calls++
		return answerText("done"), nil
	})
	a.Run(context.Background(), "hello")
	if calls != 1 {
		t.Fatalf("%d model calls; auto title must be opt-in", calls)
	}
}

func answerText(text string) core.Reply {
	r := answer()
	r.Message.Content = text
	return r
}
