package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

//nolint:gochecknoglobals // Context key used for tracing.
var contextKey = key{}

// Context tracing value to be passed through the stack trace through [context.Context].
type key struct{}

func AddContext(ctx context.Context, tracer trace.Tracer) context.Context {
	return context.WithValue(ctx, contextKey, tracer)
}
