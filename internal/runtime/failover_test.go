package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
)

func failoverAgent(t *testing.T, fallbacks ...string) *Agent {
	t.Helper()
	a, _ := testAgent(t)
	backup, other := "backup-model", "other-model"
	a.Config.Endpoint = "http://primary.invalid/v1/chat/completions"
	a.Config.Profiles = map[string]core.ModelProfile{"backup": {Model: &backup}, "other": {Model: &other}}
	a.Config.FallbackProfiles = fallbacks
	return a
}

func TestFailoverUsesExplicitProfileAfterPreResponseFailure(t *testing.T) {
	a := failoverAgent(t, "backup")
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return core.Reply{}, &provider.HTTPError{Status: 503, Kind: "server", Retryable: true}
	})
	var routed []string
	a.RoleModelFactory = func(c core.Config, role string) Model {
		routed = append(routed, role+":"+c.Profile+":"+c.Model)
		return modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
			return answer(), nil
		})
	}
	status, err := a.Run(context.Background(), "hello")
	if err != nil || status != "completed" || len(routed) != 1 || routed[0] != "fallback:backup:backup-model" {
		t.Fatal(status, err, routed)
	}
	if got := journalText(a); !strings.Contains(got, "provider_failover") || !strings.Contains(got, `"failover":true`) {
		t.Fatal("failover not journaled", got)
	}
}

func TestFailoverRefusesUnsafeOrUnconfiguredRoutes(t *testing.T) {
	cases := map[string]struct {
		err       error
		fallbacks []string
	}{
		"authentication": {&provider.HTTPError{Status: 401, Kind: "authentication"}, []string{"backup"}},
		"partial stream": {errors.New("stream ended before completion"), []string{"backup"}},
		"no fallbacks":   {provider.ErrConnection, nil},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			a := failoverAgent(t, tc.fallbacks...)
			a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
				return core.Reply{}, tc.err
			})
			a.RoleModelFactory = func(core.Config, string) Model { t.Fatal("conversation sent to fallback"); return nil }
			if status, err := a.Run(context.Background(), "hello"); err == nil || status != "failed" {
				t.Fatal(status, err)
			}
		})
	}
}

func TestFailoverChainsAndStopsAtFirstSuccess(t *testing.T) {
	a := failoverAgent(t, "backup", "other")
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return core.Reply{}, provider.ErrConnection
	})
	var order []string
	a.RoleModelFactory = func(c core.Config, _ string) Model {
		order = append(order, c.Profile)
		return modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
			if c.Profile == "backup" {
				return core.Reply{}, &provider.HTTPError{Status: 429, Kind: "rate_limit", Retryable: true}
			}
			return answer(), nil
		})
	}
	if status, err := a.Run(context.Background(), "hello"); err != nil || status != "completed" || strings.Join(order, ",") != "backup,other" {
		t.Fatal(status, err, order)
	}
}

func TestFallbackProfilesValidation(t *testing.T) {
	c := core.Defaults()
	c.Model, c.Endpoint = "m", "http://127.0.0.1:1/v1/chat/completions"
	for _, bad := range [][]string{{"missing"}, {"a", "a"}, {"bad name"}} {
		m := "x"
		c.Profiles = map[string]core.ModelProfile{"a": {Model: &m}}
		c.FallbackProfiles = bad
		if c.Validate() == nil {
			t.Fatal("accepted", bad)
		}
	}
}

func TestFailoverOverLoopbackHTTP(t *testing.T) {
	var primaryHits int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&primaryHits, 1)
		w.WriteHeader(503)
	}))
	defer primary.Close()
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"from backup\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer backup.Close()
	a := failoverAgent(t, "backup")
	t.Setenv("FIXTURE_KEY", "k")
	endpoint, model, env := backup.URL+"/v1/chat/completions", "backup-model", "FIXTURE_KEY"
	a.Config.Endpoint, a.Config.APIKeyEnv = primary.URL+"/v1/chat/completions", "FIXTURE_KEY"
	a.Config.Profiles = map[string]core.ModelProfile{"backup": {Endpoint: &endpoint, Model: &model, APIKeyEnv: &env}}
	a.Provider = provider.New(a.Config)
	status, err := a.Run(context.Background(), "hello")
	if err != nil || status != "completed" || atomic.LoadInt32(&primaryHits) != 3 {
		t.Fatal(status, err, primaryHits)
	}
	if last := a.History[len(a.History)-1]; last.Content != "from backup" {
		t.Fatal(last)
	}
}

func journalText(a *Agent) string {
	var b strings.Builder
	for _, r := range a.Store.Records() {
		b.WriteString(fmt.Sprintf("%+v\n", r))
	}
	return b.String()
}
