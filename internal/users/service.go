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

//go:generate mockgen -package $GOPACKAGE -source $GOFILE -package users -destination ./mock.gen.$GOFILE
type (
	Service interface {
		Create(context.Context, NewUser) (User, error)
		GetAll(context.Context, pagination.PageRequest) (pagination.Page[User], error)
	}

	service struct {
		repository Repository
	}
)

func NewService(r Repository) Service {
	return service{repository: r}
}

func (s service) Create(ctx context.Context, user NewUser) (User, error) {
	_, span := tracing.GetOrNewTracer(ctx).Start(
		ctx,
		"Service.Create",
	)
	defer span.End()

	createdUser, err := s.repository.Create(ctx, user)
	if err != nil {
		return User{}, fmt.Errorf("error creating user: %w", err)
	}

	return createdUser, nil
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
