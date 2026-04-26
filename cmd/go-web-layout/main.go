package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	interceptorlogging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/riandyrn/otelchi"
	otelchimetric "github.com/riandyrn/otelchi/metric"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	oteltracing "google.golang.org/grpc/experimental/opentelemetry"
	"google.golang.org/grpc/stats/opentelemetry"

	usersv1 "github.com/manuelarte/go-web-layout/internal/api/grpc/users/v1"
	"github.com/manuelarte/go-web-layout/internal/api/rest"
	"github.com/manuelarte/go-web-layout/internal/config"
	"github.com/manuelarte/go-web-layout/internal/info"
	"github.com/manuelarte/go-web-layout/internal/infrastructure/db"
	"github.com/manuelarte/go-web-layout/internal/logging"
	"github.com/manuelarte/go-web-layout/internal/tracing"
)

func main() {
	logger := slog.Default()

	err := run(logger)
	if err != nil {
		logger.Error("Failed to run server", "error", err)
	}
}

//nolint:funlen // refactor later
func run(logger *slog.Logger) error {
	ctx := context.Background()

	dbConn, err := config.Migrate()
	if err != nil {
		return fmt.Errorf("failed to migrate the database: %w", err)
	}
	defer func(dbConn *sql.DB) {
		errClose := dbConn.Close()
		if errClose != nil {
			logger.ErrorContext(ctx, "Failed to close database", slog.Any("error", errClose))
		}
	}(dbConn)

	cfg, err := env.ParseAs[config.AppEnv]()
	if err != nil {
		return err
	}

	userRepo := db.NewRepository(dbConn)

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	// initialize tracer
	tracer := otel.Tracer(info.AppName)
	// initialize meter provider & set global meter provider
	mp, err := tracing.InitMeter()
	if err != nil {
		return fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	otelShutdown, prop, tp, err := setupOTelSDK(ctx, cfg, hostname)
	if err != nil {
		return fmt.Errorf("error setting open telemetry: %w", err)
	}

	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// define base config for metric middlewares
	baseCfg := otelchimetric.NewBaseConfig(info.AppName, otelchimetric.WithMeterProvider(mp))

	r := chi.NewRouter()

	//nolint:mnd // guess
	headerTimeout := 4 * time.Second
	r.Use(
		logging.Middleware(logger),
		middleware.Logger,
		otelchi.Middleware(info.AppName, otelchi.WithChiRoutes(r)),
		otelchimetric.NewRequestDurationMillis(baseCfg),
		otelchimetric.NewRequestInFlight(baseCfg),
		otelchimetric.NewResponseSizeBytes(baseCfg),
		middleware.Recoverer,
		middleware.RequestID,
		middleware.RealIP,
		middleware.Timeout(headerTimeout),
	)
	rest.CreateRestAPI(r, cfg, userRepo)

	srvErr := make(chan error, 1)

	srv := &http.Server{
		Addr:              cfg.HTTPServeAddress,
		Handler:           r,
		ReadHeaderTimeout: headerTimeout, // Prevent G112 (CWE-400)
		BaseContext: func(net.Listener) context.Context {
			return tracing.AddContext(ctx, tracer)
		},
	}

	logger.InfoContext(ctx, "Starting Web server", slog.String("addr", srv.Addr))

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
		TraceOptions: oteltracing.TraceOptions{TracerProvider: tp, TextMapPropagator: prop},
	},
	)

	opts := []interceptorlogging.Option{
		interceptorlogging.WithLogOnEvents(interceptorlogging.StartCall, interceptorlogging.FinishCall),
	}

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptorlogging.UnaryServerInterceptor(logging.InterceptorLogger(logger), opts...),
			logging.UnaryServerInterceptor(logger),
		),
		grpc.ChainStreamInterceptor(
			interceptorlogging.StreamServerInterceptor(logging.InterceptorLogger(logger), opts...),
		),
		so,
	)
	usersv1.RegisterUsersServiceServer(s, usersv1.NewServer(userRepo))
	logger.InfoContext(ctx, "Starting gRPC server", slog.Any("addr", lis.Addr()))

	go func() {
		srvErr <- s.Serve(lis)
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		// Wait for the first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
	}

	s.GracefulStop()

	errHTTP := srv.Shutdown(context.Background())
	if errHTTP != nil {
		return fmt.Errorf("error shutting down http server: %w", errHTTP)
	}

	return nil
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func setupOTelSDK(
	ctx context.Context,
	cfg config.AppEnv,
	hostname string,
) (func(context.Context) error, propagation.TextMapPropagator, *sdktrace.TracerProvider, error) {
	shutdownFuncs := make([]func(context.Context) error, 1)

	var err error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown := func(ctx context.Context) error {
		var shutdownErr error
		for _, fn := range shutdownFuncs {
			shutdownErr = errors.Join(shutdownErr, fn(ctx))
		}

		shutdownFuncs = nil

		return shutdownErr
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	tp, err := tracing.InitTracerProvider(ctx, cfg.OtelExporterEndpoint, hostname)
	if err != nil {
		handleErr(err)

		return shutdown, nil, nil, fmt.Errorf("error initializing trace provider: %w", err)
	}

	shutdownFuncs[0] = tp.Shutdown
	otel.SetTracerProvider(tp)

	return shutdown, prop, tp, nil
}
