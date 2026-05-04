package rest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/golaxo/gofieldselect"
	"github.com/google/uuid"
	"github.com/manuelarte/ptrutils"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/manuelarte/go-web-layout/internal/config"
	"github.com/manuelarte/go-web-layout/internal/logging"
	"github.com/manuelarte/go-web-layout/internal/observability"
	"github.com/manuelarte/go-web-layout/internal/pagination"
	"github.com/manuelarte/go-web-layout/internal/users"
)

type UsersHandler struct {
	cfg        config.AppEnv
	repository users.Repository
}

func NewUsersHandler(cfg config.AppEnv, repository users.Repository) UsersHandler {
	return UsersHandler{
		cfg:        cfg,
		repository: repository,
	}
}

func (h UsersHandler) GetUser(ctx context.Context, request GetUserRequestObject) (GetUserResponseObject, error) {
	ctx, span := observability.StartSpan(
		ctx,
		"UsersHandler.GetUser",
		oteltrace.WithAttributes(attribute.String("id", request.UserId.String())),
	)
	defer span.End()

	logger := logging.FromContext(ctx)

	fieldNode, err := gofieldselect.Parse(ptrutils.DerefOr(request.Params.Fields, ""))
	if err != nil {
		return nil, &InvalidParamFormatError{
			ParamName: "fields",
			Err:       err,
		}
	}

	user, err := users.UserID(request.UserId).Get(ctx, h.repository)
	if err != nil {
		if notFoundError, ok := errors.AsType[users.NotFoundError](err); ok {
			return GetUser4XXApplicationProblemPlusJSONResponse{
				StatusCode: http.StatusNotFound,
				Body: ErrorResponse{
					Type:      "NotFound",
					Title:     "User not found",
					Detail:    notFoundError.Error(),
					Status:    http.StatusNotFound,
					RequestId: middleware.GetReqID(ctx),
				},
			}, nil
		}

		logger.ErrorContext(ctx, "Error getting user", slog.Any("err", err))

		return GetUser500ApplicationProblemPlusJSONResponse(
			ErrorResponse{
				Type:      "DatabaseError",
				Title:     "Internal Server Error",
				Detail:    "Error getting user",
				Status:    http.StatusInternalServerError,
				RequestId: middleware.GetReqID(ctx),
			},
		), nil
	}

	return GetUser200JSONResponse(transformUserDaoToDto(fieldNode, user)), nil
}

func (h UsersHandler) GetUsers(ctx context.Context, request GetUsersRequestObject) (GetUsersResponseObject, error) {
	ctx, span := observability.StartSpan(
		ctx,
		"UsersHandler.GetUsers",
	)
	defer span.End()

	requestID := middleware.GetReqID(ctx)
	page := ptrutils.DerefOr(request.Params.Page, 0)
	size := ptrutils.DerefOr(request.Params.Size, 20)

	fieldNode, err := gofieldselect.Parse(ptrutils.DerefOr(request.Params.Fields, ""))
	if err != nil {
		return nil, &InvalidParamFormatError{
			ParamName: "fields",
			Err:       err,
		}
	}

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

	pageUsers, err := h.repository.GetAll(ctx, pr)
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
		Content: transformUserDaosToDtos(fieldNode, pageUsers.Content()),
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

func transformUserDaosToDtos(fieldNode gofieldselect.Node, daos []users.User) []User {
	return lo.Map(daos, func(dao users.User, _ int) User {
		return transformUserDaoToDto(fieldNode, dao)
	})
}

func transformUserDaoToDto(fieldNode gofieldselect.Node, dao users.User) User {
	return User{
		Self:      Paths{}.GetUserEndpoint.Path(dao.ID.String()),
		Kind:      KindUser,
		Id:        gofieldselect.Get(fieldNode, "id", new(uuid.UUID(dao.ID))),
		CreatedAt: gofieldselect.Get(fieldNode, "createdAt", new(dao.CreatedAt)),
		UpdatedAt: gofieldselect.Get(fieldNode, "updatedAt", new(dao.UpdatedAt)),
		Username:  gofieldselect.Get(fieldNode, "username", new(dao.Username)),
	}
}
