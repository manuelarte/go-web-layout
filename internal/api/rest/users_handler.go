package rest

import (
	"context"
	"errors"
	"fmt"

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

func (h UsersHandler) GetUsers(ctx context.Context, request GetUsersRequestObject) (GetUsersResponseObject, error) {
	page := ptrutils.DerefOr(request.Params.Page, 0)
	size := ptrutils.DerefOr(request.Params.Size, 20)

	pr, err := pagination.NewPageRequest(page, size)
	if err != nil {
		if errors.Is(err, pagination.ErrPageMustBeGreateOrEqualThanZero) {
			return nil, ValidationError{map[string][]error{"page": {err}}}
		}

		if errors.Is(err, pagination.ErrSizeMustBeGreateOrEqualThanZero) {
			return nil, ValidationError{map[string][]error{"size": {err}}}
		}

		return nil, fmt.Errorf("error creating page request: %w", err)
	}

	pageUsers, err := h.service.GetAll(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("error getting users: %w", err)
	}

	return GetUsers200JSONResponse{
		Data: transformUserDaoToDto(pageUsers.Data()),
		Page: Page{
			Number:        page,
			Size:          size,
			TotalElements: pageUsers.TotalElements(),
			TotalPages:    pageUsers.TotalPages(),
		},
	}, nil
}

func transformUserDaoToDto(daos []users.User) []User {
	return lo.Map(daos, func(dao users.User, _ int) User {
		return User{
			Id:        dao.ID,
			CreatedAt: dao.CreatedAt,
			UpdatedAt: dao.UpdatedAt,
			Username:  dao.Username,
		}
	})
}
