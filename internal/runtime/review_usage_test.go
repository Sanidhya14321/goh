package runtime

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
	"github.com/Sanidhya14321/goh/internal/session"
)

// Review R14: usage from a fallback route is charged to the model that
// served it, with cache reads and writes kept separate.
func TestFallbackUsageAttributedToServingModel(t *testing.T) {
	a := failoverAgent(t, "backup")
	a.Config.Model = "primary-model"
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return core.Reply{}, &provider.HTTPError{Status: 503, Kind: "server", Retryable: true}
	})
	a.RoleModelFactory = func(core.Config, string) Model {
		return modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
			r := answer()
			r.Usage = &core.Usage{Prompt: 1000, Completion: 50, Total: 1050, CacheRead: 800, CacheWrite: 100}
			return r, nil
		})
	}
	if status, err := a.Run(context.Background(), "hello"); status != "completed" {
		t.Fatal(status, err)
	}
	var usage struct {
		Model     string `json:"model"`
		CacheRead int    `json:"cache_read_tokens"`
	}
	for _, r := range a.Store.Records() {
		if r.Type == "usage" {
			json.Unmarshal(r.Data, &usage)
		}
	}
	if usage.Model != "backup-model" || usage.CacheRead != 800 {
		t.Fatalf("usage recorded as %+v", usage)
	}
	st, err := session.CollectStats(a.Config.SessionDir, "", time.Time{})
	if err != nil || len(st.Models) != 1 || st.Models[0].Model != "backup-model" || st.Models[0].CacheRead != 800 || st.Models[0].CacheWrite != 100 {
		t.Fatalf("stats %+v %v", st.Models, err)
	}
	m := st.Models[0]
	m.Estimate(10, 20, 1, 12.5) // fresh 100*10 + read 800*1 + write 100*12.5 + out 50*20, per million
	if want := (100*10 + 800*1 + 100*12.5 + 50*20) / 1e6; *m.CostUSD < want-1e-12 || *m.CostUSD > want+1e-12 {
		t.Fatalf("cost %v want %v", *m.CostUSD, want)
	}
}
