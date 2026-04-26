package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/sdk/log"
)

func InitLoggingProvider(ctx context.Context, exporterURL string) (*log.LoggerProvider, error) {
	var (
		exporter log.Exporter
		err      error
	)
	if exporterURL == "" {
		exporter, err = stdoutlog.New(stdoutlog.WithPrettyPrint())
	} else {
		exporter, err = otlploggrpc.New(
			ctx,
			otlploggrpc.WithEndpoint(exporterURL),
			otlploggrpc.WithInsecure(),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize exporter: %w", err)
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter)),
	)
	return loggerProvider, nil
}
