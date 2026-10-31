package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.icytown.com/scaffold/internal/assert"
)

func TestServeMux(t *testing.T) {
	t.Parallel()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	assert.Nil(t, writer.WriteField("name", "example"))
	part, err := writer.CreateFormFile("file", "test.bin")
	if !assert.Nil(t, err) {
		t.FailNow()
	}
	_, err = part.Write([]byte{0, 255, 128, 1})
	assert.Nil(t, err)
	for _, name := range []string{"first.bin", "second.bin"} {
		part, err := writer.CreateFormFile("files", name)
		if !assert.Nil(t, err) {
			t.FailNow()
		}
		_, err = part.Write([]byte{2, 0, 254})
		assert.Nil(t, err)
	}
	assert.Nil(t, writer.Close())

	serveMux := createServeMux()
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/files/upload", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	serveMux.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"name":"example","file":{"filename":"test.bin","size":4},"files":[{"filename":"first.bin","size":3},{"filename":"second.bin","size":3}]}`, w.Body.String())
}
