package form

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"go.icytown.com/scaffold/mux/conv"
	"go.icytown.com/scaffold/mux/internal/assert"
)

type testChild struct {
	Name string
	Age  int32
	Tags []string
}

func (c *testChild) DecodeForm(values url.Values) error {
	d := NewDecoder(values)
	d.String("name", &c.Name)
	d.Int32("age", &c.Age)
	d.Strings("tags", &c.Tags)
	return d.Err()
}

func TestDecoderMap(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		values url.Values
		want   map[int32]string
	}{
		{name: "nil"},
		{name: "unrelated", values: url.Values{"items": {"ignored"}, "other[1]": {"ignored"}}},
		{
			name:   "multiple_keys_and_first_value",
			values: url.Values{"items[-1]": {"你好", "ignored"}, "items[2]": {""}, "other[3]": {"ignored"}},
			want:   map[int32]string{-1: "你好", 2: ""},
		},
		{
			name:   "key_without_values",
			values: url.Values{"items[0]": nil},
			want:   map[int32]string{0: ""},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			original := map[int32]string{99: "original"}
			got := original
			d := NewDecoder(tt.values)
			d.Map("items", &got, conv.ParseInt32, conv.ParseString)
			assert.Nil(t, d.Err())
			if tt.want == nil {
				assert.Equal(t, original, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
			assert.Equal(t, map[int32]string{99: "original"}, original)
		})
	}

	t.Run("bool_keys_and_enum_values", func(t *testing.T) {
		t.Parallel()
		type status int32
		d := NewDecoder(url.Values{"items[true]": {"-1"}, "items[0]": {"42"}})
		var got map[bool]status
		d.Map("items", &got, conv.ParseBool, conv.ParseEnum[status])
		assert.Nil(t, d.Err())
		assert.Equal(t, map[bool]status{true: -1, false: 42}, got)
	})
}

func TestDecoderMapErrors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		field string
		value string
		path  string
	}{
		{name: "missing_bracket", field: "items[1", value: "1", path: "decode field 'items' error: invalid field name 'items[1'"},
		{name: "invalid_key", field: "items[bad]", value: "1", path: "decode field 'items' error: map key 'bad' error:"},
		{name: "invalid_value", field: "items[1]", value: "bad", path: "decode field 'items[1]' error:"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			original := map[int32]int32{99: 99}
			got := original
			d := NewDecoder(url.Values{tt.field: {tt.value}, "items[2]": {"2"}})
			d.Map("items", &got, conv.ParseInt32, conv.ParseInt32)
			assertDecodeError(t, d.Err(), tt.path)
			assert.Equal(t, map[int32]int32{99: 99}, got)
			assert.Equal(t, map[int32]int32{99: 99}, original)
		})
	}
}

func TestDecoderMapStruct(t *testing.T) {
	t.Parallel()
	d := NewDecoder(url.Values{
		"items[-1].name": {"Alice", "ignored"},
		"items[-1].age":  {"12"},
		"items[-1].tags": {"one", "", "three"},
		"items[2].name":  {"Bob"},
		"other[3].name":  {"ignored"},
	})
	original := map[int32]*testChild{99: {Name: "original"}}
	got := original
	d.MapStruct("items", &got, conv.ParseInt32)
	assert.Nil(t, d.Err())
	assert.Equal(t, map[int32]*testChild{
		-1: {Name: "Alice", Age: 12, Tags: []string{"one", "", "three"}},
		2:  {Name: "Bob"},
	}, got)
	assert.Equal(t, map[int32]*testChild{99: {Name: "original"}}, original)
}

