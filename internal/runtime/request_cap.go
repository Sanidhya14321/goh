package runtime

import (
	"errors"
	"math"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
)

// Token estimates start at one token per request byte, which is safe for any
// tokenizer but about four times too high for English text, code and JSON.
// On a small window (or a provider's small per-request cap) that alone made
// goh's own tools look larger than the window. Once the provider reports how
// many prompt tokens a request really used, the estimate follows that ratio,
// with a margin, between one and four bytes per token.
const (
	calibrationMargin = 0.85
	maxBytesPerToken  = 4.0
	maxCapAdapts      = 2
)

// estimate is the conservative token estimate for a request to this agent's
// model: calibrated bytes plus the fixed image allowance.
func (a *Agent) estimate(ms []core.Message, defs []core.Tool) int {
	size, images := requestBytes(ms, defs)
	if len(defs) == 0 {
		// Not a turn request (a summary): the anchor's instructions and
		// tools are not in it, so every byte counts at the content ratio.
		return a.contentTokens(size) + images*imageTokenEstimate
	}
	return a.tokensFor(size) + images*imageTokenEstimate
}

// contentTokens estimates bytes of conversation content alone.
func (a *Agent) contentTokens(size int) int { return int(math.Ceil(float64(size) / a.ratio())) }

func (a *Agent) ratio() float64 {
	if a.bytesPerToken < 1 {
		return 1
	}
	return a.bytesPerToken
}

// tokensFor estimates a request of size bytes. Small requests are mostly
// goh's instructions and tool definitions, which tokenize far denser than,
// say, a fetched page of checksums; one ratio for the whole request then
// overstates either the fixed part or the content (found live on an 8k
// cap). So the smallest request the provider has counted exactly is the
// anchor, and only bytes beyond it are converted at the learned ratio.
func (a *Agent) tokensFor(size int) int {
	if a.anchorValid() && size >= a.anchor.bytes {
		return int(math.Ceil(float64(a.anchor.tokens)/calibrationMargin)) + int(math.Ceil(float64(size-a.anchor.bytes)/a.ratio()))
	}
	return a.contentTokens(size)
}
func (a *Agent) bytesFor(tokens int) int { return int(float64(tokens) * a.ratio()) }
func (a *Agent) anchorValid() bool {
	return a.anchor.bytes > 0 && a.anchor.model == a.Config.Model
}

// tokenAnchor is the provider's own prompt-token count for one request.
type tokenAnchor struct {
	model         string
	bytes, tokens int
}

// minRatioSample is the least content beyond the anchor that a ratio is
// learned from; smaller differences are mostly noise.
const minRatioSample = 2048

// calibrate records that size request bytes used promptTokens tokens: a new
// anchor when it is the smallest request seen, otherwise the ratio of the
// content beyond the anchor. The latest ratio wins, since a conversation's
// mix of prose, code and data changes; the margin covers the drift.
func (a *Agent) calibrate(size, promptTokens int) {
	if size <= 0 || promptTokens <= 0 {
		return
	}
	switch {
	case !a.anchorValid() || size < a.anchor.bytes:
		a.anchor = tokenAnchor{model: a.Config.Model, bytes: size, tokens: promptTokens}
		if a.bytesPerToken == 0 {
			a.bytesPerToken = clampRatio(float64(size) / float64(promptTokens) * calibrationMargin)
		}
	case size-a.anchor.bytes >= minRatioSample && promptTokens > a.anchor.tokens:
		a.bytesPerToken = clampRatio(float64(size-a.anchor.bytes) / float64(promptTokens-a.anchor.tokens) * calibrationMargin)
	}
}
func clampRatio(r float64) float64 { return min(max(r, 1), maxBytesPerToken) }

// adaptToRequestCap handles a provider refusing a request as too large
// while stating its per-request token cap (HTTP 413). Nothing was generated
// or executed, so the run can shrink its context to the cap and resend. It
// reports whether the request should be resent.
func (a *Agent) adaptToRequestCap(err error, history []core.Message, defs []core.Tool) bool {
	var h *provider.HTTPError
	if !errors.As(err, &h) || !h.TooLarge() || h.Limit <= 0 || a.capAdapts >= maxCapAdapts {
		return false
	}
	window := h.Limit
	if window >= a.Config.ContextWindow {
		// Already at or under the cap: retrying only helps if the size
		// estimate was too low, which the stated request size can correct.
		if h.Requested <= 0 {
			return false
		}
		window = a.Config.ContextWindow
	}
	a.capAdapts++
	// Providers that state a cap this way (Groq's token-per-minute limit)
	// count the requested output tokens too, so the output reserve shrinks
	// with the window.
	output := min(a.Config.OutputBudget, max(1024, window/4))
	if h.Requested > 0 {
		// The refused request may count the output reserve it asked for;
		// counting all of it as input keeps the ratio conservative. Bytes
		// beyond the last exactly counted request are what the ratio is
		// for, so only those are measured when such a count exists.
		size, _ := requestBytes(history, defs)
		bytes, tokens := size, h.Requested
		if a.anchorValid() && size > a.anchor.bytes && h.Requested > a.anchor.tokens {
			bytes, tokens = size-a.anchor.bytes, h.Requested-a.anchor.tokens
		}
		observed := float64(bytes) / float64(tokens)
		if a.bytesPerToken == 0 || observed < a.bytesPerToken {
			a.bytesPerToken = clampRatio(observed)
		}
	}
	a.event("request_cap_learned", map[string]any{"limit": h.Limit, "requested": h.Requested, "context_window": window, "output_budget": output, "previous_context_window": a.Config.ContextWindow})
	a.Config.ContextWindow, a.Config.OutputBudget = window, output
	if c, stock := a.Provider.(*provider.Client); stock {
		next := *c
		next.Config.ContextWindow, next.Config.OutputBudget = window, output
		a.Provider = &next
	}
	return true
}
