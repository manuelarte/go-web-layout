package config

// AppEnv contains the application environment variables.
type AppEnv struct {
	GRPCServeAddress string `env:"GRPC_SERVE_ADDRESS" envDefault:":3002"`
	HTTPServeAddress string `env:"HTTP_SERVE_ADDRESS" envDefault:":3001"`
}
