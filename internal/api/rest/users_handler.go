package rest

import (
	"context"
	"fmt"

	"github.com/manuelarte/ptrutils"
	"github.com/samber/lo"

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

	up, err := h.service.GetAll(ctx, ptrutils.DerefOr(request.Params.Page, 0), ptrutils.DerefOr(request.Params.Size, 0))
	if err != nil {
		return nil, fmt.Errorf("error getting users: %w", err)
	}

	return GetUsers200JSONResponse{
		Data: transformUserDaoToDto(up),
		Page: Page{
			Number:        page,
			Size:          size,
			TotalElements: 0, // TODO(manuelarte): Implement this
			TotalPages:    0, // TODO(manuelarte): Implement this
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
