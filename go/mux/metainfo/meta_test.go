package metainfo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.icytown.com/scaffold/mux/internal/assert"
)

func TestGetMeta(t *testing.T) {
	t.Parallel()
	assert.Nil(t, GetMeta(context.Background()))
	meta := New("Service", "Test", nil)
	assert.Equal(t, meta, GetMeta(WithMeta(context.Background(), meta)))
}

func TestMiddleware(t *testing.T) {
	t.Parallel()
	meta := New("Service", "Test", nil)
	var received Meta
	handler := Middleware(meta)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = GetMeta(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, meta, received)
}

func TestNew(t *testing.T) {
	t.Parallel()
	options := map[string]map[string]any{"dev.access": {"role": "editor"}}
	meta := New("Service", "Test", options)
	assert.Equal(t, "Service", meta.ServiceName())
	assert.Equal(t, "Test", meta.MethodName())
	assert.Equal(t, options["dev.access"], meta.MethodOption("dev.access"))
}

func TestWithMeta(t *testing.T) {
	t.Parallel()
	meta := New("Service", "Test", nil)
	ctx := WithMeta(context.Background(), meta)
	assert.Equal(t, meta, GetMeta(ctx))
}

func Test_meta_MethodName(t *testing.T) {
	t.Parallel()
	meta := New("Service", "Test", nil)
	assert.Equal(t, "Test", meta.MethodName())
}

func Test_meta_MethodOption(t *testing.T) {
	t.Parallel()
	options := map[string]map[string]any{
		"dev.access": {"role": "editor", "flags": []any{true, []byte("x")}},
	}
	meta := New("Service", "Test", options)
	assert.Nil(t, meta.MethodOption("missing"))
	assert.Nil(t, meta.MethodOption("dev"))
	assert.Equal(t, options["dev.access"], meta.MethodOption("dev.access"))
}

func Test_meta_ServiceName(t *testing.T) {
	t.Parallel()
	meta := New("Service", "Test", nil)
	assert.Equal(t, "Service", meta.ServiceName())
}
