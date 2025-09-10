package users

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUsernameTooShort = errors.New("username too short")
	ErrUsernameTooLong  = errors.New("username too long")
	ErrPasswordTooShort = errors.New("password too short")
	ErrPasswordTooLong  = errors.New("password too long")
)

type (
	// User model to represent a user.
	User struct {
		ID        uuid.UUID
		CreatedAt time.Time
		UpdatedAt time.Time
		Username  Username
	}

	// NewUser model to represent a new user.
	NewUser struct {
		Username Username
		Password Password
	}

	Username string

	Password string
)

func (u *NewUser) IsValid() error {
	return errors.Join(u.Username.IsValid(), u.Password.IsValid())
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
