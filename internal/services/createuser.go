package services

import (
	"context"
	"fmt"

	"github.com/manuelarte/go-web-layout/internal/users"
)

type CreateUser struct {
	repository users.Repository
}

func NewCreateUser(repository users.Repository) CreateUser {
	return CreateUser{
		repository: repository,
	}
}

// CreateUser creates a new user. It either returns the created user or one of the following errors:
// - Validation error, username and/or password are wrong.
// - Database error, can't save the user.
func (s CreateUser) CreateUser(
	ctx context.Context,
	u users.Username,
	p users.Password,
) (users.User, error) {
	user, err := s.repository.Create(ctx, u, p)
	if err != nil {
		return users.User{}, fmt.Errorf("error creating user: %w", err)
	}

	return user, nil
}
