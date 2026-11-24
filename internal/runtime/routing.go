package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
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
	// streamed is the text this attempt has shown, so the final reply can be
	// reconciled with it below.
	var streamed strings.Builder
	attempt := a.phases.requestStarted(purpose)
	shown := delta
	if delta != nil {
		shown = func(s string) { streamed.WriteString(s); a.phases.firstText(attempt); delta(s) }
	}
	reply, served, err := a.completeRoutes(ctx, purpose, primary, primaryConfig, history, defs, shown)
	a.servedBy = served
	retryable := func(retry int) bool {
		if err == nil || ctx.Err() != nil {
			return false
		}
		if errors.Is(err, provider.ErrInterrupted) {
			return retry <= a.Config.RetryInterruptedStreams
		}
		if errors.Is(err, provider.ErrBusy) {
			return retry <= busyRetries
		}
		// A reply the provider refused mid-generation (an invalid tool call)
		// executed nothing; a new sample usually succeeds.
		return errors.Is(err, provider.ErrRejectedOutput) && retry <= 2
	}
	for retry := 1; retryable(retry); retry++ {
		// The failed attempt ends here; a pause before the next one is
		// counted on the next attempt, not as provider time.
		a.phases.requestEnded(attempt, "", served != "", true)
		var retryWait time.Duration
		if e := a.tokenBudgetErr(); e != nil {
			// The discarded attempt's reported usage already spent the run's
			// budget: another request would only exceed it further.
			return core.Reply{}, e
		}
		busy := errors.Is(err, provider.ErrBusy)
		if busy {
			// An overloaded service needs a pause, not an immediate resend.
			wait := busyWait(retry)
			a.event("provider_retry", core.RetryNotice{Attempt: retry + 1, Max: busyRetries + 1, DelayMS: wait.Milliseconds(), Reason: "overloaded"})
			waitStart := time.Now()
			select {
			case <-ctx.Done():
				return core.Reply{}, ctx.Err()
			case <-time.After(wait):
			}
			retryWait = time.Since(waitStart)
		}
		// Nothing from the broken reply was committed or executed, so this
		// is a fresh request on the same history. The event tells interfaces
		// to drop the partial text already shown.
		a.event("stream_retry", map[string]any{"attempt": retry, "error": err.Error(), "busy": busy})
		streamed.Reset()
		attempt = a.phases.requestStarted(purpose)
		a.phases.retryWaited(attempt, retryWait)
		reply, served, err = a.completeRoutes(ctx, purpose, primary, primaryConfig, history, defs, shown)
		a.servedBy = served
	}
	if model := reply.Model; model != "" || err != nil {
		a.phases.requestEnded(attempt, model, served != "", err != nil)
	} else {
		a.phases.requestEnded(attempt, primaryConfig.Model, served != "", false)
	}
	if err != nil {
		return reply, err
	}
	if delta != nil && reply.Message.Content != streamed.String() {
		// Interfaces show what streamed; the journal keeps the final reply.
		// Send what the stream left out, or replace the text when the final
		// reply does not continue it.
		if rest, ok := strings.CutPrefix(reply.Message.Content, streamed.String()); ok {
			delta(rest)
		} else {
			a.event("assistant_reset", map[string]any{"reason": "the final reply differs from the streamed text"})
			if reply.Message.Content != "" {
				delta(reply.Message.Content)
			}
		}
	}
	// Calibrate from the primary route only: a fallback may share the model
	// name at another endpoint with a different tokenizer or accounting.
	if purpose == "turn" && served == "" && reply.Usage != nil && (reply.Model == "" || reply.Model == primaryConfig.Model) {
		size, images := requestBytes(history, defs)
		a.calibrate(size, reply.Usage.Prompt-images*imageTokenEstimate)
	}
	if reply.Model == "" {
		reply.Model = primaryConfig.Model
	}
	if reply.Usage == nil {
		// The provider reported nothing: token budgets cannot count this
		// call, which run_finished says (usage_missing_calls).
		a.usageMissing++
	}
	if e := a.recordUsage(purpose, reply.Model, reply.Usage, false); e != nil {
		return core.Reply{}, e
	}
	return reply, nil
}

// recordUsage journals durable usage for one model attempt (turns,
// summaries, titles, children), charged to the model that served it and
// never replayed. discarded marks an attempt whose reply was not accepted
// (an interrupted stream, a refused tool call): billed work, not history.
func (a *Agent) recordUsage(purpose, model string, u *core.Usage, discarded bool) error {
	if u == nil {
		return nil
	}
	a.runTokens += max(u.Total, u.Prompt+u.Completion)
	d := map[string]any{"model": model, "purpose": purpose, "prompt_tokens": u.Prompt, "completion_tokens": u.Completion, "total_tokens": u.Total, "cache_read_tokens": u.CacheRead, "cache_write_tokens": u.CacheWrite}
	if discarded {
		d["discarded"] = true
	}
	return a.Store.Append("usage", d)
}

// chargeFailed accounts for an attempt that ended in err, before any retry
// or fallback is decided. Reported usage is charged; a reply that was being
// generated (an interrupted stream, a refused output) without usage counts
// as missing usage. A failure before any response used nothing.
func (a *Agent) chargeFailed(purpose, model string, r core.Reply, err error) error {
	if err == nil {
		return nil
	}
	if r.Model != "" {
		model = r.Model
	}
	if r.Usage == nil {
		if errors.Is(err, provider.ErrInterrupted) || errors.Is(err, provider.ErrRejectedOutput) {
			a.usageMissing++
		}
		return nil
	}
	return a.recordUsage(purpose, model, r.Usage, true)
}

// errRunTokens marks a run stopped at its max_run_tokens limit.
var errRunTokens = errors.New("run stopped at the max_run_tokens limit")

