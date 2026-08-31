package assert

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unsafe"
)

// recordingTesting 记录断言的报告，避免预期的断言失败使当前测试失败。
type recordingTesting struct {
	messages      []string
	helperCalls   int
	missingHelper bool
}

func (r *recordingTesting) Errorf(format string, args ...any) {
	if r.helperCalls == 0 {
		r.missingHelper = true
	}
	r.messages = append(r.messages, fmt.Sprintf(format, args...))
}

func (r *recordingTesting) Helper() {
	r.helperCalls++
}

func TestElementsMatch(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		expected []int
		actual   []int
		want     bool
		message  string
	}{
		{name: "nil", want: true},
		{name: "empty", expected: []int{}, actual: []int{}, want: true},
		{name: "nil_and_empty", actual: []int{}, want: true},
		{name: "reordered", expected: []int{1, 2, 3}, actual: []int{3, 1, 2}, want: true},
		{name: "duplicate_counts_match", expected: []int{1, 1, 2}, actual: []int{2, 1, 1}, want: true},
		{name: "different_lengths", expected: []int{1}, actual: []int{1, 2}, message: "Should have same length:\nExpected: 1\nActual:   2"},
		{name: "missing_element", expected: []int{1, 2}, actual: []int{1, 3}, message: "Missing element: 2"},
		{name: "duplicate_counts_differ", expected: []int{1, 1, 2}, actual: []int{1, 2, 2}, message: "Missing element: 1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := new(recordingTesting)
			got := ElementsMatch(r, tt.expected, tt.actual)
			checkResult(t, r, got, tt.want, tt.message)
		})
	}

	t.Run("nested_values", func(t *testing.T) {
		t.Parallel()
		r := new(recordingTesting)
		got := ElementsMatch(r,
			[]map[string][]int{{"a": {1, 2}}, {"b": {3}}},
			[]map[string][]int{{"b": {3}}, {"a": {1, 2}}},
		)
		checkResult(t, r, got, true)
	})
}

func TestEqualAndNotEqual(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		expected any
		actual   any
		equal    bool
	}{
		{name: "nil", equal: true},
		{name: "integer", expected: 42, actual: 42, equal: true},
		{name: "different_integer", expected: 42, actual: 43},
		{name: "different_types", expected: int32(42), actual: int64(42)},
		{name: "slice", expected: []int{1, 2}, actual: []int{1, 2}, equal: true},
		{name: "slice_order", expected: []int{1, 2}, actual: []int{2, 1}},
		{name: "nil_and_empty_slice", expected: []int(nil), actual: []int{}},
		{name: "nil_and_empty_map", expected: map[string]int(nil), actual: map[string]int{}},
		{name: "nested_map", expected: map[string][]int{"a": {1, 2}}, actual: map[string][]int{"a": {1, 2}}, equal: true},
		{name: "different_nested_map", expected: map[string][]int{"a": {1, 2}}, actual: map[string][]int{"a": {1, 3}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			equalReport := new(recordingTesting)
			gotEqual := Equal(equalReport, tt.expected, tt.actual)
			checkResult(t, equalReport, gotEqual, tt.equal,
				"Should be equal:", fmt.Sprintf("Expected: %#v", tt.expected), fmt.Sprintf("Actual:   %#v", tt.actual))

			notEqualReport := new(recordingTesting)
			gotNotEqual := NotEqual(notEqualReport, tt.expected, tt.actual)
			checkResult(t, notEqualReport, gotNotEqual, !tt.equal,
				fmt.Sprintf("Should be not equal, value: %#v", tt.expected))
		})
	}
}

func TestNilAndNotNil(t *testing.T) {
	t.Parallel()
	var nilUnsafePointer unsafe.Pointer
	for _, tt := range []struct {
		name  string
		value any
		nil   bool
	}{
		{name: "nil", nil: true},
		{name: "nil_pointer", value: (*int)(nil), nil: true},
		{name: "nil_slice", value: []int(nil), nil: true},
		{name: "nil_map", value: map[string]int(nil), nil: true},
		{name: "nil_channel", value: (chan int)(nil), nil: true},
		{name: "nil_function", value: (func())(nil), nil: true},
		{name: "nil_unsafe_pointer", value: nilUnsafePointer, nil: true},
		{name: "pointer", value: new(int)},
		{name: "empty_slice", value: []int{}},
		{name: "empty_map", value: map[string]int{}},
		{name: "channel", value: make(chan int)},
		{name: "function", value: func() {}},
		{name: "unsafe_pointer", value: unsafe.Pointer(new(int))}, //nolint:gosec // 仅构造有效指针以验证非 nil 判断，不执行指针运算或解引用。
		{name: "zero_integer", value: 0},
		{name: "empty_string", value: ""},
		{name: "false", value: false},
		{name: "struct", value: struct{}{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			nilReport := new(recordingTesting)
			gotNil := Nil(nilReport, tt.value)
			checkResult(t, nilReport, gotNil, tt.nil,
				fmt.Sprintf("Should be nil, but got %#v", tt.value))

			notNilReport := new(recordingTesting)
			gotNotNil := NotNil(notNilReport, tt.value)
			checkResult(t, notNilReport, gotNotNil, !tt.nil, "Should be not nil, but got nil")
		})
	}
}

