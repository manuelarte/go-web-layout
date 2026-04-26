package observability

import (
	"fmt"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/sdk/log"
)

func InitLoggingProvider(exporterURL string) (*log.LoggerProvider, error) {
	var (
		exporter log.Exporter
		err      error
	)
	if exporterURL == "" {
		exporter, err = stdoutlog.New(stdoutlog.WithPrettyPrint())
	} else {
		exporter, err = otlplogsgrpc.New(otlplogsgrpc.WithEndpoint(exporterURL))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize exporter: %w", err)
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter)),
	)
	return loggerProvider, nil
}
