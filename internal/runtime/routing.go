package runtime

import (
	"context"
	"errors"
	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
	"time"
)

func (a *Agent) modelForRole(role string) (Model, core.Config, error) {
	if _, ok := a.Config.ModelRoles[role]; !ok {
		return a.Provider, a.Config, nil
	}
	config, err := a.Config.WithModelRole(role)
	if err != nil {
		return nil, config, err
	}
	if err = config.Validate(); err != nil {
		return nil, config, err
	}
	model := Model(provider.New(config))
	if a.RoleModelFactory != nil {
		model = a.RoleModelFactory(config, role)
		if model == nil {
			return nil, config, errors.New("model role factory returned no provider")
		}
	}
	data := map[string]any{"purpose": role, "model": config.Model, "profile": config.Profile, "protocol": config.EffectiveProtocol(), "context_window": config.ContextWindow}
	if err = a.Store.Append("model_route", data); err != nil {
		return nil, config, err
	}
	a.event("model_route", data)
	return model, config, nil
}

// completeWithFailover calls primary and, only for failures that occurred
// before any response body arrived, tries each explicitly configured fallback
// profile in order. A stream that produced text, a parse failure, an
// authentication failure or cancellation is never retried on another route.
func (a *Agent) completeWithFailover(ctx context.Context, purpose string, primary Model, primaryConfig core.Config, history []core.Message, defs []core.Tool, delta func(string)) (core.Reply, error) {
	reply, err := a.completeRoutes(ctx, purpose, primary, primaryConfig, history, defs, delta)
	for retry := 1; err != nil && errors.Is(err, provider.ErrInterrupted) && ctx.Err() == nil && retry <= a.Config.RetryInterruptedStreams; retry++ {
		// Nothing from the broken reply was committed or executed, so this
		// is a fresh request on the same history. The event tells interfaces
		// to drop the partial text already shown.
		a.event("stream_retry", map[string]any{"attempt": retry, "error": err.Error()})
		reply, err = a.completeRoutes(ctx, purpose, primary, primaryConfig, history, defs, delta)
	}
	if err != nil {
		return reply, err
	}
	if reply.Model == "" {
		reply.Model = primaryConfig.Model
	}
	if reply.Usage != nil {
		// Durable usage for every model call (turns, summaries, titles,
		// children), charged to the model that served it; never replayed.
		u := reply.Usage
		a.runTokens += max(u.Total, u.Prompt+u.Completion)
		if e := a.Store.Append("usage", map[string]any{"model": reply.Model, "purpose": purpose, "prompt_tokens": u.Prompt, "completion_tokens": u.Completion, "total_tokens": u.Total, "cache_read_tokens": u.CacheRead, "cache_write_tokens": u.CacheWrite}); e != nil {
			return core.Reply{}, e
		}
	}
	return reply, nil
}

func (a *Agent) completeRoutes(ctx context.Context, purpose string, primary Model, primaryConfig core.Config, history []core.Message, defs []core.Tool, delta func(string)) (core.Reply, error) {
	from := primaryConfig.Profile
	var reply core.Reply
	var err error
	cooling := len(a.Config.FallbackProfiles) > 0 && a.coolingDown(from)
	if cooling {
		// The primary failed recently: go to fallbacks first instead of
		// waiting on it again, and use it only if none of them answers.
		a.recordFailover(purpose, from, "", "cooldown", "skipped", "primary is cooling down after a recent failure")
		err = provider.ErrConnection
	} else {
		reply, err = primary.Complete(ctx, history, defs, delta)
		if err == nil || ctx.Err() != nil || !provider.Failoverable(err) {
			return reply, err
		}
		a.startCooldown(from)
	}

	for _, name := range a.Config.FallbackProfiles {
		if name == from {
			continue
		}
		reason := failoverReason(err)
		config, cfgErr := a.Config.WithFallbackProfile(name)
		if cfgErr != nil {
			a.recordFailover(purpose, from, name, reason, "skipped", cfgErr.Error())
			continue
		}
		if estimate(history, defs) > config.ContextWindow-config.OutputBudget {
			a.recordFailover(purpose, from, name, reason, "skipped", "conversation exceeds fallback context window")
			continue
		}
		model := Model(provider.New(config))
		if a.RoleModelFactory != nil {
			if model = a.RoleModelFactory(config, "fallback"); model == nil {
				a.recordFailover(purpose, from, name, reason, "skipped", "fallback factory returned no provider")
				continue
			}
		}
		if e := a.recordFailover(purpose, from, name, reason, "attempting", ""); e != nil {
			return core.Reply{}, e
		}
		reply, err = model.Complete(ctx, history, defs, delta)
		if err == nil {
			reply.Model = config.Model // usage is charged to the model that served it
			data := map[string]any{"purpose": purpose, "model": config.Model, "profile": name, "protocol": config.EffectiveProtocol(), "context_window": config.ContextWindow, "failover": true}
			if e := a.Store.Append("model_route", data); e != nil {
				return core.Reply{}, e
			}
			a.event("model_route", data)
			return reply, nil
		}
		if ctx.Err() != nil || !provider.Failoverable(err) {
			return reply, err
		}
		a.startCooldown(name)
		from = name
	}
	if cooling && ctx.Err() == nil {
		// Last resort: the primary may have recovered.
		reply, err = primary.Complete(ctx, history, defs, delta)
		if err != nil && provider.Failoverable(err) {
			a.startCooldown(primaryConfig.Profile)
		}
	}
	return reply, err
}

// providerCooldown is how long a failed route is skipped while fallbacks exist.
const providerCooldown = time.Minute

func cooldownKey(profile string) string {
	if profile == "" {
		return "default"
	}
	return profile
}

func (a *Agent) startCooldown(profile string) {
	if a.cooldowns == nil {
		a.cooldowns = map[string]time.Time{}
	}
	a.cooldowns[cooldownKey(profile)] = a.now().Add(providerCooldown)
}

func (a *Agent) coolingDown(profile string) bool {
	return a.now().Before(a.cooldowns[cooldownKey(profile)])
}

func (a *Agent) now() time.Time {
	if a.clock != nil {
		return a.clock()
	}
	return time.Now()
}

func failoverReason(err error) string {
	var h *provider.HTTPError
	if errors.As(err, &h) {
		return h.Kind
	}
	return "connection"
}

// recordFailover journals routing decisions without provider bodies or keys.
func (a *Agent) recordFailover(purpose, from, to, reason, state, detail string) error {
	if from == "" {
		from = "default"
	}
	data := map[string]any{"purpose": purpose, "from": from, "to": to, "reason": reason, "state": state}
	if detail != "" {
		data["detail"] = detail
	}
	a.event("provider_failover", data)
	return a.Store.Append("provider_failover", data)
}
