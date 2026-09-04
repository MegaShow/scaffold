package path

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.icytown.com/scaffold/mux/internal/assert"
)

// newTestDecoder 创建带 field 路径参数的解码器；不传值表示参数缺失。
func newTestDecoder(t *testing.T, value ...string) *Decoder {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/?field=ignored", nil)
	if len(value) > 0 {
		r.SetPathValue("field", value[0])
	}
	return NewDecoder(r)
}

func TestNewDecoder(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	d := NewDecoder(r)
	assert.NotNil(t, d)
}

func TestDecoder_Bool(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		value string
		want  bool
		err   bool
	}{
		{name: "valid_true", value: "true", want: true},
		{name: "valid_false", value: "false", want: false},
		{name: "missing", want: false, err: true},
		{name: "numeric_one", value: "1", want: true},
		{name: "numeric_zero", value: "0", want: false},
		{name: "invalid_number", value: "123", want: false, err: true},
		{name: "invalid", value: "invalid", want: false, err: true},
	}
	for _, tt := range testCases {
		var got bool
		d := newTestDecoder(t, tt.value)
		d.Bool("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Bytes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		value string
		want  []byte
		err   bool
	}{
		{name: "valid", value: "你好", want: []byte("你好")},
		{name: "missing", want: nil, err: true},
	}
	for _, tt := range testCases {
		var got []byte
		d := newTestDecoder(t, tt.value)
		d.Bytes("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Enum(t *testing.T) {
	t.Parallel()
	type status int32
	testCases := []struct {
		name  string
		value string
		want  status
		err   bool
	}{
		{name: "valid", value: "42", want: 42},
		{name: "missing", want: 0, err: true},
		{name: "overflow", value: "2147483648", want: 0, err: true},
	}
	for _, tt := range testCases {
		var got status
		d := newTestDecoder(t, tt.value)
		d.Enum("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Err(t *testing.T) {
	t.Parallel()
	d, v, f := newTestDecoder(t, "123"), int32(0), float32(0)
	assert.Nil(t, d.Err())
	d.Int32("field", &v)
	assert.Nil(t, d.Err())
	d.Int32("missing", &v)
	assert.NotNil(t, d.Err())
	err := d.Err()
	d.Float32("field", &f)
	assert.Equal(t, err, d.Err())
}

func TestDecoder_Float32(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		value string
		want  float32
		err   bool
	}{
		{name: "valid", value: "-1.25", want: -1.25},
		{name: "missing", want: 0, err: true},
		{name: "overflow", value: "1e39", want: 0, err: true},
		{name: "invalid", value: "invalid", want: 0, err: true},
	}
	for _, tt := range testCases {
		var got float32
		d := newTestDecoder(t, tt.value)
		d.Float32("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Float64(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		value string
		want  float64
		err   bool
	}{
		{name: "valid_decimal", value: "1.25", want: 1.25},
		{name: "valid_exponent", value: "1e100", want: 1e100},
		{name: "missing", want: 0, err: true},
		{name: "overflow", value: "1e309", want: 0, err: true},
		{name: "invalid", value: "invalid", want: 0, err: true},
	}
	for _, tt := range testCases {
		var got float64
		d := newTestDecoder(t, tt.value)
		d.Float64("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Int32(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		value string
		want  int32
		err   bool
	}{
		{name: "valid", value: "12", want: 12},
		{name: "min_value", value: "-2147483648", want: -2147483648},
		{name: "missing", want: 0, err: true},
		{name: "overflow", value: "2147483648", want: 0, err: true},
		{name: "invalid", value: "invalid", want: 0, err: true},
	}
	for _, tt := range testCases {
		var got int32
		d := newTestDecoder(t, tt.value)
		d.Int32("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Int64(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		value string
		want  int64
		err   bool
	}{
		{name: "valid", value: "123", want: 123},
		{name: "min_value", value: "-9223372036854775808", want: -9223372036854775808},
		{name: "missing", want: 0, err: true},
		{name: "overflow", value: "9223372036854775808", want: 0, err: true},
		{name: "invalid", value: "invalid", want: 0, err: true},
	}
	for _, tt := range testCases {
		var got int64
		d := newTestDecoder(t, tt.value)
		d.Int64("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_String(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		value string
		want  string
		err   bool
	}{
		{name: "valid", value: "hello", want: "hello"},
		{name: "missing", want: "", err: true},
	}
	for _, tt := range testCases {
		var got string
		d := newTestDecoder(t, tt.value)
		d.String("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Uint32(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		value string
		want  uint32
		err   bool
	}{
		{name: "valid", value: "123", want: 123},
		{name: "max_value", value: "4294967295", want: 4294967295},
		{name: "missing", want: 0, err: true},
		{name: "negative", value: "-1", want: 0, err: true},
		{name: "invalid", value: "invalid", want: 0, err: true},
	}
	for _, tt := range testCases {
		var got uint32
		d := newTestDecoder(t, tt.value)
		d.Uint32("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Uint64(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		value string
		want  uint64
		err   bool
	}{
		{name: "valid", value: "123", want: 123},
		{name: "max_value", value: "18446744073709551615", want: 18446744073709551615},
		{name: "missing", want: 0, err: true},
		{name: "overflow", value: "18446744073709551616", want: 0, err: true},
		{name: "invalid", value: "invalid", want: 0, err: true},
	}
	for _, tt := range testCases {
		var got uint64
		d := newTestDecoder(t, tt.value)
		d.Uint64("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}
