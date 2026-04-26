package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"

	sqlc2 "github.com/manuelarte/go-web-layout/internal/infrastructure/db/sqlc"
	"github.com/manuelarte/go-web-layout/internal/logging"
	"github.com/manuelarte/go-web-layout/internal/observability"
	"github.com/manuelarte/go-web-layout/internal/pagination"
	"github.com/manuelarte/go-web-layout/internal/users"
)

var _ users.Repository = new(Repository)

type Repository struct {
	db      *sql.DB
	queries *sqlc2.Queries
}

func NewRepository(db *sql.DB) Repository {
	return Repository{
		db:      db,
		queries: sqlc2.New(db),
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

	hashedPassword, err := hashPassword(p)
	if err != nil {
		return users.User{}, fmt.Errorf("error hashing password: %w", err)
	}

	created, err := r.queries.CreateUser(ctx, sqlc2.CreateUserParams{
		ID:       uuid.New(),
		Username: string(u),
		Password: hashedPassword,
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
		sqlc2.GetUsersParams{
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

	users := lo.Map(uDao, func(item sqlc2.User, index int) users.User {
		return transformModel(item)
	})

	return pagination.MustPage(users, pr, count), nil
}

func (r Repository) GetByID(ctx context.Context, id uuid.UUID) (users.User, error) {
	ctx, span := observability.StartSpan(
		ctx,
		"Repository.GetByID",
		oteltrace.WithAttributes(attribute.String("id", id.String())),
	)
	defer span.End()

	dao, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return users.User{}, fmt.Errorf("error getting user by id: %w", err)
	}

	return transformModel(dao), nil
}

func transformModel(user sqlc2.User) users.User {
	return users.User{
		ID:        users.UserID(user.ID),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Username:  users.Username(user.Username),
	}
}

func hashPassword(password users.Password) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)

	return string(bytes), err
}
