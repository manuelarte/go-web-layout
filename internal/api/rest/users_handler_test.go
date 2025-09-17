package rest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/manuelarte/go-web-layout/internal/config"
	"github.com/manuelarte/go-web-layout/internal/users"
)

func TestUsersHandler_GetUser_Error(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		id               string
		expected         ErrorResponse
		expectedMockCall func(id string, ms *users.MockService)
	}{
		"not valid uuid": {
			id: "1",
			expected: ErrorResponse{
				Type:     "InvalidParameterValue",
				Title:    "Invalid Parameter Value",
				Detail:   "userId: error unmarshaling '1' text as *uuid.UUID: invalid UUID length: 1",
				Status:   http.StatusBadRequest,
				Instance: "00000000000000000000000000000000",
			},
			expectedMockCall: func(id string, ms *users.MockService) {
			},
		},
		"not existing user": {
			id: "08ec89b3-288c-4b38-ba25-b91c81004699",
			expected: ErrorResponse{
				Type:     "NotFound",
				Title:    "User not found",
				Detail:   "No User found with id: 08ec89b3-288c-4b38-ba25-b91c81004699",
				Status:   http.StatusNotFound,
				Instance: "00000000000000000000000000000000",
			},
			expectedMockCall: func(id string, ms *users.MockService) {
				ms.EXPECT().GetByID(gomock.Any(), gomock.Eq(uuid.MustParse(id))).Return(users.User{}, sql.ErrNoRows)
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			cfg := config.AppEnv{}
			r := chi.NewRouter()
			userService := users.NewMockService(gomock.NewController(t))
			CreateRestAPI(r, cfg, userService)

			w := httptest.NewRecorder()
			url := fmt.Sprintf("/api/v1/users/%s", test.id)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
			require.NoError(t, err)
			test.expectedMockCall(test.id, userService)

			// Act
			r.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, int(test.expected.Status), w.Code)

			expectedJSON, err := json.Marshal(test.expected)
			require.NoError(t, err)
			assert.JSONEq(t, string(expectedJSON), w.Body.String())
		})
	}
}
