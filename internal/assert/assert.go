// Package assert 提供了测试用的断言工具.
package assert

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"
)

type Testing interface {
	Errorf(format string, args ...any)
	Helper()
}

// report 在断言失败时输出消息，并把用户传入的描述单独作为 Msg 行输出。
func report(t Testing, errMsg string, prefix string, msgs []string) {
	t.Helper()
	if len(msgs) > 0 {
		if prefix == "" {
			prefix = "Msg: "
		}
		errMsg += "\n" + prefix + strings.Join(msgs, " ")
	}
	t.Errorf("%s", errMsg)
}

func ElementsMatch[T any](t Testing, expected, actual []T, msgs ...string) bool {
	if len(expected) != len(actual) {
		t.Helper()
		report(t, fmt.Sprintf("Should have same length:\n"+
			"Expected: %d\n"+
			"Actual:   %d", len(expected), len(actual)),
			"Msg:      ", msgs)
		return false
	}

	matched := make([]bool, len(actual))
	for _, expectedValue := range expected {
		found := false
		for idx, actualValue := range actual {
			if !matched[idx] && reflect.DeepEqual(expectedValue, actualValue) {
				matched[idx] = true
				found = true
				break
			}
		}
		if !found {
			t.Helper()
			report(t, fmt.Sprintf("Element not found in actual slice:\n"+
				"Missing element: %#v\n"+
				"Expected: %#v\n"+
				"Actual:   %#v", expectedValue, expected, actual),
				"Msg:      ", msgs)
			return false
		}
	}
	return true
}

func Equal[T any](t Testing, expected, actual T, msgs ...string) bool {
	if !reflect.DeepEqual(expected, actual) {
		t.Helper()
		report(t, fmt.Sprintf("Should be equal:\n"+
			"Expected: %#v\n"+
			"Actual:   %#v", expected, actual),
			"Msg:      ", msgs)
		return false
	}
	return true
}

func False(t Testing, condition bool, msgs ...string) bool {
	if condition {
		t.Helper()
		report(t, "Should be false", "", msgs)
		return false
	}
	return true
}

func Nil(t Testing, value any, msgs ...string) bool {
	if !isNil(value) {
		t.Helper()
		report(t, fmt.Sprintf("Should be nil\n"+
			"Actual: %#v", value),
			"Msg:    ", msgs)
		return false
	}
	return true
}

func NotEqual[T any](t Testing, expected, actual T, msgs ...string) bool {
	if reflect.DeepEqual(expected, actual) {
		t.Helper()
		report(t, fmt.Sprintf("Should be not equal\n"+
			"Compared: %#v\n"+
			"Actual:   %#v", expected, actual),
			"Msg:      ", msgs)
		return false
	}
	return true
}

func NotNil(t Testing, value any, msgs ...string) bool {
	if isNil(value) {
		t.Helper()
		report(t, "Should be not nil, but got nil", "", msgs)
		return false
	}
	return true
}

func NotPanics(t Testing, f func(), msgs ...string) bool {
	x, stack, recovered := runAndRecover(f)
	if recovered {
		t.Helper()
		report(t, fmt.Sprintf("Should not panic:\n"+
			"Panic value: %v\n"+
			"Panic stack: %s", x, stack), "", msgs)
		return false
	}
	return true
}

func Panics(t Testing, f func(), msgs ...string) bool {
	_, _, recovered := runAndRecover(f)
	if !recovered {
		t.Helper()
		report(t, "Should panic", "", msgs)
		return false
	}
	return true
}

func True(t Testing, condition bool, msgs ...string) bool {
	if !condition {
		t.Helper()
		report(t, "Should be true", "", msgs)
		return false
	}
	return true
}

func Zero[T any](t Testing, value T, msgs ...string) bool {
	var zero T
	if !reflect.DeepEqual(value, zero) {
		t.Helper()
		report(t, fmt.Sprintf("Should be zero value\n"+
			"Actual: %#v", value),
			"Msg:    ", msgs)
		return false
	}
	return true
}

func isNil(obj any) bool {
	if obj == nil {
		return true
	}
	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return value.IsNil()
	default:
		return false
	}
}

func runAndRecover(f func()) (any, string, bool) {
	var x any
	var stack string
	recovered := true
	func() {
		defer func() {
			x = recover()
			if recovered {
				stack = string(debug.Stack())
			}
		}()
		f()
		recovered = false
	}()
	return x, stack, recovered
}
