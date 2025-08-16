package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/riandyrn/otelchi"
	otelchimetric "github.com/riandyrn/otelchi/metric"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/manuelarte/go-web-layout/internal/api/rest"
	"github.com/manuelarte/go-web-layout/internal/config"
	"github.com/manuelarte/go-web-layout/internal/tracing"
	"github.com/manuelarte/go-web-layout/internal/users"
)

//go:generate go tool oapi-codegen -config openapi-cfg.yaml ../../openapi.yml
//go:generate sqlc generate -f ../../sqlc.yml
func main() {
	err := run()
	if err != nil {
		log.Panic().Err(err).Msg("Failed to run server")
	}
}

func run() error {
	ctx := context.Background()

	db, err := config.Migrate()
	if err != nil {
		return fmt.Errorf("failed to migrate the database: %w", err)
	}
	defer func(db *sql.DB) {
		errClose := db.Close()
		if errClose != nil {
			log.Error().Err(errClose).Msg("Failed to close database")
		}
	}(db)

	cfg, err := env.ParseAs[config.AppEnv]()
	if err != nil {
		return err
	}

	userRepo := users.NewRepository(db)
	userService := users.NewService(userRepo)

	// telemetry
	tp, err := tracing.InitTracerProvider()
	if err != nil {
		return fmt.Errorf("failed to initialize tracer provider: %w", err)
	}

	defer func() {
		errShutdown := tp.Shutdown(context.Background())
		if errShutdown != nil {
			log.Printf("Error shutting down tracer provider: %v", errShutdown)
		}
	}()
	// set global tracer provider & text propagators
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	// initialize tracer
	tracer := otel.Tracer("go-web-layout")
	// initialize meter provider & set global meter provider
	mp, err := tracing.InitMeter()
	if err != nil {
		return fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	otel.SetMeterProvider(mp)
	// define base config for metric middlewares
	baseCfg := otelchimetric.NewBaseConfig("go-web-layout", otelchimetric.WithMeterProvider(mp))

	r := chi.NewRouter()

	//nolint:mnd // guess
	headerTimeout := 4 * time.Second
	r.Use(
		middleware.Logger,
		otelchi.Middleware("go-web-layout", otelchi.WithChiRoutes(r)),
		otelchimetric.NewRequestDurationMillis(baseCfg),
		otelchimetric.NewRequestInFlight(baseCfg),
		otelchimetric.NewResponseSizeBytes(baseCfg),
		middleware.Recoverer,
		middleware.RequestID,
		middleware.RealIP,
		middleware.Timeout(headerTimeout),
	)
	createRestAPI(r, userService)

	srv := &http.Server{
		Addr:              cfg.HTTPServeAddress,
		Handler:           r,
		ReadHeaderTimeout: headerTimeout, // Prevent G112 (CWE-400)
		BaseContext: func(net.Listener) context.Context {
			return context.WithValue(ctx, tracing.Context{}, tracer)
		},
	}

	log.Printf("Starting server on port %s", srv.Addr)

	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start server: %w", err)
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
