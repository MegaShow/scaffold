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
	value, err := ParseBytes("hello")
	assert.Nil(t, err)
	assert.Equal(t, []byte("hello"), value)

	value, err = ParseBytes("")
	assert.Nil(t, err)
	assert.Equal(t, []byte{}, value)

	// 中文字符
	value, err = ParseBytes("你好")
	assert.Nil(t, err)
	assert.Equal(t, []byte("你好"), value)
}

func TestParseEnum(t *testing.T) {
	t.Parallel()
	type status int32

	// 合法数值
	value, err := ParseEnum[status]("0")
	assert.Nil(t, err)
	assert.Equal(t, status(0), value)

	value, err = ParseEnum[status]("42")
	assert.Nil(t, err)
	assert.Equal(t, status(42), value)

	// 负数
	value, err = ParseEnum[status]("-1")
	assert.Nil(t, err)
	assert.Equal(t, status(-1), value)

	// 超出 int32 范围
	_, err = ParseEnum[status]("2147483648")
	assert.NotNil(t, err)

	// 非数字
	_, err = ParseEnum[status]("abc")
	assert.NotNil(t, err)
}

func TestParseFloat32(t *testing.T) {
	t.Parallel()
	// 普通数值
	value, err := ParseFloat32("3.14")
	assert.Nil(t, err)
	assert.Equal(t, float32(3.14), value)

	// 整数形式
	value, err = ParseFloat32("10")
	assert.Nil(t, err)
	assert.Equal(t, float32(10), value)

	// 科学计数法
	value, err = ParseFloat32("1e3")
	assert.Nil(t, err)
	assert.Equal(t, float32(1000), value)

	// 负数
	value, err = ParseFloat32("-2.5")
	assert.Nil(t, err)
	assert.Equal(t, float32(-2.5), value)

	// 超出 float32 范围
	_, err = ParseFloat32("1e39")
	assert.NotNil(t, err)

	// 非数字
	_, err = ParseFloat32("abc")
	assert.NotNil(t, err)
}

func TestParseFloat64(t *testing.T) {
	t.Parallel()
	// 普通数值
	value, err := ParseFloat64("3.141592653589793")
	assert.Nil(t, err)
	assert.Equal(t, 3.141592653589793, value)

	// float32 超范围但 float64 合法的数值
	value, err = ParseFloat64("1e39")
	assert.Nil(t, err)
	assert.Equal(t, 1e39, value)

	// 超出 float64 范围
	_, err = ParseFloat64("1e309")
	assert.NotNil(t, err)

	// 非数字
	_, err = ParseFloat64("abc")
	assert.NotNil(t, err)
}

func TestParseInt32(t *testing.T) {
	t.Parallel()
	// 合法数值
	value, err := ParseInt32("0")
	assert.Nil(t, err)
	assert.Equal(t, int32(0), value)

	value, err = ParseInt32("2147483647")
	assert.Nil(t, err)
	assert.Equal(t, int32(math.MaxInt32), value)

	value, err = ParseInt32("-2147483648")
	assert.Nil(t, err)
	assert.Equal(t, int32(math.MinInt32), value)

	// 超出范围
	_, err = ParseInt32("2147483648")
	assert.NotNil(t, err)

	_, err = ParseInt32("-2147483649")
	assert.NotNil(t, err)

	// 非十进制形式
	for _, s := range []string{"0x10", "1.5", "abc", "", "1e3"} {
		_, err = ParseInt32(s)
		assert.NotNil(t, err)
	}
}

func TestParseInt64(t *testing.T) {
	t.Parallel()
	// 合法数值
	value, err := ParseInt64("0")
	assert.Nil(t, err)
	assert.Equal(t, int64(0), value)

	value, err = ParseInt64("9223372036854775807")
	assert.Nil(t, err)
	assert.Equal(t, int64(math.MaxInt64), value)

	value, err = ParseInt64("-9223372036854775808")
	assert.Nil(t, err)
	assert.Equal(t, int64(math.MinInt64), value)

	// 超出范围
	_, err = ParseInt64("9223372036854775808")
	assert.NotNil(t, err)

	// 非数字
	_, err = ParseInt64("abc")
	assert.NotNil(t, err)
}

func TestParseString(t *testing.T) {
	t.Parallel()
	value, err := ParseString("hello")
	assert.Nil(t, err)
	assert.Equal(t, "hello", value)

	value, err = ParseString("")
	assert.Nil(t, err)
	assert.Equal(t, "", value)
}

func TestParseUint32(t *testing.T) {
	t.Parallel()
	// 合法数值
	value, err := ParseUint32("0")
	assert.Nil(t, err)
	assert.Equal(t, uint32(0), value)

	value, err = ParseUint32("4294967295")
	assert.Nil(t, err)
	assert.Equal(t, uint32(math.MaxUint32), value)

	// 负数
	_, err = ParseUint32("-1")
	assert.NotNil(t, err)

	// 超出范围
	_, err = ParseUint32("4294967296")
	assert.NotNil(t, err)

	// 非数字
	_, err = ParseUint32("abc")
	assert.NotNil(t, err)
}

func TestParseUint64(t *testing.T) {
	t.Parallel()
	// 合法数值
	value, err := ParseUint64("0")
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), value)

	value, err = ParseUint64("18446744073709551615")
	assert.Nil(t, err)
	assert.Equal(t, uint64(math.MaxUint64), value)

	// 负数
	_, err = ParseUint64("-1")
	assert.NotNil(t, err)

	// 超出范围
	_, err = ParseUint64("18446744073709551616")
	assert.NotNil(t, err)

	// 非数字
	_, err = ParseUint64("abc")
	assert.NotNil(t, err)
}
