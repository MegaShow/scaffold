package mux

import (
	"net/http"
	"testing"

	"go.icytown.com/scaffold/mux/internal/assert"
)

func Test_muxError_Error(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "400 bad request", ErrBadRequest.Error())
	assert.Equal(t, "415 unsupported media type", errUnsupportedMediaType.Error())
}

func Test_muxError_StatusCode(t *testing.T) {
	t.Parallel()
	assert.Equal(t, http.StatusBadRequest, ErrBadRequest.StatusCode())
	assert.Equal(t, http.StatusUnsupportedMediaType, errUnsupportedMediaType.StatusCode())
}
