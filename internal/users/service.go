package users

import (
	"context"

	"github.com/manuelarte/go-web-layout/internal/pagination"
)

var _ Service = new(service)

type (
	Service interface {
		GetAll(ctx context.Context, page, size int) (pagination.Page[User], error)
	}

	service struct {
		repository Repository
	}
)

func NewService(r Repository) Service {
	return service{repository: r}
}

func (s service) GetAll(ctx context.Context, page, size int) (pagination.Page[User], error) {
	// TODO(manuelarte): Validate page and size

	return s.repository.GetAll(ctx, page, size)
}
