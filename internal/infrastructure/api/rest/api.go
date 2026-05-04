package rest

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	resources "github.com/manuelarte/go-web-layout"
	"github.com/manuelarte/go-web-layout/internal/config"
	"github.com/manuelarte/go-web-layout/internal/logging"
	"github.com/manuelarte/go-web-layout/internal/observability"
	"github.com/manuelarte/go-web-layout/internal/users"
)

var _ StrictServerInterface = new(API)

type API struct {
	ActuatorsHandler
	UsersHandler
}

func CreateRestAPI(r chi.Router, cfg config.AppEnv, userRepository users.Repository) {
	api := API{
		UsersHandler: NewUsersHandler(cfg, userRepository),
	}
	ssi := NewStrictHandlerWithOptions(api, nil, StrictHTTPServerOptions{
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			_, span := observability.StartSpan(r.Context(), "ResponseErrorHandlerFunc")
			defer span.End()

			if _, ok := errors.AsType[ValidationError](err); ok {
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusBadRequest)

				resp := func() ValidationError {
					var target ValidationError

					_ = errors.As(err, &target)

					return target
				}().ErrorResponse(middleware.GetReqID(r.Context()))

				bytes, errMarshal := json.Marshal(resp)
				if errMarshal != nil {
					logging.FromContext(r.Context()).Error("Failed to marshal error response", slog.Any("err", errMarshal))

					return
				}

				_, _ = w.Write(bytes) // #nosec G705

				return
			}

			if invalidParamError, ok := errors.AsType[*InvalidParamFormatError](err); ok {
				resp := invalidParamError.ErrorResponse(middleware.GetReqID(r.Context()))

				bytes, errMarshal := json.Marshal(resp)
				if errMarshal != nil {
					logging.FromContext(r.Context()).Error("Failed to marshal error response", slog.Any("err", errMarshal))

					return
				}

				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write(bytes) // #nosec G705

				return
			}
		},
	})
	HandlerWithOptions(ssi, ChiServerOptions{
		BaseRouter:  r,
		Middlewares: nil,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			_, span := observability.StartSpan(r.Context(), "ErrorHandlerFunc")
			defer span.End()

			if invalidParamError, ok := errors.AsType[*InvalidParamFormatError](err); ok {
				resp := invalidParamError.ErrorResponse(middleware.GetReqID(r.Context()))

				bytes, errMarshal := json.Marshal(resp)
				if errMarshal != nil {
					logging.FromContext(r.Context()).Error("Failed to marshal error response", slog.Any("err", errMarshal))

					return
				}

				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write(bytes) // #nosec G705
			}
		},
	})

	// Prometheus
	r.Handle("/metrics", promhttp.Handler())

	// Swagger
	sfs, _ := fs.Sub(fs.FS(resources.SwaggerUI), "static/swagger-ui")
	r.Handle("/swagger/*", http.StripPrefix("/swagger/", http.FileServer(http.FS(sfs))))

	r.Get("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(resources.OpenAPI)
	})
}
