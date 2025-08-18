package users

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/manuelarte/go-web-layout/internal/config"
	"github.com/manuelarte/go-web-layout/internal/pagination"
)

func TestRepository_GetAll_Successful(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		migrate     []User
		pageRequest pagination.PageRequest
		expected    func(migrated []User, pr pagination.PageRequest) (expected pagination.Page[User])
	}{
		"empty": {
			pageRequest: pagination.MustPageRequest(0, 20),
			expected: func(migrated []User, pr pagination.PageRequest) pagination.Page[User] {
				return pagination.MustPage[User](nil, pr, 1)
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
