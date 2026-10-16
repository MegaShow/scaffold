package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.icytown.com/scaffold/internal/assert"
)

func TestServeMux(t *testing.T) {
	t.Parallel()
	serveMux := createServeMux()
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users/1", nil)
	serveMux.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "GetUser", w.Header().Get("X-Scaffold-Method"))
	assert.Equal(t, `{"user_id":"1","name":"User 1"}`, w.Body.String())
}
