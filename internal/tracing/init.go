// Package tracing provides tracing functionality.
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/manuelarte/go-web-layout/internal/info"
)

func InitMeter() (*sdkmetric.MeterProvider, error) {
	exp, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("can't initialize metrics: %w", err)
	}

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)),
	), nil
}

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

func StartSpan(ctx context.Context, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	tracer := getOrNewTracer(ctx)

	return tracer.Start(ctx, name, opts...)
}

func getOrNewTracer(ctx context.Context) oteltrace.Tracer {
	previous := ctx.Value(contextKey)
	if previous != nil {
		//nolint:errcheck // it should always be ok
		return previous.(oteltrace.Tracer)
	}

	return otel.Tracer(info.AppName)
}
