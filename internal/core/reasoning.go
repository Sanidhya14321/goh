package core

import "context"

type reasoningKey struct{}

// WithReasoning attaches a sink for streamed reasoning text (Responses
// reasoning summaries, Anthropic thinking). Providers call it as text
// arrives; it is display-only and never changes the reply.
func WithReasoning(ctx context.Context, sink func(string)) context.Context {
	return context.WithValue(ctx, reasoningKey{}, sink)
}

// ReasoningSink returns the attached sink, or nil.
func ReasoningSink(ctx context.Context) func(string) {
	sink, _ := ctx.Value(reasoningKey{}).(func(string))
	return sink
}

type retryKey struct{}

// RetryNotice describes a provider request about to be retried.
type RetryNotice struct {
	Attempt int    `json:"attempt"` // the attempt that will run next (2 or 3)
	Max     int    `json:"max"`
	DelayMS int64  `json:"delay_ms"`
	Reason  string `json:"reason"`
}

// WithRetryNotices attaches a callback for provider retries (display only).
func WithRetryNotices(ctx context.Context, notify func(RetryNotice)) context.Context {
	return context.WithValue(ctx, retryKey{}, notify)
}

// RetryNotifier returns the attached callback, or nil.
func RetryNotifier(ctx context.Context) func(RetryNotice) {
	notify, _ := ctx.Value(retryKey{}).(func(RetryNotice))
	return notify
}

type toolProgressKey struct{}

// ToolProgress reports a tool call while the model is still writing it, so
// an interface can show "Writing index.html · 1.8k tokens" instead of a bare
// spinner during a long reply. Display only.
type ToolProgress struct {
	Tool string `json:"tool"`
	// Target is a path or command seen in the arguments so far, if any.
	Target string `json:"target,omitempty"`
	// Chars counts the argument characters received so far.
	Chars int `json:"chars"`
}

// WithToolProgress attaches a callback for tool calls being written.
func WithToolProgress(ctx context.Context, notify func(ToolProgress)) context.Context {
	return context.WithValue(ctx, toolProgressKey{}, notify)
}

// ToolProgressSink returns the attached callback, or nil.
func ToolProgressSink(ctx context.Context) func(ToolProgress) {
	notify, _ := ctx.Value(toolProgressKey{}).(func(ToolProgress))
	return notify
}
