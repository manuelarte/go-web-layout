package users

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type (
	Service interface {
		GetAll(ctx context.Context, page, size int) ([]User, error)
	}

	service struct {
		// add repository
	}
)

func NewService() Service {
	return service{}
}

func (s service) GetAll(_ context.Context, _, _ int) ([]User, error) {
	user1 := User{
		ID:        uuid.MustParse("81862f49-492e-46f3-b8fb-5fee564ab1fa"),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Username:  "manuelarte",
	}

	return []User{user1}, nil
}