func TestNotPanics(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name       string
		panicValue any
		panics     bool
		message    string
	}{
		{name: "returns"},
		{name: "string", panicValue: "boom", panics: true, message: "Panic value: boom"},
		{name: "integer", panicValue: 42, panics: true, message: "Panic value: 42"},
		{name: "nil", panics: true, message: "Panic value:"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := new(recordingTesting)
			calls := 0
			got := NotPanics(r, func() {
				calls++
				if tt.panics {
					panic(tt.panicValue)
				}
			})
			if calls != 1 {
				t.Errorf("Expected callback to run once, got %d calls", calls)
			}
			checkResult(t, r, got, !tt.panics, "Should not panic:", tt.message, "Panic stack:", "assert_test.go:")
		})
	}
}

func TestPanics(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name       string
		panicValue any
		panics     bool
	}{
		{name: "returns"},
		{name: "string", panicValue: "boom", panics: true},
		{name: "integer", panicValue: 42, panics: true},
		{name: "nil", panics: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := new(recordingTesting)
			calls := 0
			got := Panics(r, func() {
				calls++
				if tt.panics {
					panic(tt.panicValue)
				}
			})
			if calls != 1 {
				t.Errorf("Expected callback to run once, got %d calls", calls)
			}
			checkResult(t, r, got, tt.panics, "Should panic")
		})
	}
}

func TestTrueAndFalse(t *testing.T) {
	t.Parallel()
	for _, condition := range []bool{true, false} {
		t.Run(strconv.FormatBool(condition), func(t *testing.T) {
			t.Parallel()
			trueReport := new(recordingTesting)
			gotTrue := True(trueReport, condition)
			checkResult(t, trueReport, gotTrue, condition, "Should be true")

			falseReport := new(recordingTesting)
			gotFalse := False(falseReport, condition)
			checkResult(t, falseReport, gotFalse, !condition, "Should be false")
		})
	}
}

func TestZero(t *testing.T) {
	t.Parallel()
	type sample struct {
		Count int
		Name  string
	}
	checkZero(t, "integer_zero", 0, true)
	checkZero(t, "integer_nonzero", 1, false)
	checkZero(t, "float_zero", 0.0, true)
	checkZero(t, "float_nonzero", 1.5, false)
	checkZero(t, "empty_string", "", true)
	checkZero(t, "string", "value", false)
	checkZero(t, "false", false, true)
	checkZero(t, "true", true, false)
	checkZero(t, "nil_pointer", (*int)(nil), true)
	checkZero(t, "pointer_to_zero", new(int), false)
	checkZero(t, "nil_slice", []int(nil), true)
	checkZero(t, "empty_slice", []int{}, false)
	checkZero(t, "nil_map", map[string]int(nil), true)
	checkZero(t, "empty_map", map[string]int{}, false)
	checkZero(t, "nil_channel", (chan int)(nil), true)
	checkZero(t, "channel", make(chan int), false)
	checkZero(t, "nil_function", (func())(nil), true)
	checkZero(t, "function", func() {}, false)
	checkZero(t, "zero_struct", sample{}, true)
	checkZero(t, "nonzero_struct", sample{Count: 1}, false)
	checkZero(t, "zero_array", [2]int{}, true)
	checkZero(t, "nonzero_array", [2]int{0, 1}, false)
	checkZero[any](t, "nil_interface", nil, true)
	checkZero[any](t, "interface_with_typed_nil", (*int)(nil), false)
}

// checkResult 使用标准库检查返回值、错误报告和失败位置标记。
func checkResult(t *testing.T, report *recordingTesting, got, want bool, messages ...string) {
	t.Helper()
	if got != want {
		t.Errorf("Expected assertion result %t, got %t", want, got)
	}
	if want {
		if len(report.messages) != 0 {
			t.Errorf("Successful assertion should report no errors, got %q", report.messages)
		}
		return
	}
	if report.missingHelper || report.helperCalls == 0 {
		t.Error("Failed assertion should call Helper before reporting an error")
	}
	if len(report.messages) != 1 {
		t.Fatalf("Failed assertion should report exactly one error, got %q", report.messages)
	}
	for _, message := range messages {
		if !strings.Contains(report.messages[0], message) {
			t.Errorf("Expected error report to contain %q, got %q", message, report.messages[0])
		}
	}
}

func checkZero[T any](t *testing.T, name string, value T, want bool) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Parallel()
		r := new(recordingTesting)
		got := Zero(r, value)
		checkResult(t, r, got, want, fmt.Sprintf("Should be zero value, but got %#v", value))
	})
}
