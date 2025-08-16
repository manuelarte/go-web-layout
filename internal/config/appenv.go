package config

type AppEnv struct {
	HTTPServeAddress string `env:"HTTP_SERVE_ADDRESS" envDefault:":3001"`
}
