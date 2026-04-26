package observability

import (
	"context"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"

	"github.com/manuelarte/go-web-layout/internal/info"
)

//nolint:wrapcheck // will be wrapped by the caller
func createResource(ctx context.Context, hostname string) (*resource.Resource, error) {
	return resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(info.AppName),
			semconv.ServiceVersionKey.String(info.Version),
			semconv.HostNameKey.String(hostname),
		),
	)
}
