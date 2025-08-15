package users

import (
	"context"
	"database/sql"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/manuelarte/go-web-layout/internal/pagination"
	"github.com/manuelarte/go-web-layout/internal/sqlc"
)

var _ Repository = new(repository)

type (
	Repository interface {
		GetAll(ctx context.Context, page, size int) (pagination.Page[User], error)
	}

	repository struct {
		queries *sqlc.Queries
	}
)

func NewRepository(db *sql.DB) Repository {
	return repository{
		queries: sqlc.New(db),
	}
}

func (r repository) GetAll(ctx context.Context, page, size int) (pagination.Page[User], error) {
	uDao, err := r.queries.GetUsers(ctx, sqlc.GetUsersParams{
		Limit:  int64(size),
		Offset: int64(page * size),
	})
	if err != nil {
		return pagination.Page[User]{}, err
	}

	count, err := r.queries.CountUsers(ctx)
	if err != nil {
		return pagination.Page[User]{}, err
	}

	users := lo.Map(uDao, func(item sqlc.User, index int) User {
		return transformModel(item)
	})

	return pagination.Page[User]{
		Data:          users,
		Size:          size,
		TotalElements: count,
		TotalPages:    int(math.Ceil(float64(count) / float64(size))),
		Number:        page,
	}, nil
}

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
