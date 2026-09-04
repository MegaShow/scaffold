package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.icytown.com/scaffold/internal/assert"
)

func TestServeMux(t *testing.T) {
	t.Parallel()
	serveMux := createServeMux()
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/users", strings.NewReader(`{"email":"test@example.com","name":"test user"}`))
	r.Header.Set("Content-Type", "application/json")
	serveMux.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"user_id":1,"name":"test user"}`, w.Body.String())

	w = httptest.NewRecorder()
	r = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users/1", nil)
	serveMux.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"user_id":1,"name":"test user"}`, w.Body.String())

	w = httptest.NewRecorder()
	r = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users/2", nil)
	serveMux.ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, `{"error":"user not found"}`, w.Body.String())

	w = httptest.NewRecorder()
	r = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users/123?ts=456", nil)
	serveMux.ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, `{"error":"no permission, ts: 456"}`, w.Body.String())
}
