package config

type AppEnv struct {
	HttpServeAddress string `env:"HTTP_SERVE_ADDRESS" envDefault:":3001"`
}
