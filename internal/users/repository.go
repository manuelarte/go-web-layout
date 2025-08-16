package users

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	"github.com/manuelarte/go-web-layout/internal/pagination"
	"github.com/manuelarte/go-web-layout/internal/sqlc"
)

var _ Repository = new(repository)

type (
	Repository interface {
		GetAll(context.Context, pagination.PageRequest) (pagination.Page[User], error)
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

func (r repository) GetAll(ctx context.Context, pr pagination.PageRequest) (pagination.Page[User], error) {
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
