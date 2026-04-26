package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/manuelarte/go-web-layout/internal/info"
)

//nolint:gochecknoglobals // Context key used for tracing.
var contextKey = key{}

// Context tracing value to be passed through the stack trace through [context.Context].
type key struct{}

func AddContext(ctx context.Context, tracer trace.Tracer) context.Context {
	return context.WithValue(ctx, contextKey, tracer)
}

func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := getOrNewTracer(ctx)

	return tracer.Start(ctx, name, opts...)
}

func getOrNewTracer(ctx context.Context) trace.Tracer {
	previous := ctx.Value(contextKey)
	if previous != nil {
		//nolint:errcheck // it should always be ok
		return previous.(trace.Tracer)
	}

	return otel.Tracer(info.AppName)
}
