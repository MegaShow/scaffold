package mux

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.icytown.com/scaffold/mux/internal/assert"
)

func TestJSONRenderer_Render(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		v      any
		err    error
		status int
		body   string
	}{
		// 成功分支：渲染 v，状态码 200
		{name: "struct_value", v: struct {
			Name string `json:"name"`
		}{Name: "Alice"}, status: http.StatusOK, body: `{"name":"Alice"}`},
		{name: "slice_value", v: []int{1, 2}, status: http.StatusOK, body: `[1,2]`},
		{name: "nil_value", v: nil, status: http.StatusOK, body: `null`},
		{name: "string_with_escape", v: "line\n\"quoted\"", status: http.StatusOK, body: `"line\n\"quoted\""`},
		// 错误分支：渲染 {"error": "..."}，状态码取 StatusCode() 或默认 500
		{name: "plain_error_default_500", v: func() {}, err: errors.New("internal failure"), status: http.StatusInternalServerError, body: `{"error":"internal failure"}`},
		{name: "bad_request_status", v: func() {}, err: ErrBadRequest, status: http.StatusBadRequest, body: `{"error":"400 bad request"}`},
		{name: "unsupported_media_type_status", v: func() {}, err: errUnsupportedMediaType, status: http.StatusUnsupportedMediaType, body: `{"error":"415 unsupported media type"}`},
		{name: "wrapped_bad_request_status", v: func() {}, err: fmt.Errorf("invalid payload: %w", ErrBadRequest), status: http.StatusBadRequest, body: `{"error":"invalid payload: 400 bad request"}`},
		{name: "error_with_escape", v: func() {}, err: errors.New("line\n\"quoted\""), status: http.StatusInternalServerError, body: `{"error":"line\n\"quoted\""}`},
	}
	for _, tt := range testCases {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "text/plain")
		err := (JSONRenderer{}).Render(w, tt.v, tt.err)
		assert.Nil(t, err, tt.name)
		assert.Equal(t, tt.status, w.Code, tt.name)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"), tt.name)
		assert.Equal(t, tt.body, w.Body.String(), tt.name)
	}

	// v 不可序列化时 MarshalWrite 返回错误
	w := httptest.NewRecorder()
	assert.NotNil(t, (JSONRenderer{}).Render(w, func() {}, nil))
}
