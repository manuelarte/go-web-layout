package users

import (
	"time"

	"github.com/google/uuid"
)

// User model to represent a user.
type User struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	Username  string
}

type NewUser struct {
	Username string
	Password string
}