func TestDecoderMapStructErrors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		field string
		value string
		path  string
	}{
		{name: "missing_separator", field: "items[1]", path: "decode field 'items' error: invalid field name"},
		{name: "invalid_key", field: "items[bad].name", value: "Alice", path: "decode field 'items' error: map key 'bad' error:"},
		{name: "invalid_child", field: "items[1].age", value: "bad", path: "decode field 'items[1]' error: decode field 'age' error:"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			original := map[int32]*testChild{99: {Name: "original"}}
			got := original
			d := NewDecoder(url.Values{tt.field: {tt.value}, "items[2].name": {"valid"}})
			d.MapStruct("items", &got, conv.ParseInt32)
			assertDecodeError(t, d.Err(), tt.path)
			assert.Equal(t, map[int32]*testChild{99: {Name: "original"}}, got)
			assert.Equal(t, map[int32]*testChild{99: {Name: "original"}}, original)
		})
	}
}

func TestDecoderMissingNestedFields(t *testing.T) {
	t.Parallel()
	for _, values := range []url.Values{nil, {}, {"child": {"ignored"}, "children": {"ignored"}, "items": {"ignored"}, "other.name": {"ignored"}}} {
		d := NewDecoder(values)
		original := &testChild{Name: "original"}
		child := original
		children := []*testChild{original}
		items := map[string]*testChild{"original": original}
		d.Struct("child", &child)
		d.Structs("children", &children)
		d.MapStruct("items", &items, conv.ParseString)
		assert.Nil(t, d.Err())
		assert.True(t, child == original)
		assert.Equal(t, []*testChild{original}, children)
		assert.Equal(t, map[string]*testChild{"original": original}, items)

		var absentChild *testChild
		var absentChildren []*testChild
		var absentItems map[string]*testChild
		d.Struct("child", &absentChild)
		d.Structs("children", &absentChildren)
		d.MapStruct("items", &absentItems, conv.ParseString)
		assert.Nil(t, d.Err())
		assert.Nil(t, absentChild)
		assert.Nil(t, absentChildren)
		assert.Nil(t, absentItems)
	}
}

func TestDecoderScalarAndRepeated(t *testing.T) {
	t.Parallel()
	type status int32
	testScalarAndRepeated(t, "bool", (*Decoder).Bool, (*Decoder).Bools, "true", true, false, "invalid")
	testScalarAndRepeated(t, "bool_false", (*Decoder).Bool, (*Decoder).Bools, "false", false, true, "invalid")
	testScalarAndRepeated(t, "bytes", (*Decoder).Bytes, (*Decoder).BytesList, "你好", []byte("你好"), []byte("original"), "")
	testScalarAndRepeated(t, "enum", (*Decoder).Enum[status], (*Decoder).Enums[status], "42", status(42), status(99), "2147483648")
	testScalarAndRepeated(t, "float32", (*Decoder).Float32, (*Decoder).Float32s, "-1.25", float32(-1.25), float32(99), "1e39")
	testScalarAndRepeated(t, "float64", (*Decoder).Float64, (*Decoder).Float64s, "1e100", 1e100, 99.0, "1e309")
	testScalarAndRepeated(t, "int32", (*Decoder).Int32, (*Decoder).Int32s, "-2147483648", int32(-2147483648), int32(99), "2147483648")
	testScalarAndRepeated(t, "int64", (*Decoder).Int64, (*Decoder).Int64s, "-9223372036854775808", int64(-9223372036854775808), int64(99), "9223372036854775808")
	testScalarAndRepeated(t, "string", (*Decoder).String, (*Decoder).Strings, "你好", "你好", "original", "")
	testScalarAndRepeated(t, "uint32", (*Decoder).Uint32, (*Decoder).Uint32s, "4294967295", uint32(4294967295), uint32(99), "-1")
	testScalarAndRepeated(t, "uint64", (*Decoder).Uint64, (*Decoder).Uint64s, "18446744073709551615", uint64(18446744073709551615), uint64(99), "18446744073709551616")
}

