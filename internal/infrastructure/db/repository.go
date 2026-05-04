package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/manuelarte/go-web-layout/internal/infrastructure/db/sqlc"
	"github.com/manuelarte/go-web-layout/internal/logging"
	"github.com/manuelarte/go-web-layout/internal/observability"
	"github.com/manuelarte/go-web-layout/internal/pagination"
	"github.com/manuelarte/go-web-layout/internal/users"
)

var _ users.Repository = new(Repository)

type Repository struct {
	db      *sql.DB
	queries *sqlc.Queries
}

func NewRepository(db *sql.DB) Repository {
	return Repository{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r Repository) Create(ctx context.Context, u users.Username, p users.Password) (users.User, error) {
	ctx, span := observability.StartSpan(ctx, "Repository.Create")
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "username",
			Value: attribute.StringValue(string(u)),
		},
	)

	nu, err := newNewUser(u, p)
	if err != nil {
		return users.User{}, fmt.Errorf("error validating new user fields: %w", err)
	}

	created, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:       uuid.New(),
		Username: string(nu.username),
		Password: nu.hashedPassword,
	})
	if err != nil {
		return users.User{}, fmt.Errorf("error creating user: %w", err)
	}

	return transformModel(created), nil
}

func (r Repository) GetAll(ctx context.Context, pr pagination.PageRequest) (pagination.Page[users.User], error) {
	ctx, span := observability.StartSpan(
		ctx,
		"Repository.GetAll",
		oteltrace.WithAttributes(attribute.Int("page", pr.Page()), attribute.Int("size", pr.Size())),
	)
	defer span.End()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return pagination.Page[users.User]{}, fmt.Errorf("error starting transaction: %w", err)
	}
	defer func(tx *sql.Tx) {
		errRollback := tx.Rollback()
		if errRollback != nil {
			logging.FromContext(ctx).ErrorContext(ctx, "Failed to rollback transaction", slog.Any("err", errRollback))
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
		return pagination.Page[users.User]{}, fmt.Errorf("error getting users: %w", err)
	}

	count, err := r.queries.WithTx(tx).CountUsers(ctx)
	if err != nil {
		return pagination.Page[users.User]{}, fmt.Errorf("error counting users: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return pagination.Page[users.User]{}, fmt.Errorf("error committing transaction: %w", err)
	}

	usersMapped := lo.Map(uDao, func(item sqlc.User, index int) users.User {
		return transformModel(item)
	})

	return pagination.MustPage(usersMapped, pr, count), nil
}

func (r Repository) GetByID(ctx context.Context, id users.UserID) (users.User, error) {
	ctx, span := observability.StartSpan(
		ctx,
		"Repository.GetByID",
		oteltrace.WithAttributes(attribute.String("id", id.String())),
	)
	defer span.End()

	dao, err := r.queries.GetUserByID(ctx, uuid.UUID(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return users.User{}, users.NotFoundError{ID: id}
		}

		return users.User{}, fmt.Errorf("error getting user by id: %w", err)
	}

	return transformModel(dao), nil
}

func transformModel(user sqlc.User) users.User {
	return users.NewUser(
		users.UserID(user.ID),
		user.CreatedAt,
		user.UpdatedAt,
		users.Username(user.Username),
	)
}
