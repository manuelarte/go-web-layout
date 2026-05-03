package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUsernameTooShort       = errors.New("username too short")
	ErrUsernameTooLong        = errors.New("username too long")
	ErrPasswordTooShort       = errors.New("password too short")
	ErrPasswordTooLong        = errors.New("password too long")
	_                   error = new(NotFoundError)
)

type (
	// User model to represent a user.
	User struct {
		ID        UserID
		CreatedAt time.Time
		UpdatedAt time.Time
		Username  Username
	}

	UserID uuid.UUID

	Username string

	Password string

	NotFoundError struct {
		ID UserID
	}
)

func (u NotFoundError) Error() string {
	return fmt.Sprintf("user with id %s not found", u.ID.String())
}

// Get gets a user by its ID.
// Can return either UserNotFoundError if the user id is not found,
// or any other database error.
//
//nolint:wrapcheck // already wrapped in the repository
func (id UserID) Get(ctx context.Context, r Repository) (User, error) {
	return r.GetByID(ctx, id)
}

// NewUser creates a new user.
func NewUser(ctx context.Context, u Username, p Password, r Repository) (User, error) {
	if err := u.IsValid(); err != nil {
		return User{}, err
	}

	if err := p.IsValid(); err != nil {
		return User{}, err
	}

	user, err := r.Create(ctx, u, p)
	if err != nil {
		return User{}, fmt.Errorf("error creating user: %w", err)
	}

	return user, nil
}

func (id UserID) String() string {
	return uuid.UUID(id).String()
}

func (u Username) IsValid() error {
	if len(u) < 3 {
		return ErrUsernameTooShort
	}

	if len(u) > 32 {
		return ErrUsernameTooLong
	}

	return nil
}

func (p Password) IsValid() error {
	if len(p) < 8 {
		return ErrPasswordTooShort
	}

	if len(p) > 64 {
		return ErrPasswordTooLong
	}

	return nil
}