func TestDecoderStopsAfterFirstError(t *testing.T) {
	t.Parallel()
	d := NewDecoder(url.Values{
		"bad": {"invalid"}, "scalar": {"changed"}, "repeated": {"changed"},
		"items[1]": {"changed"}, "child.name": {"changed"},
		"children[0].name": {"changed"}, "objects[1].name": {"changed"},
	})
	var bad int32
	d.Int32("bad", &bad)
	firstErr := d.Err()
	assertDecodeError(t, firstErr, "decode field 'bad' error:")
	assert.True(t, errors.Is(firstErr, strconv.ErrSyntax))
	var numberError *strconv.NumError
	assert.True(t, errors.As(firstErr, &numberError))

	scalar := "original"
	repeated := []string{"original"}
	items := map[int32]string{99: "original"}
	original := &testChild{Name: "original"}
	child := original
	children := []*testChild{original}
	objects := map[int32]*testChild{99: original}
	d.String("scalar", &scalar)
	d.Strings("repeated", &repeated)
	d.Map("items", &items, conv.ParseInt32, conv.ParseString)
	d.Struct("child", &child)
	d.Structs("children", &children)
	d.MapStruct("objects", &objects, conv.ParseInt32)
	d.Bool("bad", new(bool))
	assert.True(t, errors.Is(d.Err(), firstErr))
	assert.Equal(t, "original", scalar)
	assert.Equal(t, []string{"original"}, repeated)
	assert.Equal(t, map[int32]string{99: "original"}, items)
	assert.True(t, child == original)
	assert.Equal(t, []*testChild{original}, children)
	assert.Equal(t, map[int32]*testChild{99: original}, objects)
	assert.Equal(t, &testChild{Name: "original"}, original)
}

func TestDecoderStruct(t *testing.T) {
	t.Parallel()
	original := &testChild{Name: "original", Age: 99}
	got := original
	d := NewDecoder(url.Values{
		"child.name": {"Alice", "ignored"}, "child.age": {"12"},
		"child.tags": {"one", "", "three"}, "children.name": {"ignored"}, "name": {"ignored"},
	})
	d.Struct("child", &got)
	assert.Nil(t, d.Err())
	assert.Equal(t, &testChild{Name: "Alice", Age: 12, Tags: []string{"one", "", "three"}}, got)
	assert.False(t, got == original)
	assert.Equal(t, &testChild{Name: "original", Age: 99}, original)
}

func TestDecoderStructError(t *testing.T) {
	t.Parallel()
	original := &testChild{Name: "original", Age: 99}
	got := original
	d := NewDecoder(url.Values{"child.name": {"changed"}, "child.age": {"bad"}})
	d.Struct("child", &got)
	assertDecodeError(t, d.Err(), "decode field 'child' error: decode field 'age' error:")
	assert.True(t, errors.Is(d.Err(), strconv.ErrSyntax))
	assert.True(t, got == original)
	assert.Equal(t, &testChild{Name: "original", Age: 99}, original)
}

func TestDecoderStructs(t *testing.T) {
	t.Parallel()
	d := NewDecoder(url.Values{
		"children[1].name": {"Bob"}, "children[0].name": {"Alice", "ignored"},
		"children[0].age": {"12"}, "children[0].tags": {"one", "", "three"},
		"other[2].name": {"ignored"},
	})
	original := []*testChild{{Name: "original"}}
	got := original
	d.Structs("children", &got)
	assert.Nil(t, d.Err())
	assert.Equal(t, []*testChild{
		{Name: "Alice", Age: 12, Tags: []string{"one", "", "three"}},
		{Name: "Bob"},
	}, got)
	assert.Equal(t, []*testChild{{Name: "original"}}, original)
}

