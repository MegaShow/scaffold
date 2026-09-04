package assert

import (
	"testing"
	"unsafe"
)

// recordingTesting 记录是否输出了错误，用于判断断言失败时是否报告。
type recordingTesting struct {
	errored bool
}

func (r *recordingTesting) Errorf(string, ...any) { r.errored = true }
func (r *recordingTesting) Helper()               {}

func TestElementsMatch(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		a, b []int
		want bool
	}{
		{name: "nil_nil", a: nil, b: nil, want: true},
		{name: "empty_empty", a: []int{}, b: []int{}, want: true},
		{name: "nil_empty", a: nil, b: []int{}, want: true},
		{name: "order", a: []int{1, 2, 3}, b: []int{3, 1, 2}, want: true},
		{name: "duplicates", a: []int{1, 1, 2}, b: []int{2, 1, 1}, want: true},
		{name: "length_mismatch", a: []int{1}, b: []int{1, 2}, want: false},
		{name: "missing_element", a: []int{1, 2}, b: []int{1, 3}, want: false},
		{name: "duplicate_mismatch", a: []int{1, 1, 2}, b: []int{1, 2, 2}, want: false},
	}
	for _, tt := range testCases {
		r := new(recordingTesting)
		got := ElementsMatch(r, tt.a, tt.b)
		if got != tt.want {
			t.Errorf("%s: ElementsMatch = %t, want %t", tt.name, got, tt.want)
		}
		if r.errored != !tt.want {
			t.Errorf("%s: errored = %t, want %t", tt.name, r.errored, !tt.want)
		}
	}

	// 复杂元素类型仍按顺序匹配。
	r := new(recordingTesting)
	got := ElementsMatch(r,
		[]map[string][]int{{"a": {1, 2}}, {"b": {3}}},
		[]map[string][]int{{"b": {3}}, {"a": {1, 2}}})
	if !got || r.errored {
		t.Errorf("complex elements: got=%t errored=%t, want true/false", got, r.errored)
	}
}

// equalCases 是 Equal/NotEqual 共享的对比用例，want 为 Equal 的期望。
var equalCases = []struct {
	name string
	a, b any
	want bool
}{
	{name: "nil", a: nil, b: nil, want: true},
	{name: "int_equal", a: 42, b: 42, want: true},
	{name: "int_not_equal", a: 42, b: 43, want: false},
	{name: "type_mismatch", a: int32(42), b: int64(42), want: false},
	{name: "slice_equal", a: []int{1, 2}, b: []int{1, 2}, want: true},
	{name: "slice_order", a: []int{1, 2}, b: []int{2, 1}, want: false},
	{name: "nil_slice_vs_empty", a: []int(nil), b: []int{}, want: false},
	{name: "nil_map_vs_empty", a: map[string]int(nil), b: map[string]int{}, want: false},
	{name: "map_equal", a: map[string][]int{"a": {1, 2}}, b: map[string][]int{"a": {1, 2}}, want: true},
	{name: "map_not_equal", a: map[string][]int{"a": {1, 2}}, b: map[string][]int{"a": {1, 3}}, want: false},
}

func TestEqual(t *testing.T) {
	t.Parallel()
	for _, tt := range equalCases {
		r := new(recordingTesting)
		got := Equal(r, tt.a, tt.b)
		if got != tt.want {
			t.Errorf("%s: Equal = %t, want %t", tt.name, got, tt.want)
		}
		if r.errored != !tt.want {
			t.Errorf("%s: errored = %t, want %t", tt.name, r.errored, !tt.want)
		}
	}
}

func TestFalse(t *testing.T) {
	t.Parallel()
	r := new(recordingTesting)
	if got := False(r, false); !got {
		t.Errorf("False(false) = %t, want true", got)
	}
	if r.errored {
		t.Error("False(false) should not report error")
	}

	r = new(recordingTesting)
	if got := False(r, true); got {
		t.Errorf("False(true) = %t, want false", got)
	}
	if !r.errored {
		t.Error("False(true) should report error")
	}
}

func TestNil(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		v    any
		want bool
	}{
		{name: "nil", v: nil, want: true},
		{name: "nil_pointer", v: (*int)(nil), want: true},
		{name: "nil_slice", v: []int(nil), want: true},
		{name: "nil_map", v: map[string]int(nil), want: true},
		{name: "nil_chan", v: (chan int)(nil), want: true},
		{name: "nil_func", v: (func())(nil), want: true},
		{name: "nil_unsafe_pointer", v: unsafe.Pointer(nil), want: true}, //nolint:gosec // 测试 isNil 对 unsafe.Pointer 的处理
		{name: "int", v: 0, want: false},
		{name: "string", v: "", want: false},
		{name: "bool", v: false, want: false},
		{name: "pointer", v: new(int), want: false},
		{name: "slice", v: []int{}, want: false},
		{name: "map", v: map[string]int{}, want: false},
		{name: "chan", v: make(chan int), want: false},
		{name: "func", v: func() {}, want: false},
		{name: "struct", v: struct{}{}, want: false},
		{name: "unsafe_pointer", v: unsafe.Pointer(new(int)), want: false}, //nolint:gosec // 测试 isNil 对 unsafe.Pointer 的处理
	}
	for _, tt := range testCases {
		r := new(recordingTesting)
		got := Nil(r, tt.v)
		if got != tt.want {
			t.Errorf("%s: Nil = %t, want %t", tt.name, got, tt.want)
		}
		if r.errored != !tt.want {
			t.Errorf("%s: errored = %t, want %t", tt.name, r.errored, !tt.want)
		}
	}
}

