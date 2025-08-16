package users

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/manuelarte/go-web-layout/internal/pagination"
	"github.com/manuelarte/go-web-layout/internal/tracing"
)

var _ Service = new(service)

type (
	Service interface {
		GetAll(context.Context, pagination.PageRequest) (pagination.Page[User], error)
	}

	service struct {
		repository Repository
	}
)

func NewService(r Repository) Service {
	return service{repository: r}
}

func (s service) GetAll(ctx context.Context, pr pagination.PageRequest) (pagination.Page[User], error) {
	_, span := tracing.GetOrNewTracer(ctx).Start(
		ctx,
		"Service.GetAll",
		oteltrace.WithAttributes(attribute.Int("page", pr.Page()), attribute.Int("size", pr.Size())),
	)
	defer span.End()

	pageUsers, err := s.repository.GetAll(ctx, pr)
	if err != nil {
		return pagination.Page[User]{}, fmt.Errorf("error getting users: %w", err)
	}

	return pageUsers, nil
}