func TestDecoderStructsErrors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		values url.Values
		path   string
	}{
		{name: "missing_separator", values: url.Values{"children[0]": {"bad"}}, path: "decode field 'children' error: invalid field name"},
		{name: "empty_index", values: url.Values{"children[].name": {"bad"}}, path: "decode field 'children' error: invalid index ''"},
		{name: "negative_index", values: url.Values{"children[-1].name": {"bad"}}, path: "decode field 'children' error: invalid index '-1'"},
		{name: "signed_index", values: url.Values{"children[+0].name": {"bad"}}, path: "decode field 'children' error: invalid index '+0'"},
		{name: "leading_zero", values: url.Values{"children[00].name": {"bad"}}, path: "decode field 'children' error: invalid index '00'"},
		{name: "non_decimal_index", values: url.Values{"children[0x0].name": {"bad"}}, path: "decode field 'children' error: invalid index '0x0'"},
		{name: "overflow_index", values: url.Values{"children[99999999999999999999].name": {"bad"}}, path: "decode field 'children' error: invalid index"},
		{name: "missing_zero", values: url.Values{"children[1].name": {"bad"}}, path: "decode field 'children' error: invalid index '1'"},
		{name: "gap", values: url.Values{"children[0].name": {"valid"}, "children[2].name": {"bad"}}, path: "decode field 'children' error: invalid index '2'"},
		{name: "invalid_child", values: url.Values{"children[0].name": {"valid"}, "children[1].age": {"bad"}}, path: "decode field 'children[1]' error: decode field 'age' error:"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			original := []*testChild{{Name: "original"}}
			got := original
			d := NewDecoder(tt.values)
			d.Structs("children", &got)
			assertDecodeError(t, d.Err(), tt.path)
			assert.Equal(t, []*testChild{{Name: "original"}}, got)
			assert.Equal(t, []*testChild{{Name: "original"}}, original)
		})
	}
}

func assertDecodeError(t *testing.T, err error, prefix string) {
	t.Helper()
	if assert.NotNil(t, err) {
		if !strings.HasPrefix(err.Error(), prefix) {
			t.Errorf("Expected error message to start with %q, got %q", prefix, err.Error())
		}
	}
}

func testScalarAndRepeated[T any](
	t *testing.T,
	name string,
	scalar func(*Decoder, string, *T),
	repeated func(*Decoder, string, *[]T),
	text string,
	want, original T,
	invalid string,
) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Parallel()
		t.Run("scalar", func(t *testing.T) {
			t.Parallel()
			got := original
			d := NewDecoder(url.Values{"field": {text, "ignored"}})
			scalar(d, "field", &got)
			assert.Nil(t, d.Err())
			assert.Equal(t, want, got)
		})
		t.Run("scalar_empty_or_missing", func(t *testing.T) {
			t.Parallel()
			for _, values := range []url.Values{nil, {}, {"field": nil}, {"field": {}}, {"field": {"", text}}} {
				got := original
				d := NewDecoder(values)
				scalar(d, "field", &got)
				assert.Nil(t, d.Err())
				assert.Equal(t, original, got)
			}
		})
		t.Run("repeated", func(t *testing.T) {
			t.Parallel()
			backing := []T{original, original, original, original}
			got := backing
			d := NewDecoder(url.Values{"field": {"", text, "", text}})
			repeated(d, "field", &got)
			assert.Nil(t, d.Err())
			var zero T
			assert.Equal(t, []T{zero, want, zero, want}, got)
			assert.Equal(t, []T{original, original, original, original}, backing)
		})
		t.Run("repeated_empty_or_missing", func(t *testing.T) {
			t.Parallel()
			for _, values := range []url.Values{nil, {}, {"field": nil}, {"field": {}}} {
				got := []T{original}
				d := NewDecoder(values)
				repeated(d, "field", &got)
				assert.Nil(t, d.Err())
				if values.Has("field") {
					assert.Equal(t, []T{}, got)
				} else {
					assert.Equal(t, []T{original}, got)
				}
			}
		})
		if invalid == "" {
			return
		}
		t.Run("scalar_error", func(t *testing.T) {
			t.Parallel()
			got := original
			d := NewDecoder(url.Values{"field": {invalid}})
			scalar(d, "field", &got)
			assertDecodeError(t, d.Err(), "decode field 'field' error:")
			assert.Equal(t, original, got)
		})
		t.Run("repeated_error", func(t *testing.T) {
			t.Parallel()
			backing := []T{original, original, original}
			got := backing
			d := NewDecoder(url.Values{"field": {text, "", invalid}})
			repeated(d, "field", &got)
			assertDecodeError(t, d.Err(), "decode field 'field[2]' error:")
			assert.Equal(t, []T{original, original, original}, got)
			assert.Equal(t, []T{original, original, original}, backing)
		})
	})
}
