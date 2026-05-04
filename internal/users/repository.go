package users

import (
	"context"

	"github.com/manuelarte/go-web-layout/internal/pagination"
)

//go:generate mockgen -typed -package $GOPACKAGE -source $GOFILE -package users -destination ./mock.gen.$GOFILE
type (
	// Repository interface with the user's repository methods.
	Repository interface {
		// Create creates a new user.
		Create(context.Context, Username, Password) (User, error)
		// GetAll gets all users paginated.
		GetAll(context.Context, pagination.PageRequest) (pagination.Page[User], error)
		// GetByID gets a user by its ID.
		// Can return either UserNotFoundError if the user id is not found,
		// or any other database error.
		GetByID(context.Context, UserID) (User, error)
	}
)
