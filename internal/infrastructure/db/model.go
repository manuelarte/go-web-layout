package db

import (
	"errors"
	"fmt"

	"github.com/manuelarte/go-web-layout/internal/users"
)

type newUser struct {
	username       users.Username
	hashedPassword string
}

// newNewUser creates a new user. Returns the user or a validation error.
func newNewUser(u users.Username, p users.Password) (newUser, error) {
	errUsername := u.IsValid()
	errPassword := p.IsValid()

	if err := errors.Join(errUsername, errPassword); err != nil {
		return newUser{}, err
	}

	hashedPassword, errHash := p.Hash()
	if errHash != nil {
		return newUser{}, fmt.Errorf("error hashing password: %w", errHash)
	}

	return newUser{username: u, hashedPassword: hashedPassword}, nil
}
