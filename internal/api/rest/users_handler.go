package rest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/manuelarte/ptrutils"
	"github.com/samber/lo"

	"github.com/manuelarte/go-web-layout/internal/pagination"
	"github.com/manuelarte/go-web-layout/internal/users"
)

type UsersHandler struct {
	service users.Service
}

func NewUsersHandler(service users.Service) UsersHandler {
	return UsersHandler{
		service: service,
	}
}

func (h UsersHandler) GetUser(ctx context.Context, request GetUserRequestObject) (GetUserResponseObject, error) {
	user, err := h.service.GetByID(ctx, request.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetUser4XXJSONResponse{
				StatusCode: http.StatusNotFound,
				Body: ErrorResponse{
					Code:    strconv.Itoa(http.StatusNotFound),
					Message: fmt.Sprintf("No user found with id: %s", request.UserId.String()),
				},
			}, nil
		}

		return nil, fmt.Errorf("error getting user: %w", err)
	}

	return GetUser200JSONResponse(transformUserDaoToDto(user)), nil
}

func (h UsersHandler) GetUsers(ctx context.Context, request GetUsersRequestObject) (GetUsersResponseObject, error) {
	page := ptrutils.DerefOr(request.Params.Page, 0)
	size := ptrutils.DerefOr(request.Params.Size, 20)

	pr, err := pagination.NewPageRequest(int(page), int(size))
	if err != nil {
		if errors.Is(err, pagination.ErrPageMustBeGreaterOrEqualThanZero) {
			return nil, ValidationError{map[string][]error{"page": {err}}}
		}

		if errors.Is(err, pagination.ErrSizeMustBeGreaterOrEqualThanZero) {
			return nil, ValidationError{map[string][]error{"size": {err}}}
		}

		return nil, fmt.Errorf("error creating page request: %w", err)
	}

	pageUsers, err := h.service.GetAll(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("error getting users: %w", err)
	}

	return GetUsers200JSONResponse{
		Self:    fmt.Sprintf("/api/v1/users?page=%d&size=%d", page, size),
		Kind:    KindCollection,
		Content: transformUserDaosToDtos(pageUsers.Content()),
		Page: Page{
			Number:        page,
			Size:          size,
			TotalElements: pageUsers.TotalElements(),
			//gosec:disable G115 -- Not expecting to overflow
			TotalPages: int32(pageUsers.TotalPages()),
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
		Self:      fmt.Sprintf("/api/v1/users/%s", dao.ID),
		Kind:      KindUser,
		Id:        dao.ID,
		CreatedAt: dao.CreatedAt,
		UpdatedAt: dao.UpdatedAt,
		Username:  dao.Username,
	}
}
