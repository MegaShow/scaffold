// Package path 提供生成代码使用的类型安全路径参数解码器。
package path

import (
	"fmt"
	"net/http"

	"go.icytown.com/scaffold/mux/conv"
)

// Decoder 保存请求、当前路由的参数名和首个解析错误。
type Decoder struct {
	request *http.Request
	err     error
}

// NewDecoder 创建路径解码器。
func NewDecoder(r *http.Request) *Decoder {
	return &Decoder{request: r}
}

// Bool 解析 bool 路径字段。
func (d *Decoder) Bool(name string, v *bool) {
	d.scalar(name, v, conv.ParseBool)
}

// Bytes 解析 []byte 路径字段。
func (d *Decoder) Bytes(name string, v *[]byte) {
	d.scalar(name, v, conv.ParseBytes)
}

// Enum 解析底层类型为 int32 的枚举字段。
func (d *Decoder) Enum[T ~int32](name string, v *T) {
	d.scalar(name, v, conv.ParseEnum[T])
}

// Err 返回首个解析错误。
func (d *Decoder) Err() error {
	return d.err
}

// Float32 解析 float32 路径字段。
func (d *Decoder) Float32(name string, v *float32) {
	d.scalar(name, v, conv.ParseFloat32)
}

// Float64 解析 float64 路径字段。
func (d *Decoder) Float64(name string, v *float64) {
	d.scalar(name, v, conv.ParseFloat64)
}

// Int32 解析 int32 路径字段。
func (d *Decoder) Int32(name string, v *int32) {
	d.scalar(name, v, conv.ParseInt32)
}

// Int64 解析 int64 路径字段。
func (d *Decoder) Int64(name string, v *int64) {
	d.scalar(name, v, conv.ParseInt64)
}

// String 解析string 路径字段。
func (d *Decoder) String(name string, v *string) {
	d.scalar(name, v, conv.ParseString)
}

// Uint32 解析uint32 路径字段。
func (d *Decoder) Uint32(name string, v *uint32) {
	d.scalar(name, v, conv.ParseUint32)
}

// Uint64 解析uint64 路径字段。
func (d *Decoder) Uint64(name string, v *uint64) {
	d.scalar(name, v, conv.ParseUint64)
}

// scalar 按生成代码的字段调用顺序解码；必需参数为空时记录错误，后续字段不再修改。
func (d *Decoder) scalar[T any](name string, v *T, parse func(string) (T, error)) {
	if d.err != nil {
		return
	}
	text := d.request.PathValue(name)
	if text == "" {
		d.err = fmt.Errorf("missing path parameter '%s'", name)
		return
	}
	value, err := parse(text)
	if err != nil {
		d.err = fmt.Errorf("decode path parameter '%s': %w", name, err)
		return
	}
	*v = value
}
