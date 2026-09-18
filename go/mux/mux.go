package mux

import "net/http"

var _ Mux = (*http.ServeMux)(nil)

// Mux 抽象 HTTP 路由器，用于将路由注册与具体框架解耦。
//
// pattern 采用 Go net/http 风格："METHOD /path"，路径参数统一使用 {name} 语法，如 "GET /users/{user_id}"。
type Mux interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}
