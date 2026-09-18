package mux

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go.icytown.com/scaffold/mux/form"
	"go.icytown.com/scaffold/mux/internal/assert"
)

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

func TestAdaptParser_Parse(t *testing.T) {
	t.Parallel()
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
