package rest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/manuelarte/ptrutils"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/manuelarte/go-web-layout/internal/config"
	"github.com/manuelarte/go-web-layout/internal/pagination"
	"github.com/manuelarte/go-web-layout/internal/tracing"
	"github.com/manuelarte/go-web-layout/internal/users"
)

type UsersHandler struct {
	cfg     config.AppEnv
	service users.Service
}

func NewUsersHandler(cfg config.AppEnv, service users.Service) UsersHandler {
	return UsersHandler{
		cfg:     cfg,
		service: service,
	}
}

func (h UsersHandler) GetUser(ctx context.Context, request GetUserRequestObject) (GetUserResponseObject, error) {
	ctx, span := tracing.StartSpan(
		ctx,
		"Service.GetAll",
		oteltrace.WithAttributes(attribute.String("id", request.UserId.String())),
	)
	defer span.End()

	user, err := h.service.GetByID(ctx, request.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetUser4XXApplicationProblemPlusJSONResponse{
				StatusCode: http.StatusNotFound,
				Body: ErrorResponse{
					Type:     "NotFound",
					Title:    "User not found",
					Detail:   fmt.Sprintf("No User found with id: %s", request.UserId.String()),
					Status:   http.StatusNotFound,
					Instance: span.SpanContext().TraceID().String(),
				},
			}, nil
		}

		return nil, fmt.Errorf("error getting user: %w", err)
	}

	return GetUser200JSONResponse(transformUserDaoToDto(user)), nil
}

func (h UsersHandler) GetUsers(ctx context.Context, request GetUsersRequestObject) (GetUsersResponseObject, error) {
	//nolint:errcheck // requestID is added by a middleware
	requestID := ctx.Value(middleware.RequestIDKey).(string)
	page := ptrutils.DerefOr(request.Params.Page, 0)
	size := ptrutils.DerefOr(request.Params.Size, 20)

	pr, err := pagination.NewPageRequest(int(page), int(size))
	if err != nil {
		if errors.Is(err, pagination.ErrPageMustBeGreaterOrEqualThanZero) {
			return nil, &InvalidParamFormatError{
				ParamName: "page",
				Err:       err,
			}
		}

		if errors.Is(err, pagination.ErrSizeMustBeGreaterOrEqualThanZero) {
			return nil, &InvalidParamFormatError{
				ParamName: "size",
				Err:       err,
			}
		}

		return nil, fmt.Errorf("error creating page request: %w", err)
	}

	pageUsers, err := h.service.GetAll(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("error getting users: %w", err)
	}

	urlBuilder := func(page, size int32) string {
		return fmt.Sprintf("%s?page=%d&size=%d", Paths{}.GetUsersEndpoint.Path(), page, size)
	}
	self := urlBuilder(page, size)
	prev := urlBuilder(page-1, size)
	first := urlBuilder(0, size)
	//gosec:disable G115 -- Not expecting to overflow
	last := urlBuilder(int32(pageUsers.TotalPages()-1), size)

	next := urlBuilder(page+1, size)
	if page == 0 {
		prev = ""
	}
	//gosec:disable G115 -- Not expecting to overflow
	if page == int32(pageUsers.TotalPages()-1) {
		next = ""
	}

	return GetUsers200JSONResponse{
		Kind:    KindPage,
		Content: transformUserDaosToDtos(pageUsers.Content()),
		Page: Page{
			Number:        page,
			Size:          size,
			TotalElements: pageUsers.TotalElements(),
			//gosec:disable G115 -- Not expecting to overflow
			TotalPages: int32(pageUsers.TotalPages()),
			Self:       self,
			Prev:       prev,
			Next:       next,
			First:      first,
			Last:       last,
		},
		Metadata: RequestMetadata{
			Environment: h.cfg.Env,
			RequestId:   requestID,
			ServerId:    h.cfg.ServerID,
			ApiVersion:  "v1",
		},
	}, nil
}

func transformUserDaosToDtos(daos []users.User) []User {
	return lo.Map(daos, func(dao users.User, _ int) User {
		return transformUserDaoToDto(dao)
	})
}

func transformUserDaoToDto(dao users.User) User {
	return User{
		Self:      Paths{}.GetUserEndpoint.Path(dao.ID.String()),
		Kind:      KindUser,
		Id:        uuid.UUID(dao.ID),
		CreatedAt: dao.CreatedAt,
		UpdatedAt: dao.UpdatedAt,
		Username:  dao.Username,
	}
}
