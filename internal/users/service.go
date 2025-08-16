package users

import (
	"context"
	"fmt"

	"github.com/manuelarte/go-web-layout/internal/pagination"
)

var _ Service = new(service)

type (
	Service interface {
		GetAll(ctx context.Context, pr pagination.PageRequest) (pagination.Page[User], error)
	}

	service struct {
		repository Repository
	}
)

func NewService(r Repository) Service {
	return service{repository: r}
}

func (s service) GetAll(ctx context.Context, pr pagination.PageRequest) (pagination.Page[User], error) {
	// TODO(manuelarte): opentelemetry
	pageUsers, err := s.repository.GetAll(ctx, pr)
	if err != nil {
		return pagination.Page[User]{}, fmt.Errorf("error getting users: %w", err)
	}

	return pageUsers, nil
}
