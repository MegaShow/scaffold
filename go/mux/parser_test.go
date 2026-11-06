package mux

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"go.icytown.com/scaffold/mux/form"
	"go.icytown.com/scaffold/mux/internal/assert"
)

type failingParser struct {
	err error
}

func (p failingParser) Parse(*http.Request, any) error {
	return p.err
}

type request struct {
	Name string   `json:"name"`
	Age  int32    `json:"age"`
	Tags []string `json:"tags"`
}

func (v *request) DecodeForm(values url.Values) error {
	d := form.NewDecoder(values)
	d.String("name", &v.Name)
	d.Int32("age", &v.Age)
	d.Strings("tags", &v.Tags)
	return d.Err()
}

type uploadRequest struct {
	Name           string
	Files          []*multipart.FileHeader
	decodeError    error
	formCalls      int
	multipartCalls int
}

func (v *uploadRequest) DecodeForm(values url.Values) error {
	v.formCalls++
	v.Name = values.Get("name")
	return v.decodeError
}

func (v *uploadRequest) DecodeMultipart(form *multipart.Form) error {
	v.multipartCalls++
	if err := v.DecodeForm(form.Value); err != nil {
		return err
	}
	v.Files = form.File["archive"]
	return nil
}

type multipartOnlyRequest struct {
	Name  string
	Files []*multipart.FileHeader
}

func (v *multipartOnlyRequest) DecodeMultipart(form *multipart.Form) error {
	v.Name = form.Value["name"][0]
	v.Files = form.File["archive"]
	return nil
}

func newUploadRequest(t *testing.T, count int) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	assert.Nil(t, w.WriteField("name", "release"))
	for i := range count {
		part, err := w.CreateFormFile("archive", fmt.Sprintf("release-%d.zip", i))
		if !assert.Nil(t, err) {
			t.FailNow()
		}
		_, err = part.Write([]byte{0, 255, 1, 128})
		assert.Nil(t, err)
	}
	assert.Nil(t, w.Close())
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", &body)
	r.Header.Set("Content-Type", w.FormDataContentType())
	return r
}

