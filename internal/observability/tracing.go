package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"

	"github.com/manuelarte/go-web-layout/internal/info"
)

// InitTracerProvider initializes the tracer provider.
func InitTracerProvider(ctx context.Context, exporterURL, hostname string) (*sdktrace.TracerProvider, error) {
	var (
		exporter sdktrace.SpanExporter
		err      error
	)

	if exporterURL == "" {
		exporter, err = stdout.New(stdout.WithPrettyPrint())
	} else {
		exporter, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(exporterURL),
			otlptracegrpc.WithInsecure(),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize exporter: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(info.AppName),
			semconv.ServiceVersionKey.String(info.Version),
			semconv.HostNameKey.String(hostname),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize resource: %w", err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	), nil
}
