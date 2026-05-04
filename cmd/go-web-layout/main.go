package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	interceptorlogging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/riandyrn/otelchi"
	otelchimetric "github.com/riandyrn/otelchi/metric"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc"

	"github.com/manuelarte/go-web-layout/internal/config"
	"github.com/manuelarte/go-web-layout/internal/info"
	grpc2 "github.com/manuelarte/go-web-layout/internal/infrastructure/api/grpc"
	usersv1 "github.com/manuelarte/go-web-layout/internal/infrastructure/api/grpc/users/v1"
	"github.com/manuelarte/go-web-layout/internal/infrastructure/api/rest"
	"github.com/manuelarte/go-web-layout/internal/infrastructure/db"
	loggingCfg "github.com/manuelarte/go-web-layout/internal/logging"
	"github.com/manuelarte/go-web-layout/internal/logging/wideevents"
	"github.com/manuelarte/go-web-layout/internal/observability"
	"github.com/manuelarte/go-web-layout/internal/services"
)

func main() {
	err := run()
	if err != nil {
		//nolint:sloglint // only logging in default this error
		slog.Error("Failed to run server", "error", err)
	}
}

func run() error {
	ctx := context.Background()

	cfg, err := config.GetAppEnv()
	if err != nil {
		return fmt.Errorf("failed to get app env: %w", err)
	}

	tracer := otel.Tracer(info.AppName)

	otelShutdown, mp, lp, err := setupOTelSDK(ctx, cfg)
	if err != nil {
		return fmt.Errorf("error setting open telemetry: %w", err)
	}

	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// Create a bridged slog logger
	logger := otelslog.NewLogger(info.AppName, otelslog.WithLoggerProvider(lp))

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

	userRepo := db.NewRepository(dbConn)

	// define base config for metric middlewares
	baseCfg := otelchimetric.NewBaseConfig(info.AppName, otelchimetric.WithMeterProvider(mp))

	r := chi.NewRouter()

	//nolint:mnd // guess
	headerTimeout := 4 * time.Second
	r.Use(
		loggingCfg.Middleware(logger),
		addHostValue(),
		middleware.Logger,
		otelchi.Middleware(info.AppName, otelchi.WithChiRoutes(r)),
		otelchimetric.NewServerRequestDuration(baseCfg),
		otelchimetric.NewServerActiveRequests(baseCfg),
		otelchimetric.NewServerResponseBodySize(baseCfg),
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

	loggingOpts := []interceptorlogging.Option{
		interceptorlogging.WithLogOnEvents(),
	}

	createUserService := services.NewCreateUser(userRepo)

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			interceptorlogging.UnaryServerInterceptor(loggingCfg.InterceptorLogger(logger), loggingOpts...),
			loggingCfg.AddToContext(logger),
			wideevents.AddCreateUserWideEvent(),
		),
		grpc.ChainStreamInterceptor(
			interceptorlogging.StreamServerInterceptor(loggingCfg.InterceptorLogger(logger), loggingOpts...),
		),
	)
	usersv1.RegisterUsersServiceServer(s, grpc2.NewServer(createUserService))
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
) (func(context.Context) error, *sdkmetric.MeterProvider, *log.LoggerProvider, error) {
	shutdownFuncs := make([]func(context.Context) error, 3)

	var err error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown := func(ctx context.Context) error {
		var shutdownErr error

		for _, fn := range shutdownFuncs {
			if fn != nil {
				shutdownErr = errors.Join(shutdownErr, fn(ctx))
			}
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

	tp, err := observability.InitTracerProvider(ctx, cfg.OtelExporterEndpoint, cfg.Hostname)
	if err != nil {
		handleErr(err)

		return shutdown, nil, nil, fmt.Errorf("error initializing trace provider: %w", err)
	}

	shutdownFuncs[0] = tp.Shutdown
	otel.SetTracerProvider(tp)

	mp, err := observability.InitMeterProvider(ctx, cfg.OtelExporterEndpoint, cfg.Hostname)
	if err != nil {
		handleErr(err)

		return shutdown, nil, nil, fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	shutdownFuncs[1] = mp.Shutdown
	otel.SetMeterProvider(mp)

	loggerProvider, err := observability.InitLoggingProvider(ctx, cfg.OtelExporterEndpoint, cfg.Hostname)
	if err != nil {
		handleErr(err)

		return shutdown, nil, nil, fmt.Errorf("failed to initialize logger provider: %w", err)
	}

	shutdownFuncs[2] = loggerProvider.Shutdown
	global.SetLoggerProvider(loggerProvider)

	return shutdown, mp, loggerProvider, nil
}

// addHostValue set the host as a parameter.
func addHostValue() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//nolint:staticcheck // looking for a better solution
			ctx := context.WithValue(r.Context(), "host", fmt.Sprintf("http://%s", r.Host))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
