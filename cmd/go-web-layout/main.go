package main

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"

	"github.com/manuelarte/go-web-layout/internal/api/rest"
	"github.com/manuelarte/go-web-layout/internal/config"
	"github.com/manuelarte/go-web-layout/internal/users"
)

//go:generate go tool oapi-codegen -config openapi-cfg.yaml ../../openapi.yml
func main() {
	err := run()
	if err != nil {
		log.Panic().Err(err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := env.ParseAs[config.AppEnv]()
	if err != nil {
		return err
	}

	userService := users.NewService()

	r := chi.NewRouter()

	//nolint:mnd // guess
	headerTimeout := 4 * time.Second
	r.Use(
		middleware.Logger,
		middleware.Recoverer,
		middleware.RequestID,
		middleware.RealIP,
		middleware.Timeout(headerTimeout),
	)
	createRestAPI(r, userService)

	srv := &http.Server{
		Addr:              cfg.HttpServeAddress,
		Handler:           r,
		ReadHeaderTimeout: headerTimeout, // Prevent G112 (CWE-400)
	}

	log.Printf("Starting server on port %s", srv.Addr)
	if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func createRestAPI(r chi.Router, userService users.Service) {
	api := rest.API{
		UsersHandler: rest.NewUsersHandler(userService),
	}
	ssi := rest.NewStrictHandler(api, nil)
	rest.HandlerFromMux(ssi, r)
}