func TestAdaptParser_Parse(t *testing.T) {
	t.Parallel()
	t.Run("custom_and_default_parsers", func(t *testing.T) {
		t.Parallel()
		customError := errors.New("custom parser failed")
		custom := failingParser{err: customError}
		for _, p := range []*AdaptParser{{}, {Form: custom}, {JSON: custom}, {Form: custom, JSON: custom}, {Multipart: custom}, {Form: custom, JSON: custom, Multipart: custom}} {
			for _, contentType := range []string{"application/x-www-form-urlencoded", "application/json", "multipart/form-data"} {
				body := "name=Alice"
				wantName := "Alice"
				configured := p.Form
				if contentType == "application/json" {
					body = `{"name":"Alice"}`
					configured = p.JSON
				}
				r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
				if contentType == "multipart/form-data" {
					r = newUploadRequest(t, 1)
					configured = p.Multipart
					wantName = "release"
				} else {
					r.Header.Set("Content-Type", contentType)
				}
				t.Cleanup(func() {
					if r.MultipartForm != nil {
						assert.Nil(t, r.MultipartForm.RemoveAll())
					}
				})
				var target request
				err := p.Parse(r, &target)
				if configured != nil {
					assert.Equal(t, customError, err)
				} else {
					assert.Nil(t, err)
					assert.Equal(t, wantName, target.Name)
				}
			}
		}
	})

	testCases := []struct {
		name        string
		method      string
		path        string
		body        string
		contentType string // 空字符串表示不设置 Content-Type
		want        request
		unsupported bool // 期望返回 errUnsupportedMediaType
		err         bool // 期望返回任意错误（不含 unsupported 分支）
	}{
		// GET 一律走 Form，即使 Content-Type 是 application/json
		{name: "get_query_uses_form", method: http.MethodGet, path: "/?name=Alice&age=42&tags=a&tags=b", want: request{Name: "Alice", Age: 42, Tags: []string{"a", "b"}}},
		{name: "get_with_json_content_type_uses_form", method: http.MethodGet, path: "/?name=Alice&age=42", contentType: "application/json", want: request{Name: "Alice", Age: 42}},
		{name: "get_form_decode_error_passthrough", method: http.MethodGet, path: "/?age=bad", want: request{}, err: true},
		// POST 表单走 Form
		{name: "post_form_urlencoded", method: http.MethodPost, path: "/", body: "name=Alice&age=42", contentType: "application/x-www-form-urlencoded", want: request{Name: "Alice", Age: 42}},
		{name: "post_form_urlencoded_with_charset", method: http.MethodPost, path: "/", body: "name=Alice&age=42", contentType: "application/x-www-form-urlencoded; charset=utf-8", want: request{Name: "Alice", Age: 42}},
		// POST JSON 走 JSON
		{name: "post_json", method: http.MethodPost, path: "/", body: `{"name":"Alice","age":42,"tags":["a","b"]}`, contentType: "application/json", want: request{Name: "Alice", Age: 42, Tags: []string{"a", "b"}}},
		{name: "post_json_uppercase", method: http.MethodPost, path: "/", body: `{"name":"Alice","age":42}`, contentType: "APPLICATION/JSON", want: request{Name: "Alice", Age: 42}},
		{name: "post_json_with_charset", method: http.MethodPost, path: "/", body: `{"name":"Alice","age":42}`, contentType: "application/json; charset=utf-8", want: request{Name: "Alice", Age: 42}},
		{name: "post_json_syntax_error_passthrough", method: http.MethodPost, path: "/", body: `{"age":`, contentType: "application/json", want: request{}, err: true},
		// 不支持的媒体类型
		{name: "post_unsupported", method: http.MethodPost, path: "/", body: "name=Alice", contentType: "text/plain", want: request{}, unsupported: true},
		{name: "post_empty_content_type", method: http.MethodPost, path: "/", body: "name=Alice", contentType: "", want: request{}, unsupported: true},
		// PUT/DELETE
		{name: "put_form_urlencoded", method: http.MethodPut, path: "/", body: "name=Alice&age=42", contentType: "application/x-www-form-urlencoded", want: request{Name: "Alice", Age: 42}},
		{name: "put_json", method: http.MethodPut, path: "/", body: `{"name":"Alice","age":42}`, contentType: "application/json", want: request{Name: "Alice", Age: 42}},
		{name: "delete_unsupported", method: http.MethodDelete, path: "/", body: "name=Alice", contentType: "text/plain", want: request{}, unsupported: true},
	}
	for _, tt := range testCases {
		p := &AdaptParser{Form: FormParser{}, JSON: JSONParser{}}
		r := httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, strings.NewReader(tt.body))
		if tt.contentType != "" {
			r.Header.Set("Content-Type", tt.contentType)
		}
		var target request
		err := p.Parse(r, &target)
		if tt.unsupported {
			assert.Equal(t, error(errUnsupportedMediaType), err, tt.name)
		} else {
			assert.Equal(t, tt.err, err != nil, tt.name)
		}
		assert.Equal(t, tt.want, target, tt.name)
	}
}

