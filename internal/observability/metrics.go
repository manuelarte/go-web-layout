package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// InitMeterProvider initializes the meter provider.
func InitMeterProvider(ctx context.Context, exporterURL string) (*sdkmetric.MeterProvider, error) {
	var (
		exporter sdkmetric.Exporter
		err      error
	)
	if exporterURL == "" {
		exporter, err = stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	} else {
		exporter, err = otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithEndpoint(exporterURL),
			otlpmetricgrpc.WithInsecure(),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize exporter: %w", err)
	}

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	), nil
}
