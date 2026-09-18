package form

import (
	"net/url"
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

func TestNewDecoder(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewDecoder(nil))
	assert.NotNil(t, NewDecoder(url.Values{"field": {"value"}}))
}

func TestDecoder_Bool(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   bool
		err    bool
	}{
		{name: "valid_true", values: url.Values{"field": {"true", "ignored"}}, want: true},
		{name: "valid_false", values: url.Values{"field": {"false", "ignored"}}, want: false},
		{name: "missing", values: nil, want: false},
		{name: "nil_values", values: url.Values{"field": nil}, want: false},
		{name: "empty_values", values: url.Values{"field": {}}, want: false},
		{name: "single_true", values: url.Values{"field": {"true"}}, want: true},
		{name: "single_false", values: url.Values{"field": {"false"}}, want: false},
		{name: "numeric_one", values: url.Values{"field": {"1"}}, want: true},
		{name: "numeric_zero", values: url.Values{"field": {"0"}}, want: false},
		{name: "empty_string", values: url.Values{"field": {""}}, want: false},
		{name: "invalid_number", values: url.Values{"field": {"123"}}, want: false, err: true},
		{name: "invalid", values: url.Values{"field": {"invalid"}}, want: false, err: true},
	}
	for _, tt := range testCases {
		var got bool
		d := NewDecoder(tt.values)
		d.Bool("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Bools(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   []bool
		err    bool
	}{
		{name: "valid", values: url.Values{"field": {"", "true", "", "true"}}, want: []bool{false, true, false, true}},
		{name: "all_false", values: url.Values{"field": {"", "false", "", "false"}}, want: []bool{false, false, false, false}},
		{name: "missing", values: nil, want: nil},
		{name: "nil_values", values: url.Values{"field": nil}, want: []bool{}},
		{name: "empty_values", values: url.Values{"field": {}}, want: []bool{}},
		{name: "invalid", values: url.Values{"field": {"true", "", "invalid"}}, want: nil, err: true},
	}
	for _, tt := range testCases {
		var got []bool
		d := NewDecoder(tt.values)
		d.Bools("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Bytes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   []byte
	}{
		{name: "valid", values: url.Values{"field": {"你好", "ignored"}}, want: []byte("你好")},
		{name: "missing", values: nil, want: nil},
		{name: "nil_values", values: url.Values{"field": nil}, want: nil},
		{name: "empty_values", values: url.Values{"field": {}}, want: nil},
		{name: "first_empty", values: url.Values{"field": {"", "你好"}}, want: nil},
	}
	for _, tt := range testCases {
		var got []byte
		d := NewDecoder(tt.values)
		d.Bytes("field", &got)
		assert.Nil(t, d.Err(), tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_BytesList(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   [][]byte
	}{
		{name: "valid", values: url.Values{"field": {"", "你好", "", "你好"}}, want: [][]byte{nil, []byte("你好"), nil, []byte("你好")}},
		{name: "missing", values: nil, want: nil},
		{name: "nil_values", values: url.Values{"field": nil}, want: [][]byte{}},
		{name: "empty_values", values: url.Values{"field": {}}, want: [][]byte{}},
	}
	for _, tt := range testCases {
		var got [][]byte
		d := NewDecoder(tt.values)
		d.BytesList("field", &got)
		assert.Nil(t, d.Err(), tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Enum(t *testing.T) {
	t.Parallel()
	type status int32
	testCases := []struct {
		name   string
		values url.Values
		want   status
		err    bool
	}{
		{name: "valid", values: url.Values{"field": {"42", "ignored"}}, want: 42},
		{name: "missing", values: nil, want: 0},
		{name: "nil_values", values: url.Values{"field": nil}, want: 0},
		{name: "empty_values", values: url.Values{"field": {}}, want: 0},
		{name: "first_empty", values: url.Values{"field": {"", "42"}}, want: 0},
		{name: "overflow", values: url.Values{"field": {"2147483648"}}, want: 0, err: true},
	}
	for _, tt := range testCases {
		var got status
		d := NewDecoder(tt.values)
		d.Enum("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Enums(t *testing.T) {
	t.Parallel()
	type status int32
	testCases := []struct {
		name   string
		values url.Values
		want   []status
		err    bool
	}{
		{name: "valid", values: url.Values{"field": {"", "42", "", "42"}}, want: []status{0, 42, 0, 42}},
		{name: "missing", values: nil, want: nil},
		{name: "nil_values", values: url.Values{"field": nil}, want: []status{}},
		{name: "empty_values", values: url.Values{"field": {}}, want: []status{}},
		{name: "overflow", values: url.Values{"field": {"42", "", "2147483648"}}, want: nil, err: true},
	}
	for _, tt := range testCases {
		var got []status
		d := NewDecoder(tt.values)
		d.Enums("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Err(t *testing.T) {
	t.Parallel()
	d, v, f := NewDecoder(url.Values{"field": {"123"}, "field2": {"abc"}}), int32(0), float32(0)
	assert.Nil(t, d.Err())
	d.Int32("field", &v)
	assert.Nil(t, d.Err())
	d.Int32("field2", &v)
	assert.NotNil(t, d.Err())
	err := d.Err()
	d.Float32("field2", &f)
	assert.Equal(t, err, d.Err())
}

func TestDecoder_Float32(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   float32
		err    bool
	}{
		{name: "valid", values: url.Values{"field": {"-1.25", "ignored"}}, want: -1.25},
		{name: "missing", values: nil, want: 0},
		{name: "nil_values", values: url.Values{"field": nil}, want: 0},
		{name: "empty_values", values: url.Values{"field": {}}, want: 0},
		{name: "first_empty", values: url.Values{"field": {"", "-1.25"}}, want: 0},
		{name: "overflow", values: url.Values{"field": {"1e39"}}, want: 0, err: true},
	}
	for _, tt := range testCases {
		var got float32
		d := NewDecoder(tt.values)
		d.Float32("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Float32s(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   []float32
		err    bool
	}{
		{name: "valid", values: url.Values{"field": {"", "-1.25", "", "-1.25"}}, want: []float32{0, -1.25, 0, -1.25}},
		{name: "missing", values: nil, want: nil},
		{name: "nil_values", values: url.Values{"field": nil}, want: []float32{}},
		{name: "empty_values", values: url.Values{"field": {}}, want: []float32{}},
		{name: "overflow", values: url.Values{"field": {"-1.25", "", "1e39"}}, want: nil, err: true},
	}
	for _, tt := range testCases {
		var got []float32
		d := NewDecoder(tt.values)
		d.Float32s("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Float64(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   float64
		err    bool
	}{
		{name: "valid_decimal", values: url.Values{"field": {"1.25"}}, want: 1.25},
		{name: "valid_exponent", values: url.Values{"field": {"1e100", "ignored"}}, want: 1e100},
		{name: "missing", values: nil, want: 0},
		{name: "nil_values", values: url.Values{"field": nil}, want: 0},
		{name: "empty_values", values: url.Values{"field": {}}, want: 0},
		{name: "first_empty", values: url.Values{"field": {"", "1e100"}}, want: 0},
		{name: "overflow", values: url.Values{"field": {"1e309"}}, want: 0, err: true},
	}
	for _, tt := range testCases {
		var got float64
		d := NewDecoder(tt.values)
		d.Float64("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Float64s(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   []float64
		err    bool
	}{
		{name: "single", values: url.Values{"field": {"1.25"}}, want: []float64{1.25}},
		{name: "mixed", values: url.Values{"field": {"", "1.25", "", "1e100"}}, want: []float64{0, 1.25, 0, 1e100}},
		{name: "missing", values: nil, want: nil},
		{name: "nil_values", values: url.Values{"field": nil}, want: []float64{}},
		{name: "empty_values", values: url.Values{"field": {}}, want: []float64{}},
		{name: "overflow", values: url.Values{"field": {"1e100", "", "1e309"}}, want: nil, err: true},
	}
	for _, tt := range testCases {
		var got []float64
		d := NewDecoder(tt.values)
		d.Float64s("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Int32(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   int32
		err    bool
	}{
		{name: "valid", values: url.Values{"field": {"12"}}, want: 12},
		{name: "min_value", values: url.Values{"field": {"-2147483648", "ignored"}}, want: -2147483648},
		{name: "missing", values: nil, want: 0},
		{name: "nil_values", values: url.Values{"field": nil}, want: 0},
		{name: "empty_values", values: url.Values{"field": {}}, want: 0},
		{name: "first_empty", values: url.Values{"field": {"", "-2147483648"}}, want: 0},
		{name: "overflow", values: url.Values{"field": {"2147483648"}}, want: 0, err: true},
	}
	for _, tt := range testCases {
		var got int32
		d := NewDecoder(tt.values)
		d.Int32("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Int32s(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   []int32
		err    bool
	}{
		{name: "valid", values: url.Values{"field": {"", "12", "", "-2147483648"}}, want: []int32{0, 12, 0, -2147483648}},
		{name: "missing", values: nil, want: nil},
		{name: "nil_values", values: url.Values{"field": nil}, want: []int32{}},
		{name: "empty_values", values: url.Values{"field": {}}, want: []int32{}},
		{name: "overflow", values: url.Values{"field": {"-2147483648", "", "2147483648"}}, want: nil, err: true},
	}
	for _, tt := range testCases {
		var got []int32
		d := NewDecoder(tt.values)
		d.Int32s("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Int64(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   int64
		err    bool
	}{
		{name: "valid", values: url.Values{"field": {"123", "ignored"}}, want: 123},
		{name: "min_value", values: url.Values{"field": {"-9223372036854775808", "ignored"}}, want: -9223372036854775808},
		{name: "missing", values: nil, want: 0},
		{name: "nil_values", values: url.Values{"field": nil}, want: 0},
		{name: "empty_values", values: url.Values{"field": {}}, want: 0},
		{name: "first_empty", values: url.Values{"field": {"", "-9223372036854775808"}}, want: 0},
		{name: "overflow", values: url.Values{"field": {"9223372036854775808"}}, want: 0, err: true},
	}
	for _, tt := range testCases {
		var got int64
		d := NewDecoder(tt.values)
		d.Int64("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Int64s(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   []int64
		err    bool
	}{
		{name: "valid", values: url.Values{"field": {"", "123", "", "-9223372036854775808"}}, want: []int64{0, 123, 0, -9223372036854775808}},
		{name: "missing", values: nil, want: nil},
		{name: "nil_values", values: url.Values{"field": nil}, want: []int64{}},
		{name: "empty_values", values: url.Values{"field": {}}, want: []int64{}},
		{name: "overflow", values: url.Values{"field": {"-9223372036854775808", "", "9223372036854775808"}}, want: nil, err: true},
	}
	for _, tt := range testCases {
		var got []int64
		d := NewDecoder(tt.values)
		d.Int64s("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Map(t *testing.T) {
	t.Parallel()
	// map[int32]string
	testCases := []struct {
		name   string
		values url.Values
		want   map[int32]string
		err    bool
	}{
		{name: "missing", values: nil, want: nil},
		{name: "unrelated", values: url.Values{"items": {"ignored"}, "other[1]": {"ignored"}}, want: nil},
		{name: "multiple_keys", values: url.Values{"items[-1]": {"你好", "ignored"}, "items[2]": {""}, "other[3]": {"ignored"}}, want: map[int32]string{-1: "你好", 2: ""}},
		{name: "nil_value", values: url.Values{"items[0]": nil}, want: map[int32]string{0: ""}},
		{name: "invalid_bracket", values: url.Values{"items[1": {"1"}, "items[2]": {"2"}}, want: nil, err: true},
		{name: "invalid_key", values: url.Values{"items[bad]": {"1"}, "items[2]": {"2"}}, want: nil, err: true},
	}
	for _, tt := range testCases {
		var got map[int32]string
		d := NewDecoder(tt.values)
		d.Map("items", &got, conv.ParseInt32, conv.ParseString)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}

	// map[int32]int32
	var intValues map[int32]int32
	d := NewDecoder(url.Values{"items[1]": {"bad"}, "items[2]": {"2"}})
	d.Map("items", &intValues, conv.ParseInt32, conv.ParseInt32)
	assert.NotNil(t, d.Err())
	assert.Equal(t, nil, intValues)

	// map[bool]enum
	type status int32
	var enumValues map[bool]status
	d = NewDecoder(url.Values{"items[true]": {"-1"}, "items[0]": {"42"}})
	d.Map("items", &enumValues, conv.ParseBool, conv.ParseEnum[status])
	assert.Nil(t, d.Err())
	assert.Equal(t, map[bool]status{true: -1, false: 42}, enumValues)
}

func TestDecoder_MapStruct(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   map[int32]*testChild
		err    bool
	}{
		{name: "missing", values: nil, want: nil},
		{name: "unrelated", values: url.Values{"child": {"ignored"}, "children": {"ignored"}, "items": {"ignored"}, "other.name": {"ignored"}}, want: nil},
		{name: "multiple_entries", values: url.Values{"items[-1].name": {"Alice", "ignored"}, "items[-1].age": {"12"}, "items[-1].tags": {"one", "", "three"}, "items[2].name": {"Bob"}, "other[3].name": {"ignored"}}, want: map[int32]*testChild{-1: {Name: "Alice", Age: 12, Tags: []string{"one", "", "three"}}, 2: {Name: "Bob"}}},
		{name: "invalid_entry", values: url.Values{"items[1]": {""}, "items[2].name": {"valid"}}, want: nil, err: true},
		{name: "invalid_key", values: url.Values{"items[bad].name": {"Alice"}, "items[2].name": {"valid"}}, want: nil, err: true},
		{name: "invalid_value", values: url.Values{"items[1].age": {"bad"}, "items[2].name": {"valid"}}, want: nil, err: true},
	}
	for _, tt := range testCases {
		var got map[int32]*testChild
		d := NewDecoder(tt.values)
		d.MapStruct("items", &got, conv.ParseInt32)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_String(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   string
	}{
		{name: "valid", values: url.Values{"field": {"你好", "ignored"}}, want: "你好"},
		{name: "missing", values: nil, want: ""},
		{name: "nil_values", values: url.Values{"field": nil}, want: ""},
		{name: "empty_values", values: url.Values{"field": {}}, want: ""},
		{name: "first_empty", values: url.Values{"field": {"", "你好"}}, want: ""},
	}
	for _, tt := range testCases {
		var got string
		d := NewDecoder(tt.values)
		d.String("field", &got)
		assert.Nil(t, d.Err(), tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Strings(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   []string
	}{
		{name: "valid", values: url.Values{"field": {"", "你好", "", "你好"}}, want: []string{"", "你好", "", "你好"}},
		{name: "missing", values: nil, want: nil},
		{name: "nil_values", values: url.Values{"field": nil}, want: []string{}},
		{name: "empty_values", values: url.Values{"field": {}}, want: []string{}},
	}
	for _, tt := range testCases {
		var got []string
		d := NewDecoder(tt.values)
		d.Strings("field", &got)
		assert.Nil(t, d.Err(), tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Struct(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   *testChild
		err    bool
	}{
		{name: "missing", values: nil, want: &testChild{}},
		{name: "empty", values: url.Values{}, want: &testChild{}},
		{name: "unrelated", values: url.Values{"child": {"ignored"}, "children": {"ignored"}, "items": {"ignored"}, "other.name": {"ignored"}}, want: &testChild{}},
		{name: "valid", values: url.Values{"child.name": {"Alice", "ignored"}, "child.age": {"12"}, "child.tags": {"one", "", "three"}, "children.name": {"ignored"}, "name": {"ignored"}}, want: &testChild{Name: "Alice", Age: 12, Tags: []string{"one", "", "three"}}},
		{name: "invalid_value", values: url.Values{"child.name": {"changed"}, "child.age": {"bad"}}, want: &testChild{}, err: true},
	}
	for _, tt := range testCases {
		got := &testChild{}
		d := NewDecoder(tt.values)
		d.Struct("child", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Structs(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   []*testChild
		err    bool
	}{
		{name: "missing", values: nil, want: nil},
		{name: "empty", values: url.Values{}, want: nil},
		{name: "unrelated", values: url.Values{"child": {"ignored"}, "children": {"ignored"}, "items": {"ignored"}, "other.name": {"ignored"}}, want: nil},
		{name: "valid", values: url.Values{"children[1].name": {"Bob"}, "children[0].name": {"Alice", "ignored"}, "children[0].age": {"12"}, "children[0].tags": {"one", "", "three"}, "other[2].name": {"ignored"}}, want: []*testChild{{Name: "Alice", Age: 12, Tags: []string{"one", "", "three"}}, {Name: "Bob"}}},
		{name: "missing_separator", values: url.Values{"children[0]": {"bad"}}, want: nil, err: true},
		{name: "empty_index", values: url.Values{"children[].name": {"bad"}}, want: nil, err: true},
		{name: "negative_index", values: url.Values{"children[-1].name": {"bad"}}, want: nil, err: true},
		{name: "signed_index", values: url.Values{"children[+0].name": {"bad"}}, want: nil, err: true},
		{name: "leading_zero", values: url.Values{"children[00].name": {"bad"}}, want: nil, err: true},
		{name: "non_decimal_index", values: url.Values{"children[0x0].name": {"bad"}}, want: nil, err: true},
		{name: "overflow_index", values: url.Values{"children[99999999999999999999].name": {"bad"}}, want: nil, err: true},
		{name: "missing_zero", values: url.Values{"children[1].name": {"bad"}}, want: nil, err: true},
		{name: "gap", values: url.Values{"children[0].name": {"valid"}, "children[2].name": {"bad"}}, want: nil, err: true},
		{name: "invalid_child", values: url.Values{"children[0].name": {"valid"}, "children[1].age": {"bad"}}, want: nil, err: true},
	}
	for _, tt := range testCases {
		var got []*testChild
		d := NewDecoder(tt.values)
		d.Structs("children", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Uint32(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   uint32
		err    bool
	}{
		{name: "valid", values: url.Values{"field": {"123", "ignored"}}, want: 123},
		{name: "max_value", values: url.Values{"field": {"4294967295", "ignored"}}, want: 4294967295},
		{name: "missing", values: nil, want: 0},
		{name: "nil_values", values: url.Values{"field": nil}, want: 0},
		{name: "empty_values", values: url.Values{"field": {}}, want: 0},
		{name: "first_empty", values: url.Values{"field": {"", "4294967295"}}, want: 0},
		{name: "negative", values: url.Values{"field": {"-1"}}, want: 0, err: true},
	}
	for _, tt := range testCases {
		var got uint32
		d := NewDecoder(tt.values)
		d.Uint32("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Uint32s(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   []uint32
		err    bool
	}{
		{name: "valid", values: url.Values{"field": {"", "123", "", "4294967295"}}, want: []uint32{0, 123, 0, 4294967295}},
		{name: "missing", values: nil, want: nil},
		{name: "nil_values", values: url.Values{"field": nil}, want: []uint32{}},
		{name: "empty_values", values: url.Values{"field": {}}, want: []uint32{}},
		{name: "negative", values: url.Values{"field": {"4294967295", "", "-1"}}, want: nil, err: true},
	}
	for _, tt := range testCases {
		var got []uint32
		d := NewDecoder(tt.values)
		d.Uint32s("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Uint64(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   uint64
		err    bool
	}{
		{name: "valid", values: url.Values{"field": {"123", "ignored"}}, want: 123},
		{name: "max_value", values: url.Values{"field": {"18446744073709551615", "ignored"}}, want: 18446744073709551615},
		{name: "missing", values: nil, want: 0},
		{name: "nil_values", values: url.Values{"field": nil}, want: 0},
		{name: "empty_values", values: url.Values{"field": {}}, want: 0},
		{name: "first_empty", values: url.Values{"field": {"", "18446744073709551615"}}, want: 0},
		{name: "overflow", values: url.Values{"field": {"18446744073709551616"}}, want: 0, err: true},
	}
	for _, tt := range testCases {
		var got uint64
		d := NewDecoder(tt.values)
		d.Uint64("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func TestDecoder_Uint64s(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values url.Values
		want   []uint64
		err    bool
	}{
		{name: "valid", values: url.Values{"field": {"", "123", "", "18446744073709551615"}}, want: []uint64{0, 123, 0, 18446744073709551615}},
		{name: "missing", values: nil, want: nil},
		{name: "nil_values", values: url.Values{"field": nil}, want: []uint64{}},
		{name: "empty_values", values: url.Values{"field": {}}, want: []uint64{}},
		{name: "overflow", values: url.Values{"field": {"18446744073709551615", "", "18446744073709551616"}}, want: nil, err: true},
	}
	for _, tt := range testCases {
		var got []uint64
		d := NewDecoder(tt.values)
		d.Uint64s("field", &got)
		assert.Equal(t, tt.err, d.Err() != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}
