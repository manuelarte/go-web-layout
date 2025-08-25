package users

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"

	"github.com/manuelarte/go-web-layout/internal/pagination"
	"github.com/manuelarte/go-web-layout/internal/sqlc"
	"github.com/manuelarte/go-web-layout/internal/tracing"
)

var _ Repository = new(repository)

type (
	// Repository interface with the user's repository methods.
	Repository interface {
		// Create creates a new user.
		Create(context.Context, NewUser) (User, error)
		// GetAll gets all users paginated.
		GetAll(context.Context, pagination.PageRequest) (pagination.Page[User], error)
		// GetByID gets a user by its ID.
		GetByID(context.Context, uuid.UUID) (User, error)
	}

	repository struct {
		db      *sql.DB
		queries *sqlc.Queries
	}
)

func NewRepository(db *sql.DB) Repository {
	return repository{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r repository) Create(ctx context.Context, user NewUser) (User, error) {
	_, span := tracing.GetOrNewTracer(ctx).Start(
		ctx,
		"Service.Create",
	)
	defer span.End()

	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		return User{}, fmt.Errorf("error hashing password: %w", err)
	}

	created, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:       uuid.New().String(),
		Username: user.Username,
		Password: hashedPassword,
	})
	if err != nil {
		return User{}, fmt.Errorf("error creating user: %w", err)
	}

	return transformModel(created), nil
}

func (r repository) GetAll(ctx context.Context, pr pagination.PageRequest) (pagination.Page[User], error) {
	_, span := tracing.GetOrNewTracer(ctx).Start(
		ctx,
		"Repository.GetAll",
		oteltrace.WithAttributes(attribute.Int("page", pr.Page()), attribute.Int("size", pr.Size())),
	)
	defer span.End()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return pagination.Page[User]{}, fmt.Errorf("error starting transaction: %w", err)
	}
	defer func(tx *sql.Tx) {
		errRollback := tx.Rollback()
		if errRollback != nil {
			log.Info().Err(errRollback).Msg("Failed to rollback transaction")
		}
	}(tx)

	uDao, err := r.queries.WithTx(tx).GetUsers(
		ctx,
		sqlc.GetUsersParams{
			Limit:  int64(pr.Size()),
			Offset: int64(pr.Offset()),
		},
	)
	if err != nil {
		return pagination.Page[User]{}, fmt.Errorf("error getting users: %w", err)
	}

	count, err := r.queries.WithTx(tx).CountUsers(ctx)
	if err != nil {
		return pagination.Page[User]{}, fmt.Errorf("error counting users: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return pagination.Page[User]{}, fmt.Errorf("error committing transaction: %w", err)
	}

	users := lo.Map(uDao, func(item sqlc.User, index int) User {
		return transformModel(item)
	})

	return pagination.MustPage(users, pr, count), nil
}

func (r repository) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	_, span := tracing.GetOrNewTracer(ctx).Start(
		ctx,
		"Repository.GetByID",
		oteltrace.WithAttributes(attribute.String("id", id.String())),
	)
	defer span.End()

	dao, err := r.queries.GetUserByID(ctx, id.String())
	if err != nil {
		return User{}, fmt.Errorf("error getting user by id: %w", err)
	}

	return transformModel(dao), nil
}

//nolint:errcheck // TODO check how to do it better.
func transformModel(user sqlc.User) User {
	layout := "2006-01-02 15:04:05"
	createdAt, _ := time.Parse(layout, user.CreatedAt.(string))
	updatedAt, _ := time.Parse(layout, user.UpdatedAt.(string))

	return User{
		ID:        uuid.MustParse(user.ID.(string)),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Username:  user.Username,
	}
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)

	return string(bytes), err
}