// tokenBudgetErr reports whether the run's observed usage has reached
// max_run_tokens. The limit is checked at request boundaries, so a single
// request can finish above it; no further request is sent once it is hit.
func (a *Agent) tokenBudgetErr() error {
	if a.Config.MaxRunTokens > 0 && a.runTokens >= a.Config.MaxRunTokens {
		return fmt.Errorf("%w (%d used of %d)", errRunTokens, a.runTokens, a.Config.MaxRunTokens)
	}
	return nil
}

// completeRoutes returns the reply or error of the route that answered last
// and that route: the fallback profile name, or "" for the primary.
func (a *Agent) completeRoutes(ctx context.Context, purpose string, primary Model, primaryConfig core.Config, history []core.Message, defs []core.Tool, delta func(string)) (core.Reply, string, error) {
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
		if e := a.chargeFailed(purpose, primaryConfig.Model, reply, err); e != nil {
			return core.Reply{}, "", e
		}
		if err == nil || ctx.Err() != nil || !provider.Failoverable(err) {
			return reply, "", err
		}
		a.startCooldown(from)
	}

	// Fallbacks in order, each once, never the primary again; fallbacks in
	// their own cooldown wait until every fresh route has failed.
	primaryName := primaryConfig.Profile
	seen := map[string]bool{cooldownKey(primaryName): true}
	var order, cooled []string
	for _, name := range a.Config.FallbackProfiles {
		if seen[cooldownKey(name)] {
			continue
		}
		seen[cooldownKey(name)] = true
		if a.coolingDown(name) {
			cooled = append(cooled, name)
			continue
		}
		order = append(order, name)
	}
	for _, name := range cooled {
		a.recordFailover(purpose, from, name, "cooldown", "skipped", "fallback is cooling down after a recent failure")
	}
	try := func(name string) (core.Reply, error, bool) {
		reason := failoverReason(err)
		config, cfgErr := a.Config.WithFallbackProfile(name)
		if cfgErr != nil {
			a.recordFailover(purpose, from, name, reason, "skipped", cfgErr.Error())
			return core.Reply{}, nil, false
		}
		if a.estimate(history, defs) > config.ContextWindow-config.OutputBudget {
			a.recordFailover(purpose, from, name, reason, "skipped", "conversation exceeds fallback context window")
			return core.Reply{}, nil, false
		}
		if limit := a.fallbackCaps[cooldownKey(name)]; limit > 0 && a.estimate(history, defs) > limit-min(config.OutputBudget, max(1024, limit/4)) {
			a.recordFailover(purpose, from, name, reason, "skipped", "conversation exceeds the request cap this fallback stated")
			return core.Reply{}, nil, false
		}
		model := Model(provider.New(config))
		if a.RoleModelFactory != nil {
			if model = a.RoleModelFactory(config, "fallback"); model == nil {
				a.recordFailover(purpose, from, name, reason, "skipped", "fallback factory returned no provider")
				return core.Reply{}, nil, false
			}
		}
		if e := a.tokenBudgetErr(); e != nil {
			return core.Reply{}, e, true
		}
		if e := a.recordFailover(purpose, from, name, reason, "attempting", ""); e != nil {
			return core.Reply{}, e, true
		}
		r, e := model.Complete(ctx, history, defs, delta)
		if ce := a.chargeFailed(purpose, config.Model, r, e); ce != nil {
			return core.Reply{}, ce, true
		}
		if e == nil {
			r.Model = config.Model // usage is charged to the model that served it
			data := map[string]any{"purpose": purpose, "model": config.Model, "profile": name, "protocol": config.EffectiveProtocol(), "context_window": config.ContextWindow, "failover": true, "primary": primaryName}
			if e := a.Store.Append("model_route", data); e != nil {
				return core.Reply{}, e, true
			}
			a.event("model_route", data)
			return r, nil, true
		}
		var h *provider.HTTPError
		if errors.As(e, &h) && h.TooLarge() && h.Limit > 0 {
			// The cap belongs to this fallback; the primary keeps its own.
			if a.fallbackCaps == nil {
				a.fallbackCaps = map[string]int{}
			}
			a.fallbackCaps[cooldownKey(name)] = h.Limit
			e = fmt.Errorf("fallback profile %s refused the request: %w", name, e)
		}
		reply, err = r, e
		if ctx.Err() != nil || !provider.Failoverable(e) {
			return r, e, true
		}
		a.startCooldown(name)
		from = name
		return core.Reply{}, nil, false
	}
	for _, name := range order {
		if r, e, done := try(name); done {
			return r, name, e
		}
	}
	if !cooling && ctx.Err() == nil {
		// Every fresh route failed: cooling fallbacks may have recovered.
		for _, name := range cooled {
			if r, e, done := try(name); done {
				return r, name, e
			}
		}
	}
	if cooling && ctx.Err() == nil {
		// Last resort: the primary may have recovered.
		if e := a.tokenBudgetErr(); e != nil {
			return core.Reply{}, "", e
		}
		reply, err = primary.Complete(ctx, history, defs, delta)
		if e := a.chargeFailed(purpose, primaryConfig.Model, reply, err); e != nil {
			return core.Reply{}, "", e
		}
		if err != nil && provider.Failoverable(err) {
			a.startCooldown(primaryConfig.Profile)
		}
		return reply, "", err
	}
	// Every route failed; err is the last one's (from names it).
	if from == primaryConfig.Profile {
		from = ""
	}
	return reply, from, err
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

// busyRetries is how often a turn is resent after the provider reported
// itself overloaded inside the stream; busyWait is the pause before each.
const busyRetries = 3

var busyWait = func(retry int) time.Duration { return time.Duration(5<<(retry-1)) * time.Second } // 5, 10, 20 s
