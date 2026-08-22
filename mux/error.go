package mux

import "net/http"

var (
	// ErrBadRequest 表示请求格式错误，渲染时映射为 HTTP 400。
	ErrBadRequest           = &muxError{msg: "400 bad request", statusCode: http.StatusBadRequest}
	errUnsupportedMediaType = &muxError{msg: "415 unsupported media type", statusCode: http.StatusUnsupportedMediaType}
)

var _ StatusCoder = (*muxError)(nil)

// StatusCoder 抽象错误携带 HTTP 状态码的能力，
// 实现该接口的错误在渲染时会使用其状态码。
type StatusCoder interface {
	StatusCode() int
}

type muxError struct {
	msg        string
	statusCode int
}

func (e *muxError) Error() string {
	return e.msg
}

func (e *muxError) StatusCode() int {
	return e.statusCode
}