func TestFormParser_Parse(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		method string
		path   string
		body   string
		want   request
		err    bool
	}{
		{name: "query", method: http.MethodGet, path: "/?name=Alice+Smith&age=42&tags=a&tags=b", want: request{Name: "Alice Smith", Age: 42, Tags: []string{"a", "b"}}},
		{name: "body", method: http.MethodPost, path: "/", body: "name=Alice%20Smith&age=42&tags=a&tags=b", want: request{Name: "Alice Smith", Age: 42, Tags: []string{"a", "b"}}},
		{name: "query_and_body", method: http.MethodPost, path: "/?name=query&age=42", body: "name=body", want: request{Name: "body", Age: 42}},
		{name: "empty", method: http.MethodGet, path: "/", want: request{}},
		{name: "preserve", method: http.MethodGet, path: "/?name=patched", want: request{Name: "patched"}},
		{name: "invalid_query", method: http.MethodGet, path: "/?name=%zz", want: request{}, err: true},
		{name: "invalid_body", method: http.MethodPost, path: "/", body: "name=%zz", want: request{}, err: true},
		{name: "invalid_decode", method: http.MethodGet, path: "/?age=bad", want: request{}, err: true},
	}
	for _, tt := range testCases {
		r := httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, strings.NewReader(tt.body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		var target request
		err := (FormParser{}).Parse(r, &target)
		assert.Equal(t, tt.err, err != nil, tt.name)
		assert.Equal(t, tt.want, target, tt.name)
	}

	// 非 Decodable 类型：不应走到此分支
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	var unsupported struct{ Name string }
	assert.NotNil(t, (FormParser{}).Parse(r, &unsupported))
}

func TestJSONParser_Parse(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		body string
		want request
		err  bool
	}{
		{name: "valid", body: `{"name":"Alice","age":42,"tags":["a","b"]}`, want: request{Name: "Alice", Age: 42, Tags: []string{"a", "b"}}},
		{name: "empty", body: "", err: true},
		{name: "invalid_syntax", body: `{"age":`, err: true},
		{name: "type_mismatch", body: `{"age":"invalid"}`, err: true},
	}
	for _, tt := range testCases {
		target := request{}
		r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(tt.body))
		err := (JSONParser{}).Parse(r, &target)
		assert.Equal(t, tt.err, err != nil, tt.name)
		assert.Equal(t, tt.want, target, tt.name)
	}
}

