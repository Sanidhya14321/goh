package runtime

import (
	"context"
	"strings"
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

// Review P2: a fallback that failed is skipped while it cools down, in favour
// of the next fresh fallback.
func TestFailedFallbackCoolsDown(t *testing.T) {
	a := failoverAgent(t, "backup", "other")
	now := time.Unix(1000, 0)
	a.clock = func() time.Time { return now }
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return core.Reply{}, &provider.HTTPError{Status: 503, Kind: "server", Retryable: true}
	})
	var routed []string
	a.RoleModelFactory = func(c core.Config, _ string) Model {
		return modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
			routed = append(routed, c.Profile)
			if c.Profile == "backup" {
				return core.Reply{}, &provider.HTTPError{Status: 503, Kind: "server", Retryable: true}
			}
			return answer(), nil
		})
	}
	if s, err := a.Run(context.Background(), "one"); s != "completed" {
		for _, r := range a.Store.Records() {
			if r.Type == "provider_failover" || r.Type == "failover" {
				t.Log(string(r.Data))
			}
		}
		t.Fatal(s, err)
	}
	a.Run(context.Background(), "two")
	got := strings.Join(routed, ",")
	if got != "backup,other,other" {
		t.Fatalf("routes %s", got)
	}
}
