package runtime

import (
	"context"
	"errors"
	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/provider"
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
	reply, err := primary.Complete(ctx, history, defs, delta)
	if err == nil || ctx.Err() != nil || !provider.Failoverable(err) {
		return reply, err
	}
	from := primaryConfig.Profile
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
		from = name
	}
	return reply, err
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
