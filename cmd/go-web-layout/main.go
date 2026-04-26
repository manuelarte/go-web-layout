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
	"github.com/manuelarte/go-web-layout/internal/observability"
	"github.com/riandyrn/otelchi"
	otelchimetric "github.com/riandyrn/otelchi/metric"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc"

	"github.com/manuelarte/go-web-layout/internal/config"
	"github.com/manuelarte/go-web-layout/internal/info"
	usersv2 "github.com/manuelarte/go-web-layout/internal/infrastructure/api/grpc/users/v1"
	"github.com/manuelarte/go-web-layout/internal/infrastructure/api/rest"
	"github.com/manuelarte/go-web-layout/internal/infrastructure/db"
	"github.com/manuelarte/go-web-layout/internal/logging"
)

func main() {
	logger := slog.Default()

	err := run(logger)
	if err != nil {
		logger.Error("Failed to run server", "error", err)
	}
}

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

	otelShutdown, mp, err := setupOTelSDK(ctx, cfg, hostname)
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
			return observability.AddContext(ctx, tracer)
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

	opts := []interceptorlogging.Option{
		interceptorlogging.WithLogOnEvents(interceptorlogging.StartCall, interceptorlogging.FinishCall),
	}

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			interceptorlogging.UnaryServerInterceptor(logging.InterceptorLogger(logger), opts...),
			logging.UnaryServerInterceptor(logger),
		),
		grpc.ChainStreamInterceptor(
			interceptorlogging.StreamServerInterceptor(logging.InterceptorLogger(logger), opts...),
		),
	)
	usersv2.RegisterUsersServiceServer(s, usersv2.NewServer(userRepo))
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
) (func(context.Context) error, *sdkmetric.MeterProvider, error) {
	shutdownFuncs := make([]func(context.Context) error, 2)

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

	tp, err := observability.InitTracerProvider(ctx, cfg.OtelExporterEndpoint, hostname)
	if err != nil {
		handleErr(err)

		return shutdown, nil, fmt.Errorf("error initializing trace provider: %w", err)
	}

	shutdownFuncs[0] = tp.Shutdown
	otel.SetTracerProvider(tp)

	mp, err := observability.InitMeterProvider()
	if err != nil {
		handleErr(err)

		return shutdown, nil, fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	shutdownFuncs[1] = mp.Shutdown
	otel.SetMeterProvider(mp)

	return shutdown, mp, nil
}