func TestNotNil(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		v    any
		want bool
	}{
		{name: "nil", v: nil, want: false},
		{name: "nil_pointer", v: (*int)(nil), want: false},
		{name: "nil_slice", v: []int(nil), want: false},
		{name: "nil_map", v: map[string]int(nil), want: false},
		{name: "nil_chan", v: (chan int)(nil), want: false},
		{name: "nil_func", v: (func())(nil), want: false},
		{name: "nil_unsafe_pointer", v: unsafe.Pointer(nil), want: false}, //nolint:gosec // 测试 NotNil 对 unsafe.Pointer 的处理
		{name: "pointer", v: new(int), want: true},
		{name: "slice", v: []int{}, want: true},
		{name: "map", v: map[string]int{}, want: true},
		{name: "chan", v: make(chan int), want: true},
		{name: "func", v: func() {}, want: true},
		{name: "unsafe_pointer", v: unsafe.Pointer(new(int)), want: true}, //nolint:gosec // 测试 NotNil 对 unsafe.Pointer 的处理
		{name: "int", v: 0, want: true},
		{name: "string", v: "", want: true},
		{name: "bool", v: false, want: true},
		{name: "struct", v: struct{}{}, want: true},
	}
	for _, tt := range testCases {
		r := new(recordingTesting)
		got := NotNil(r, tt.v)
		if got != tt.want {
			t.Errorf("%s: NotNil = %t, want %t", tt.name, got, tt.want)
		}
		if r.errored != !tt.want {
			t.Errorf("%s: errored = %t, want %t", tt.name, r.errored, !tt.want)
		}
	}
}

func TestNotEqual(t *testing.T) {
	t.Parallel()
	for _, tt := range equalCases {
		r := new(recordingTesting)
		want := !tt.want
		got := NotEqual(r, tt.a, tt.b)
		if got != want {
			t.Errorf("%s: NotEqual = %t, want %t", tt.name, got, want)
		}
		if r.errored != !want {
			t.Errorf("%s: errored = %t, want %t", tt.name, r.errored, !want)
		}
	}
}

func TestNotPanics(t *testing.T) {
	t.Parallel()
	calls := 0
	r := new(recordingTesting)
	got := NotPanics(r, func() { calls++ })
	if calls != 1 {
		t.Errorf("callback should run once, got %d", calls)
	}
	if !got {
		t.Errorf("NotPanics(no panic) = %t, want true", got)
	}
	if r.errored {
		t.Error("NotPanics(no panic) should not report error")
	}

	calls = 0
	r = new(recordingTesting)
	got = NotPanics(r, func() { calls++; panic("boom") })
	if calls != 1 {
		t.Errorf("callback should run once, got %d", calls)
	}
	if got {
		t.Errorf("NotPanics(panic) = %t, want false", got)
	}
	if !r.errored {
		t.Error("NotPanics(panic) should report error")
	}

	calls = 0
	r = new(recordingTesting)
	got = NotPanics(r, func() { calls++; panic(42) })
	if calls != 1 {
		t.Errorf("callback should run once, got %d", calls)
	}
	if got {
		t.Errorf("NotPanics(int panic) = %t, want false", got)
	}
	if !r.errored {
		t.Error("NotPanics(int panic) should report error")
	}

	calls = 0
	r = new(recordingTesting)
	got = NotPanics(r, func() { calls++; panic(nil) })
	if calls != 1 {
		t.Errorf("callback should run once, got %d", calls)
	}
	if got {
		t.Errorf("NotPanics(nil panic) = %t, want false", got)
	}
	if !r.errored {
		t.Error("NotPanics(nil panic) should report error")
	}
}

