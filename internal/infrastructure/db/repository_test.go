package db

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/manuelarte/go-web-layout/internal/config"
	"github.com/manuelarte/go-web-layout/internal/pagination"
	"github.com/manuelarte/go-web-layout/internal/users"
)

func TestRepositoryGetAllSuccessful(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		migrate     []users.User
		pageRequest pagination.PageRequest
		expected    func(migrated []users.User, pr pagination.PageRequest) (expected pagination.Page[users.User])
	}{
		"empty": {
			pageRequest: pagination.MustPageRequest(0, 20),
			expected: func(migrated []users.User, pr pagination.PageRequest) pagination.Page[users.User] {
				return pagination.MustPage[users.User](nil, pr, 1)
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			db, err := config.Migrate()
			require.NoError(t, err)

			r := NewRepository(db)
			pr := pagination.MustPageRequest(0, 10)

			// Act
			actual, err := r.GetAll(t.Context(), pr)

			// Assert
			require.NoError(t, err)
			assert.Subset(t, actual.Content(), test.expected(test.migrate, test.pageRequest).Content())
		})
	}
}

func TestRepositoryGetByIDNotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	db, err := config.Migrate()
	require.NoError(t, err)

	r := NewRepository(db)

	// Act
	notFoundID := users.UserID(uuid.New())
	_, err = r.GetByID(t.Context(), notFoundID)

	// Assert
	wantErr := users.NotFoundError{ID: notFoundID}
	assert.Equal(t, wantErr, err)
}
