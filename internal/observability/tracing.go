package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// InitTracerProvider initializes the tracer provider.
func InitTracerProvider(ctx context.Context, exporterURL string, hostname string) (*sdktrace.TracerProvider, error) {
	var (
		exporter sdktrace.SpanExporter
		err      error
	)

	if exporterURL == "" {
		exporter, err = stdout.New(stdout.WithPrettyPrint())
	} else {
		exporter, err = otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithEndpoint(exporterURL),
			otlptracegrpc.WithInsecure(),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize exporter: %w", err)
	}

	res, err := createResource(ctx, hostname)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize resource: %w", err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	), nil
}
