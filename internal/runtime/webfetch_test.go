package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

func TestWebfetchRuntimeAdmissionAndPlanBoundary(t *testing.T) {
	for _, mode := range []string{"headless", "deny", "granted", "plan"} {
		t.Run(mode, func(t *testing.T) {
			requests := atomic.Int32{}
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte("Runtime local content"))
			}))
			defer s.Close()
			a, _ := testAgent(t)
			if mode == "granted" || mode == "plan" {
				a.Grants["webfetch"] = true
			}
			if mode == "plan" {
				a.Config.AgentMode = "plan"
			}
			if mode == "deny" {
				a.Approve = func(context.Context, core.Permission) (core.Decision, error) { return core.Decision{}, nil }
			}
			turn := 0
			a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
				turn++
				if turn == 1 {
					return answer(toolCall("fetch", "webfetch", map[string]any{"url": s.URL, "format": "text"})), nil
				}
				if mode == "granted" {
					out := toolResult(t, h[len(h)-1])
					if out["content"] != "Runtime local content" {
						t.Fatal(out)
					}
				}
				return answer(), nil
			})
			status, e := a.Run(context.Background(), "read local webpage")
			if mode == "headless" {
				if status != "needs_permission" || e == nil {
					t.Fatal(status, e)
				}
			} else if status != "completed" || e != nil {
				t.Fatal(status, e)
			}
			want := int32(0)
			if mode == "granted" {
				want = 1
			}
			if requests.Load() != want {
				t.Fatal("unapproved or missing network request", requests.Load(), want)
			}
		})
	}
}
