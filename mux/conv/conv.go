// Package conv 提供 path 和 form 共用的字段值转换函数，不负责取值或判断字段是否缺失。
package conv

import (
	"fmt"
	"strconv"
)

// ParseBool 将 1、true 解析为 true，将 0、false 解析为 false。
func ParseBool(s string) (bool, error) {
	switch s {
	case "1", "true":
		return true, nil
	case "0", "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value '%s'", s)
	}
}

// ParseBytes 将字符串直接转换为字节切片。
func ParseBytes(s string) ([]byte, error) {
	return []byte(s), nil
}

// ParseEnum 按十进制解析底层类型为 int32 的枚举，不限制枚举定义之外的数值。
func ParseEnum[T ~int32](s string) (T, error) {
	value, err := ParseInt32(s)
	return T(value), err
}

// ParseFloat32 解析 32 位浮点数，超出范围时返回错误。
func ParseFloat32(s string) (float32, error) {
	value, err := strconv.ParseFloat(s, 32)
	return float32(value), err
}

// ParseFloat64 解析 64 位浮点数，超出范围时返回错误。
func ParseFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// ParseInt32 按十进制解析 32 位有符号整数。
func ParseInt32(s string) (int32, error) {
	value, err := strconv.ParseInt(s, 10, 32)
	return int32(value), err
}

// ParseInt64 按十进制解析 64 位有符号整数。
func ParseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// ParseString 原样返回字符串，与其他转换函数保持一致的调用签名。
func ParseString(s string) (string, error) {
	return s, nil
}

// ParseUint32 按十进制解析 32 位无符号整数。
func ParseUint32(s string) (uint32, error) {
	value, err := strconv.ParseUint(s, 10, 32)
	return uint32(value), err
}

// ParseUint64 按十进制解析 64 位无符号整数。
func ParseUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}
