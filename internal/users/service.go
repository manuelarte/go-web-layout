package users

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/manuelarte/go-web-layout/internal/pagination"
	"github.com/manuelarte/go-web-layout/internal/tracing"
)

var _ Service = new(service)

//go:generate mockgen -package $GOPACKAGE -source $GOFILE -package users -destination ./mock.gen.$GOFILE
type (
	// Service interface with the user's service methods.
	Service interface {
		// Create creates a new user.
		Create(context.Context, NewUser) (User, error)
		// GetAll gets all users paginated.
		GetAll(context.Context, pagination.PageRequest) (pagination.Page[User], error)
		// GetByID gets a user by its ID.
		GetByID(context.Context, uuid.UUID) (User, error)
	}

	service struct {
		repository Repository
	}
)

func NewService(r Repository) Service {
	return service{repository: r}
}

func (s service) Create(ctx context.Context, user NewUser) (User, error) {
	ctx, span := tracing.StartSpan(
		ctx,
		"Service.Create",
	)
	defer span.End()

	if err := user.IsValid(); err != nil {
		return User{}, fmt.Errorf("invalid user: %w", err)
	}

	createdUser, err := s.repository.Create(ctx, user)
	if err != nil {
		return User{}, fmt.Errorf("error creating user: %w", err)
	}

	return createdUser, nil
}

func (s service) GetAll(ctx context.Context, pr pagination.PageRequest) (pagination.Page[User], error) {
	ctx, span := tracing.StartSpan(
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

func (s service) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	ctx, span := tracing.StartSpan(
		ctx,
		"Service.GetByID",
		oteltrace.WithAttributes(attribute.String("id", id.String())),
	)
	defer span.End()

	user, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("error getting user by id: %w", err)
	}

	return user, nil
}
