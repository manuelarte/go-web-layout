package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActuatorsHandler_ActuatorsInfoRoute(t *testing.T) {
	t.Parallel()

	// Arrange
	r := chi.NewRouter()
	api := API{}
	ssi := NewStrictHandler(api, nil)
	HandlerFromMux(ssi, r)

	w := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/actuators/info", http.NoBody)
	require.NoError(t, err)

	// Act
	r.ServeHTTP(w, req)

	expected := Info{
		App: InfoApp{
			Description: "Example of web project layout",
			Name:        "Go-Web-Layout",
			Version:     "local",
		},
	}

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var actual Info
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &actual))
	assert.Equal(t, expected, actual)
}
