// Package form 提供生成代码使用的类型安全表单解码函数。
package form

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"go.icytown.com/scaffold/mux/conv"
)

type fieldParser[T any] func(name string) (T, error)

// Decodable 表示能够将表单数据解码到自身的类型。
type Decodable interface {
	DecodeForm(form url.Values) error
}

// Decoder 保存表单和首个解析错误；发生错误后，后续字段不会再被修改。
type Decoder struct {
	values url.Values
	err    error
}

// NewDecoder 创建一次表单解码所需的状态。
func NewDecoder(values url.Values) *Decoder {
	return &Decoder{values: values}
}

// Bool 解析布尔字段。
func (d *Decoder) Bool(name string, v *bool) {
	d.scalar(name, v, conv.ParseBool)
}

// Bools 解析布尔数组，保留空元素的位置。
func (d *Decoder) Bools(name string, v *[]bool) {
	d.repeated(name, v, conv.ParseBool)
}

// Bytes 解析字节字段。
func (d *Decoder) Bytes(name string, v *[]byte) {
	d.scalar(name, v, conv.ParseBytes)
}

// BytesList 解析字节数组。
func (d *Decoder) BytesList(name string, v *[][]byte) {
	d.repeated(name, v, conv.ParseBytes)
}

// Enum 解析底层类型为 int32 的枚举。
func (d *Decoder) Enum[T ~int32](name string, v *T) {
	d.scalar(name, v, conv.ParseEnum[T])
}

// Enums 解析枚举数组。
func (d *Decoder) Enums[T ~int32](name string, v *[]T) {
	d.repeated(name, v, conv.ParseEnum[T])
}

// Err 返回首个解析错误。
func (d *Decoder) Err() error {
	return d.err
}

// Float32 解析数值字段。
func (d *Decoder) Float32(name string, v *float32) {
	d.scalar(name, v, conv.ParseFloat32)
}

// Float32s 解析数值数组。
func (d *Decoder) Float32s(name string, v *[]float32) {
	d.repeated(name, v, conv.ParseFloat32)
}

// Float64 解析数值字段。
func (d *Decoder) Float64(name string, v *float64) {
	d.scalar(name, v, conv.ParseFloat64)
}

// Float64s 解析数值数组。
func (d *Decoder) Float64s(name string, v *[]float64) {
	d.repeated(name, v, conv.ParseFloat64)
}

// Int32 解析数值字段。
func (d *Decoder) Int32(name string, v *int32) {
	d.scalar(name, v, conv.ParseInt32)
}

// Int32s 解析数值数组。
func (d *Decoder) Int32s(name string, v *[]int32) {
	d.repeated(name, v, conv.ParseInt32)
}

// Int64 解析数值字段。
func (d *Decoder) Int64(name string, v *int64) {
	d.scalar(name, v, conv.ParseInt64)
}

// Int64s 解析数值数组。
func (d *Decoder) Int64s(name string, v *[]int64) {
	d.repeated(name, v, conv.ParseInt64)
}

// Map 按 field[key] 解码基础类型或枚举映射；同名表单键取第一个值，全部成功后才赋值。
func (d *Decoder) Map[K comparable, V any](
	name string,
	v *map[K]V,
	keyParser func(string) (K, error),
	valueParser func(string) (V, error),
) {
	if d.err != nil {
		return
	}
	values := make(map[K]V)
	prefix := name + "["
	for field := range d.values {
		tail, ok := strings.CutPrefix(field, prefix)
		if !ok {
			continue
		}
		text, ok := strings.CutSuffix(tail, "]")
		if !ok {
			d.fail(name, fmt.Errorf("invalid field name '%s'", field))
			return
		}
		key, err := keyParser(text)
		if err != nil {
			d.fail(name, fmt.Errorf("map key '%s' error: %w", text, err))
			return
		}
		value, err := valueParser(d.values.Get(field))
		if err != nil {
			d.fail(name+"["+text+"]", err)
			return
		}
		values[key] = value
	}
	if len(values) > 0 {
		*v = values
	}
}

