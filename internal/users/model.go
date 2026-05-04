package users

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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
		id        UserID
		createdAt time.Time
		updatedAt time.Time
		username  Username
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

func NewUser(
	id UserID,
	createdAt time.Time,
	updatedAt time.Time,
	username Username,
) User {
	return User{
		id:        id,
		createdAt: createdAt,
		updatedAt: updatedAt,
		username:  username,
	}
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

func (p Password) Hash() (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(p), 14)
	if err != nil {
		return "", fmt.Errorf("error hashing password: %w", err)
	}

	return string(bytes), nil
}

func (u User) Username() Username {
	return u.username
}

func (u User) ID() UserID {
	return u.id
}

func (u User) CreatedAt() time.Time {
	return u.createdAt
}

func (u User) UpdatedAt() time.Time {
	return u.updatedAt
}
