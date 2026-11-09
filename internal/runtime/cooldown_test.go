package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
)

func TestFailedPrimaryCoolsDownWhileFallbacksExist(t *testing.T) {
	a := failoverAgent(t, "backup")
	now := time.Unix(1000, 0)
	a.clock = func() time.Time { return now }
	primaryCalls, backupFails := 0, false
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		primaryCalls++
		return core.Reply{}, &provider.HTTPError{Status: 429, Kind: "rate_limit", Retryable: true}
	})
	a.RoleModelFactory = func(core.Config, string) Model {
		return modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
			if backupFails {
				return core.Reply{}, &provider.HTTPError{Status: 503, Kind: "server", Retryable: true}
			}
			return answer(), nil
		})
	}
	if s, err := a.Run(context.Background(), "one"); s != "completed" || err != nil || primaryCalls != 1 {
		t.Fatal(s, err, primaryCalls)
	}
	// Within the cooldown the primary is skipped entirely.
	if s, err := a.Run(context.Background(), "two"); s != "completed" || err != nil || primaryCalls != 1 {
		t.Fatal("primary not skipped during cooldown", s, err, primaryCalls)
	}
	// If every fallback fails, the cooling primary is still tried last.
	backupFails = true
	a.Run(context.Background(), "three")
	if primaryCalls != 2 {
		t.Fatal("primary not tried as last resort", primaryCalls)
	}
	// After the cooldown it is tried first again.
	now = now.Add(2 * time.Minute)
	backupFails = false
	a.Run(context.Background(), "four")
	if primaryCalls != 3 {
		t.Fatal("cooldown never expired", primaryCalls)
	}
}
