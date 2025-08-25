package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/riandyrn/otelchi"
	otelchimetric "github.com/riandyrn/otelchi/metric"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	oteltracing "google.golang.org/grpc/experimental/opentelemetry"
	"google.golang.org/grpc/stats/opentelemetry"

	resources "github.com/manuelarte/go-web-layout"
	usersv1 "github.com/manuelarte/go-web-layout/internal/api/grpc/users/v1"
	"github.com/manuelarte/go-web-layout/internal/api/rest"
	"github.com/manuelarte/go-web-layout/internal/config"
	"github.com/manuelarte/go-web-layout/internal/tracing"
	"github.com/manuelarte/go-web-layout/internal/users"
)

//go:generate go tool oapi-codegen -config openapi-cfg.yaml ../../resources/openapi.yml
//go:generate sqlc generate -f ../../sqlc.yml
func main() {
	err := run()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to run server")
	}
}

//nolint:funlen // refactor later
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

	textMapPropagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(textMapPropagator)
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

	srvErr := make(chan error, 1)

	srv := &http.Server{
		Addr:              cfg.HTTPServeAddress,
		Handler:           r,
		ReadHeaderTimeout: headerTimeout, // Prevent G112 (CWE-400)
		BaseContext: func(net.Listener) context.Context {
			return context.WithValue(ctx, tracing.Context{}, tracer)
		},
	}

	log.Printf("Starting Web server on port %s", srv.Addr)

	go func() {
		srvErr <- srv.ListenAndServe()
	}()

	listenConfig := net.ListenConfig{}

	lis, err := listenConfig.Listen(ctx, "tcp", cfg.GRPCServeAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	so := opentelemetry.ServerOption(opentelemetry.Options{
		MetricsOptions: opentelemetry.MetricsOptions{
			MeterProvider: mp,
			// These are example experimental gRPC metrics, which are disabled
			// by default and must be explicitly enabled. For the full,
			// up-to-date list of metrics, see:
			// https://grpc.io/docs/guides/opentelemetry-metrics/#instruments
			Metrics: opentelemetry.DefaultMetrics().Add(
				"grpc.lb.pick_first.connection_attempts_succeeded",
				"grpc.lb.pick_first.connection_attempts_failed",
			),
		},
		TraceOptions: oteltracing.TraceOptions{TracerProvider: tp, TextMapPropagator: textMapPropagator},
	},
	)

	s := grpc.NewServer(so)
	usersv1.RegisterUsersServiceServer(s, usersv1.NewServer(userService))
	log.Printf("Starting gRPC server on port %s", lis.Addr())

	go func() {
		srvErr <- s.Serve(lis)
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		// Wait for first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
	}

	s.GracefulStop()

	errHTTP := srv.Shutdown(context.Background())
	if errHTTP != nil {
		return fmt.Errorf("error shutting down http server: %w", errHTTP)
	}

	return nil
}

func createRestAPI(r chi.Router, userService users.Service) {
	api := rest.API{
		UsersHandler: rest.NewUsersHandler(userService),
	}
	ssi := rest.NewStrictHandlerWithOptions(api, nil, rest.StrictHTTPServerOptions{
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			var validationErr rest.ValidationError
			if errors.As(err, &validationErr) {
				w.WriteHeader(http.StatusBadRequest)
				resp := func() rest.ValidationError {
					var target rest.ValidationError
					_ = errors.As(err, &target)

					return target
				}().ErrorResponse()
				bytes, errMarshal := json.Marshal(resp)
				if errMarshal != nil {
					log.Error().Err(errMarshal).Msg("Failed to marshal error response")

					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(bytes)

				return
			}
		},
	})
	rest.HandlerFromMux(ssi, r)

	// Prometheus
	r.Handle("/metrics", promhttp.Handler())

	// Swagger
	sfs, _ := fs.Sub(fs.FS(resources.SwaggerUI), "static/swagger-ui")
	r.Handle("/swagger/*", http.StripPrefix("/swagger/", http.FileServer(http.FS(sfs))))

	r.Get("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(resources.OpenAPI)
	})
}
