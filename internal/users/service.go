package users

import (
	"context"
	"fmt"

	appcontext "github.com/manuelarte/go-web-layout/internal/context"
	"github.com/manuelarte/go-web-layout/internal/pagination"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
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
	_, span := ctx.Value(appcontext.Tracer{}).(oteltrace.Tracer).Start(ctx, "GetAll", oteltrace.WithAttributes(attribute.Int("page", pr.Page()), attribute.Int("size", pr.Size())))
	defer span.End()

	pageUsers, err := s.repository.GetAll(ctx, pr)
	if err != nil {
		return pagination.Page[User]{}, fmt.Errorf("error getting users: %w", err)
	}

	return pageUsers, nil
}
