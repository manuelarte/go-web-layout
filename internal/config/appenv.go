package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
)

// AppEnv contains the application environment variables.
type AppEnv struct {
	// keep-sorted start
	// Env is the application environment.
	Env string `env:"ENV" envDefault:"local"`
	// GRPCServeAddress is the address to run the gRPC server.
	GRPCServeAddress string `env:"GRPC_SERVE_ADDRESS" envDefault:":3002"`
	// HTTPServeAddress is the address to run the HTTP server.
	HTTPServeAddress string `env:"HTTP_SERVE_ADDRESS" envDefault:":3001"`
	// Hostname is the hostname of the server.
	Hostname string `env:"HOSTNAME"`
	// OtelExporterEndpoint address for the OpenTelemetry exporter.
	OtelExporterEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	// ServerID is the server id.
	ServerID string `env:"SERVER_ID" envDefault:"local"`
	// keep-sorted end
}

func GetAppEnv() (AppEnv, error) {
	cfg, err := env.ParseAs[AppEnv]()
	if err != nil {
		return AppEnv{}, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	if cfg.Hostname == "" {
		hostname, errHostname := os.Hostname()
		if errHostname != nil {
			return AppEnv{}, fmt.Errorf("failed to get hostname: %w", errHostname)
		}

		cfg.Hostname = hostname
	}

	return cfg, nil
}
