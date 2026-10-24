package mux

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.icytown.com/scaffold/mux/internal/assert"
)

func TestChain(t *testing.T) {
	t.Parallel()
	var order []string
	first := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "first")
			next.ServeHTTP(w, r)
		})
	}
	second := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "second")
			next.ServeHTTP(w, r)
		})
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	Chain(handler, first, second).ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, []string{"first", "second", "handler"}, order)
}