// MapStruct 按 field[key].name 解码结构映射；支持 protobuf 的键类型，全部成功后才赋值。
func (d *Decoder) MapStruct[K comparable, T any, P interface {
	*T
	Decodable
}](name string, v *map[K]P, keyParser func(string) (K, error)) {
	if d.err != nil {
		return
	}
	forms := d.mapForms(name)
	if d.err != nil || len(forms) == 0 {
		return
	}
	values := make(map[K]P, len(forms))
	for text, childForm := range forms {
		key, err := keyParser(text)
		if err != nil {
			d.fail(name, fmt.Errorf("map key '%s' error: %w", text, err))
			return
		}
		child := P(new(T))
		if err := child.DecodeForm(childForm); err != nil {
			d.fail(name+"["+text+"]", err)
			return
		}
		values[key] = child
	}
	*v = values
}

// String 解析字符串字段。
func (d *Decoder) String(name string, v *string) {
	d.scalar(name, v, conv.ParseString)
}

// Strings 解析字符串数组。
func (d *Decoder) Strings(name string, v *[]string) {
	d.repeated(name, v, conv.ParseString)
}

// Struct 去掉字段名前缀后解码子结构。
func (d *Decoder) Struct[T any, P interface {
	*T
	Decodable
}](name string, v *P) {
	if d.err != nil {
		return
	}
	values := make(url.Values)
	prefix := name + "."
	for key, ss := range d.values {
		if childName, ok := strings.CutPrefix(key, prefix); ok {
			values[childName] = ss
		}
	}
	if len(values) == 0 {
		return
	}
	value := P(new(T))
	if err := value.DecodeForm(values); err != nil {
		d.fail(name, err)
		return
	}
	*v = value
}

// Structs 按 field[0].name 解码结构数组；索引只允许无符号、无前导零的十进制写法，必须从零开始且连续。
func (d *Decoder) Structs[T any, P interface {
	*T
	Decodable
}](name string, v *[]P) {
	if d.err != nil {
		return
	}
	forms := d.mapForms(name)
	if d.err != nil || len(forms) == 0 {
		return
	}
	values := make([]P, len(forms))
	for text, childForm := range forms {
		index, err := strconv.Atoi(text)
		if err != nil || text[0] < '0' || text[0] > '9' || len(text) > 1 && text[0] == '0' || index >= len(values) {
			d.fail(name, fmt.Errorf("invalid index '%s'", text))
			return
		}
		child := P(new(T))
		if err := child.DecodeForm(childForm); err != nil {
			d.fail(name+"["+text+"]", err)
			return
		}
		values[index] = child
	}
	*v = values
}

// Uint32 解析数值字段。
func (d *Decoder) Uint32(name string, v *uint32) {
	d.scalar(name, v, conv.ParseUint32)
}

// Uint32s 解析数值数组。
func (d *Decoder) Uint32s(name string, v *[]uint32) {
	d.repeated(name, v, conv.ParseUint32)
}

// Uint64 解析数值字段。
func (d *Decoder) Uint64(name string, v *uint64) {
	d.scalar(name, v, conv.ParseUint64)
}

// Uint64s 解析数值数组。
func (d *Decoder) Uint64s(name string, v *[]uint64) {
	d.repeated(name, v, conv.ParseUint64)
}

func (d *Decoder) fail(name string, err error) {
	if d.err != nil || err == nil {
		return
	}
	d.err = fmt.Errorf("decode field '%s' error: %w", name, err)
}

func (d *Decoder) mapForms(name string) map[string]url.Values {
	forms := make(map[string]url.Values)
	prefix := name + "["
	for field, ss := range d.values {
		tail, ok := strings.CutPrefix(field, prefix)
		if !ok {
			continue
		}
		key, childName, ok := strings.Cut(tail, "].")
		if !ok {
			d.fail(name, fmt.Errorf("invalid field name '%s'", tail))
			return nil
		}
		if forms[key] == nil {
			forms[key] = make(url.Values)
		}
		forms[key][childName] = ss
	}
	return forms
}

func (d *Decoder) repeated[T any](name string, v *[]T, parser fieldParser[T]) {
	if d.err != nil {
		return
	}
	ss, ok := d.values[name]
	if !ok {
		return
	}
	values := make([]T, len(ss))
	for i, s := range ss {
		if s == "" {
			continue
		}
		value, err := parser(s)
		if err != nil {
			d.fail(name+"["+strconv.Itoa(i)+"]", err)
			return
		}
		values[i] = value
	}
	*v = values
}

func (d *Decoder) scalar[T any](name string, v *T, parser fieldParser[T]) {
	if d.err != nil {
		return
	}
	s := d.values.Get(name)
	if s == "" {
		return
	}
	value, err := parser(s)
	if err != nil {
		d.fail(name, err)
		return
	}
	*v = value
}
