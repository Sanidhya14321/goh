package runtime

import (
	"context"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
)

// Handoff P1: a fallback's request cap never teaches the primary. The run
// stops with the refusal instead of resending the same oversized request,
// and later turns skip that fallback while the request exceeds its cap.
func TestFallbackRefusalPreservesPrimaryWindow(t *testing.T) {
	a := failoverAgent(t, "backup")
	a.Config.ContextWindow = 1048576
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return core.Reply{}, provider.ErrConnection
	})
	fallbackCalls := 0
	a.RoleModelFactory = func(core.Config, string) Model {
		return modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
			fallbackCalls++
			return core.Reply{}, &provider.HTTPError{Status: 413, Kind: "request_too_large", Limit: 50000, Requested: 60000}
		})
	}
	status, err := a.Run(context.Background(), "hello")
	if status == "completed" || err == nil {
		t.Fatal(status, err)
	}
	if a.Config.ContextWindow != 1048576 {
		t.Fatalf("primary window taught by fallback: %d", a.Config.ContextWindow)
	}
	if fallbackCalls != 1 {
		t.Fatalf("same oversized fallback request sent %d times", fallbackCalls)
	}
}

// A primary refusal still teaches the primary (existing behaviour).
func TestPrimaryRefusalTeachesPrimary(t *testing.T) {
	a := failoverAgent(t)
	a.Config.ContextWindow = 1048576
	calls := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		calls++
		if calls == 1 {
			return core.Reply{}, &provider.HTTPError{Status: 413, Kind: "request_too_large", Limit: 50000, Requested: 60000}
		}
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "hello"); status != "completed" {
		t.Fatal(status, err)
	}
	if a.Config.ContextWindow != 50000 || calls != 2 {
		t.Fatalf("window %d calls %d", a.Config.ContextWindow, calls)
	}
}

// Two routes sharing one model string at different endpoints: fallback
// usage never calibrates the primary's token estimate, and a later primary
// turn still calibrates from its own usage.
func TestSameModelFallbackDoesNotCalibratePrimary(t *testing.T) {
	a := failoverAgent(t, "backup")
	shared := a.Config.Model
	backupURL := "http://backup.invalid/v1/chat/completions"
	a.Config.Profiles = map[string]core.ModelProfile{"backup": {Model: &shared, Endpoint: &backupURL}}
	primaryUp := false
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		if !primaryUp {
			return core.Reply{}, provider.ErrConnection
		}
		r := answer()
		r.Usage = &core.Usage{Prompt: 4000, Completion: 10, Total: 4010}
		return r, nil
	})
	a.RoleModelFactory = func(core.Config, string) Model {
		return modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
			r := answer()
			r.Usage = &core.Usage{Prompt: 10, Completion: 10, Total: 20}
			return r, nil
		})
	}
	if status, err := a.Run(context.Background(), "hello"); status != "completed" {
		t.Fatal(status, err)
	}
	if a.bytesPerToken != 0 || a.anchor.tokens != 0 {
		t.Fatalf("fallback usage calibrated primary: ratio %v anchor %+v", a.bytesPerToken, a.anchor)
	}
	primaryUp = true
	a.cooldowns = nil
	if status, err := a.Run(context.Background(), "again"); status != "completed" {
		t.Fatal(status, err)
	}
	if a.anchor.tokens != 4000 {
		t.Fatalf("primary turn did not calibrate: %+v", a.anchor)
	}
}
