package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"go.icytown.com/scaffold/internal/assert"
)

func TestGinMux(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	router := createRouter()
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users/1", nil)
	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"user_id":"1","name":"User 1"}`, w.Body.String())
}
