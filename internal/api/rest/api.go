package rest

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"

	resources "github.com/manuelarte/go-web-layout"
	"github.com/manuelarte/go-web-layout/internal/config"
	"github.com/manuelarte/go-web-layout/internal/tracing"
	"github.com/manuelarte/go-web-layout/internal/users"
)

var _ StrictServerInterface = new(API)

type API struct {
	ActuatorsHandler
	UsersHandler
}

func CreateRestAPI(r chi.Router, cfg config.AppEnv, userService users.Service) {
	api := API{
		UsersHandler: NewUsersHandler(cfg, userService),
	}
	ssi := NewStrictHandlerWithOptions(api, nil, StrictHTTPServerOptions{
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			_, span := tracing.StartSpan(r.Context(), "ResponseErrorHandlerFunc")
			defer span.End()

			var validationErr ValidationError
			if errors.As(err, &validationErr) {
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("Content-Type", "application/problem+json")

				resp := func() ValidationError {
					var target ValidationError

					_ = errors.As(err, &target)

					return target
				}().ErrorResponse(span.SpanContext().TraceID().String())

				bytes, errMarshal := json.Marshal(resp)
				if errMarshal != nil {
					log.Error().Err(errMarshal).Msg("Failed to marshal error response")

					return
				}

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(bytes)

				return
			}

			var invalidParamError *InvalidParamFormatError
			if errors.As(err, &invalidParamError) {
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("Content-Type", "application/problem+json")

				resp := invalidParamError.ErrorResponse(span.SpanContext().TraceID().String())

				bytes, errMarshal := json.Marshal(resp)
				if errMarshal != nil {
					log.Error().Err(errMarshal).Msg("Failed to marshal error response")

					return
				}

				_, _ = w.Write(bytes)

				return
			}
		},
	})
	HandlerWithOptions(ssi, ChiServerOptions{
		BaseRouter:  r,
		Middlewares: nil,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			_, span := tracing.StartSpan(r.Context(), "ErrorHandlerFunc")
			defer span.End()

			w.Header().Set("Content-Type", "application/problem+json")

			var invalidParamError *InvalidParamFormatError
			if errors.As(err, &invalidParamError) {
				w.WriteHeader(http.StatusBadRequest)

				resp := invalidParamError.ErrorResponse(span.SpanContext().TraceID().String())

				bytes, errMarshal := json.Marshal(resp)
				if errMarshal != nil {
					log.Error().Err(errMarshal).Msg("Failed to marshal error response")

					return
				}

				_, _ = w.Write(bytes)
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
