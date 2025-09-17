package config

// AppEnv contains the application environment variables.
type AppEnv struct {
	// ENV is the application environment.
	ENV string `env:"ENV" envDefault:"local"`
	// GRPCServeAddress is the address to run the gRPC server.
	GRPCServeAddress string `env:"GRPC_SERVE_ADDRESS" envDefault:":3002"`
	// HTTPServeAddress is the address to run the HTTP server.
	HTTPServeAddress string `env:"HTTP_SERVE_ADDRESS" envDefault:":3001"`
	// ServerID is the server id.
	SERVER_ID string `env:"SERVER_ID" envDefault:"local"`
}