func TestMultipartParser_Parse(t *testing.T) {
	t.Run("partial_parse_failure_removes_disk_files", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("TMP", dir)
		t.Setenv("TEMP", dir)
		t.Setenv("TMPDIR", dir)
		for _, limited := range []bool{false, true} {
			r := newUploadRequest(t, 1)
			parser := MultipartParser{MaxMemory: -1}
			if limited {
				parser.MaxBytes = r.ContentLength - 1
				r.ContentLength = -1
			} else {
				body, err := io.ReadAll(r.Body)
				assert.Nil(t, err)
				r.Body = io.NopCloser(bytes.NewReader(body[:len(body)-10]))
			}
			assert.NotNil(t, parser.Parse(r, &uploadRequest{}))
			entries, err := os.ReadDir(dir)
			assert.Nil(t, err)
			assert.Equal(t, 0, len(entries), "partial parse must remove temporary files")
		}
	})
	t.Run("chunked_epilogue_counts_toward_total_limit", func(t *testing.T) {
		t.Parallel()
		r := newUploadRequest(t, 1)
		limit := r.ContentLength + 10
		r.Body = io.NopCloser(io.MultiReader(r.Body, bytes.NewReader(bytes.Repeat([]byte("x"), 8192))))
		r.ContentLength = -1
		err := (MultipartParser{MaxMemory: -1, MaxBytes: limit}).Parse(r, &uploadRequest{})
		assert.Equal(t, error(errContentTooLarge), err)
		if r.MultipartForm != nil {
			_, err = r.MultipartForm.File["archive"][0].Open()
			assert.True(t, errors.Is(err, os.ErrNotExist))
		}
	})
	for _, count := range []int{0, 1, 2} {
		t.Run(fmt.Sprintf("files_%d", count), func(t *testing.T) {
			t.Parallel()
			r := newUploadRequest(t, count)
			defer func() {
				if r.MultipartForm != nil {
					assert.Nil(t, r.MultipartForm.RemoveAll())
				}
			}()
			var target uploadRequest
			assert.Nil(t, (MultipartParser{MaxMemory: -1}).Parse(r, &target))
			assert.Equal(t, "release", target.Name)
			assert.Equal(t, 1, target.multipartCalls)
			assert.Equal(t, 1, target.formCalls)
			assert.Equal(t, count, len(target.Files))
			for i, header := range target.Files {
				assert.Equal(t, fmt.Sprintf("release-%d.zip", i), header.Filename)
				assert.Equal(t, int64(4), header.Size)
				file, err := header.Open()
				if !assert.Nil(t, err) {
					t.FailNow()
				}
				data, err := io.ReadAll(file)
				assert.Nil(t, err)
				assert.Nil(t, file.Close())
				assert.Equal(t, []byte{0, 255, 1, 128}, data)
			}
		})
	}
	t.Run("multipart_decoder_without_form_decoder", func(t *testing.T) {
		t.Parallel()
		r := newUploadRequest(t, 1)
		defer func() {
			if r.MultipartForm != nil {
				assert.Nil(t, r.MultipartForm.RemoveAll())
			}
		}()
		var target multipartOnlyRequest
		assert.Nil(t, (MultipartParser{}).Parse(r, &target))
		assert.Equal(t, "release", target.Name)
		assert.Equal(t, 1, len(target.Files))
	})
	t.Run("form_decoder_fallback", func(t *testing.T) {
		t.Parallel()
		r := newUploadRequest(t, 0)
		defer func() {
			if r.MultipartForm != nil {
				assert.Nil(t, r.MultipartForm.RemoveAll())
			}
		}()
		var target request
		assert.Nil(t, (MultipartParser{}).Parse(r, &target))
		assert.Equal(t, "release", target.Name)
	})
	t.Run("unsupported_target_removes_disk_file", func(t *testing.T) {
		t.Parallel()
		r := newUploadRequest(t, 1)
		err := (MultipartParser{MaxMemory: -1}).Parse(r, &struct{}{})
		if !assert.NotNil(t, err) {
			t.FailNow()
		}
		assert.Equal(t, "not implementation DecodeMultipart or DecodeForm", err.Error())
		_, err = r.MultipartForm.File["archive"][0].Open()
		assert.True(t, errors.Is(err, os.ErrNotExist))
	})
	t.Run("decode_failure_removes_disk_file", func(t *testing.T) {
		t.Parallel()
		r := newUploadRequest(t, 1)
		target := uploadRequest{decodeError: errors.New("invalid field")}
		assert.Equal(t, target.decodeError, (MultipartParser{MaxMemory: -1}).Parse(r, &target))
		_, err := r.MultipartForm.File["archive"][0].Open()
		assert.True(t, errors.Is(err, os.ErrNotExist))
	})
	t.Run("invalid_boundary", func(t *testing.T) {
		t.Parallel()
		r := newUploadRequest(t, 1)
		r.Header.Set("Content-Type", "multipart/form-data")
		assert.NotNil(t, (MultipartParser{}).Parse(r, &uploadRequest{}))
	})
	t.Run("total_size_limit", func(t *testing.T) {
		t.Parallel()
		r := newUploadRequest(t, 1)
		err := (MultipartParser{MaxBytes: 20}).Parse(r, &uploadRequest{})
		assert.Equal(t, error(errContentTooLarge), err)
	})
	t.Run("multipart_message_too_large", func(t *testing.T) {
		t.Parallel()
		// 普通字段不能落盘，超过内存预算时触发 multipart.ErrMessageTooLarge
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		assert.Nil(t, writer.WriteField("name", strings.Repeat("x", 11<<20)))
		assert.Nil(t, writer.Close())
		r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", &body)
		r.Header.Set("Content-Type", writer.FormDataContentType())
		err := (MultipartParser{MaxMemory: 1}).Parse(r, &uploadRequest{})
		assert.Equal(t, error(errContentTooLarge), err)
	})
}
