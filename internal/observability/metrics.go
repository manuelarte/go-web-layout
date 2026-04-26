package observability

import (
	"fmt"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// InitMeterProvider initializes the meter provider.
func InitMeterProvider() (*sdkmetric.MeterProvider, error) {
	exp, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("can't initialize metrics: %w", err)
	}

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)),
	), nil
}
