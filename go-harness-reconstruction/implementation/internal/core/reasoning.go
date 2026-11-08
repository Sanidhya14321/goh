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