func TestPanics(t *testing.T) {
	t.Parallel()
	calls := 0
	r := new(recordingTesting)
	got := Panics(r, func() { calls++ })
	if calls != 1 {
		t.Errorf("callback should run once, got %d", calls)
	}
	if got {
		t.Errorf("Panics(no panic) = %t, want false", got)
	}
	if !r.errored {
		t.Error("Panics(no panic) should report error")
	}

	calls = 0
	r = new(recordingTesting)
	got = Panics(r, func() { calls++; panic("boom") })
	if calls != 1 {
		t.Errorf("callback should run once, got %d", calls)
	}
	if !got {
		t.Errorf("Panics(panic) = %t, want true", got)
	}
	if r.errored {
		t.Error("Panics(panic) should not report error")
	}

	calls = 0
	r = new(recordingTesting)
	got = Panics(r, func() { calls++; panic(42) })
	if calls != 1 {
		t.Errorf("callback should run once, got %d", calls)
	}
	if !got {
		t.Errorf("Panics(int panic) = %t, want true", got)
	}
	if r.errored {
		t.Error("Panics(int panic) should not report error")
	}

	calls = 0
	r = new(recordingTesting)
	got = Panics(r, func() { calls++; panic(nil) })
	if calls != 1 {
		t.Errorf("callback should run once, got %d", calls)
	}
	if !got {
		t.Errorf("Panics(nil panic) = %t, want true", got)
	}
	if r.errored {
		t.Error("Panics(nil panic) should not report error")
	}
}

func TestTrue(t *testing.T) {
	t.Parallel()
	r := new(recordingTesting)
	if got := True(r, true); !got {
		t.Errorf("True(true) = %t, want true", got)
	}
	if r.errored {
		t.Error("True(true) should not report error")
	}

	r = new(recordingTesting)
	if got := True(r, false); got {
		t.Errorf("True(false) = %t, want false", got)
	}
	if !r.errored {
		t.Error("True(false) should report error")
	}
}

func TestZero(t *testing.T) {
	t.Parallel()
	type sample struct {
		Count int
		Name  string
	}
	pointer := new(int)
	channel := make(chan int)
	function := func() {}

	// 通过用例：不应报告错误。
	r := new(recordingTesting)
	if got := Zero(r, 0); !got {
		t.Errorf("Zero(0) = %t, want true", got)
	}
	if got := Zero(r, ""); !got {
		t.Errorf("Zero(\"\") = %t, want true", got)
	}
	if got := Zero(r, false); !got {
		t.Errorf("Zero(false) = %t, want true", got)
	}
	if got := Zero(r, (*int)(nil)); !got {
		t.Errorf("Zero((*int)(nil)) = %t, want true", got)
	}
	if got := Zero(r, []int(nil)); !got {
		t.Errorf("Zero([]int(nil)) = %t, want true", got)
	}
	if got := Zero(r, map[string]int(nil)); !got {
		t.Errorf("Zero(map nil) = %t, want true", got)
	}
	if got := Zero(r, (chan int)(nil)); !got {
		t.Errorf("Zero(chan nil) = %t, want true", got)
	}
	if got := Zero(r, (func())(nil)); !got {
		t.Errorf("Zero(func nil) = %t, want true", got)
	}
	if got := Zero(r, sample{}); !got {
		t.Errorf("Zero(sample{}) = %t, want true", got)
	}
	if got := Zero(r, [2]int{}); !got {
		t.Errorf("Zero([2]int{}) = %t, want true", got)
	}
	if got := Zero[any](r, nil); !got {
		t.Errorf("Zero[any](nil) = %t, want true", got)
	}
	if r.errored {
		t.Error("zero-value inputs should not report error")
	}

	// 失败用例：应报告错误。
	r = new(recordingTesting)
	if got := Zero(r, 1); got {
		t.Errorf("Zero(1) = %t, want false", got)
	}
	if got := Zero(r, 1.5); got {
		t.Errorf("Zero(1.5) = %t, want false", got)
	}
	if got := Zero(r, "value"); got {
		t.Errorf("Zero(\"value\") = %t, want false", got)
	}
	if got := Zero(r, true); got {
		t.Errorf("Zero(true) = %t, want false", got)
	}
	if got := Zero(r, pointer); got {
		t.Errorf("Zero(pointer) = %t, want false", got)
	}
	if got := Zero(r, []int{}); got {
		t.Errorf("Zero([]int{}) = %t, want false", got)
	}
	if got := Zero(r, map[string]int{}); got {
		t.Errorf("Zero(map{}) = %t, want false", got)
	}
	if got := Zero(r, channel); got {
		t.Errorf("Zero(channel) = %t, want false", got)
	}
	if got := Zero(r, function); got {
		t.Errorf("Zero(function) = %t, want false", got)
	}
	if got := Zero(r, sample{Count: 1}); got {
		t.Errorf("Zero(sample{Count:1}) = %t, want false", got)
	}
	if got := Zero(r, [2]int{0, 1}); got {
		t.Errorf("Zero([2]int{0,1}) = %t, want false", got)
	}
	if got := Zero[any](r, (*int)(nil)); got {
		t.Errorf("Zero[any]((*int)(nil)) = %t, want false", got)
	}
	if !r.errored {
		t.Error("non-zero inputs should report error")
	}
}
