package conv

import (
	"math"
	"testing"

	"go.icytown.com/scaffold/mux/internal/assert"
)

func TestParseBool(t *testing.T) {
	t.Parallel()
	// 合法取值
	for _, s := range []string{"1", "true"} {
		value, err := ParseBool(s)
		assert.Nil(t, err)
		assert.True(t, value)
	}
	for _, s := range []string{"0", "false"} {
		value, err := ParseBool(s)
		assert.Nil(t, err)
		assert.False(t, value)
	}
	// 非法取值
	for _, s := range []string{"", "2", "True", "FALSE", "yes", "no"} {
		_, err := ParseBool(s)
		assert.NotNil(t, err)
	}
}

func TestParseBytes(t *testing.T) {
	t.Parallel()
	// 合法取值
	testCases := []struct {
		arg  string
		want []byte
	}{
		{arg: "hello", want: []byte("hello")},
		{arg: "", want: []byte{}},
		{arg: "你好", want: []byte("你好")},
	}
	for _, tt := range testCases {
		value, err := ParseBytes(tt.arg)
		assert.Nil(t, err)
		assert.Equal(t, tt.want, value)
	}
}

func TestParseEnum(t *testing.T) {
	t.Parallel()
	type status int32
	// 合法取值
	testCases := []struct {
		arg  string
		want status
	}{
		{arg: "0", want: 0},
		{arg: "42", want: 42},
		{arg: "-1", want: -1},
	}
	for _, tt := range testCases {
		value, err := ParseEnum[status](tt.arg)
		assert.Nil(t, err)
		assert.Equal(t, tt.want, value)
	}
	// 非法取值
	for _, s := range []string{"2147483648", "abc"} {
		_, err := ParseEnum[status](s)
		assert.NotNil(t, err)
	}
}

func TestParseFloat32(t *testing.T) {
	t.Parallel()
	// 合法取值
	testCases := []struct {
		arg  string
		want float32
	}{
		{arg: "3.14", want: 3.14},
		{arg: "10", want: 10},
		{arg: "1e3", want: 1000},
		{arg: "-2.5", want: -2.5},
	}
	for _, tt := range testCases {
		value, err := ParseFloat32(tt.arg)
		assert.Nil(t, err)
		assert.Equal(t, tt.want, value)
	}
	// 非法取值
	for _, s := range []string{"1e39", "abc"} {
		_, err := ParseFloat32(s)
		assert.NotNil(t, err)
	}
}

func TestParseFloat64(t *testing.T) {
	t.Parallel()
	// 合法取值
	testCases := []struct {
		arg  string
		want float64
	}{
		{arg: "3.141592653589793", want: 3.141592653589793},
		{arg: "1e39", want: 1e39},
	}
	for _, tt := range testCases {
		value, err := ParseFloat64(tt.arg)
		assert.Nil(t, err)
		assert.Equal(t, tt.want, value)
	}
	// 非法取值
	for _, s := range []string{"1e309", "abc"} {
		_, err := ParseFloat64(s)
		assert.NotNil(t, err)
	}
}

func TestParseInt32(t *testing.T) {
	t.Parallel()
	// 合法取值
	testCases := []struct {
		arg  string
		want int32
	}{
		{arg: "0", want: 0},
		{arg: "2147483647", want: math.MaxInt32},
		{arg: "-2147483648", want: math.MinInt32},
	}
	for _, tt := range testCases {
		value, err := ParseInt32(tt.arg)
		assert.Nil(t, err)
		assert.Equal(t, tt.want, value)
	}
	// 非法取值
	for _, s := range []string{"2147483648", "-2147483649", "0x10", "1.5", "abc", "", "1e3"} {
		_, err := ParseInt32(s)
		assert.NotNil(t, err)
	}
}

func TestParseInt64(t *testing.T) {
	t.Parallel()
	// 合法取值
	testCases := []struct {
		arg  string
		want int64
	}{
		{arg: "0", want: 0},
		{arg: "9223372036854775807", want: math.MaxInt64},
		{arg: "-9223372036854775808", want: math.MinInt64},
	}
	for _, tt := range testCases {
		value, err := ParseInt64(tt.arg)
		assert.Nil(t, err)
		assert.Equal(t, tt.want, value)
	}
	// 非法取值
	for _, s := range []string{"9223372036854775808", "abc"} {
		_, err := ParseInt64(s)
		assert.NotNil(t, err)
	}
}

func TestParseString(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		arg  string
		want string
	}{
		{arg: "hello", want: "hello"},
		{arg: "", want: ""},
	}
	for _, tt := range testCases {
		value, err := ParseString(tt.arg)
		assert.Nil(t, err)
		assert.Equal(t, tt.want, value)
	}
}

func TestParseUint32(t *testing.T) {
	t.Parallel()
	// 合法取值
	testCases := []struct {
		arg  string
		want uint32
	}{
		{arg: "0", want: 0},
		{arg: "4294967295", want: math.MaxUint32},
	}
	for _, tt := range testCases {
		value, err := ParseUint32(tt.arg)
		assert.Nil(t, err)
		assert.Equal(t, tt.want, value)
	}
	// 非法取值
	for _, s := range []string{"-1", "4294967296", "abc"} {
		_, err := ParseUint32(s)
		assert.NotNil(t, err)
	}
}

func TestParseUint64(t *testing.T) {
	t.Parallel()
	// 合法取值
	testCases := []struct {
		arg  string
		want uint64
	}{
		{arg: "0", want: 0},
		{arg: "18446744073709551615", want: math.MaxUint64},
	}
	for _, tt := range testCases {
		value, err := ParseUint64(tt.arg)
		assert.Nil(t, err)
		assert.Equal(t, tt.want, value)
	}
	// 非法取值
	for _, s := range []string{"-1", "18446744073709551616", "abc"} {
		_, err := ParseUint64(s)
		assert.NotNil(t, err)
	}
}
