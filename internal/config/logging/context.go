package logging

import (
	"context"
	"log/slog"
)

type loggerKey struct{}

// FromContext returns the slog.Logger from the context.
// If no logger is found, it returns the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}

	return slog.Default()
}

// withLogger returns a new context with the given logger attached.
func withLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}
