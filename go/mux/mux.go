package mux

import "net/http"

var _ Mux = (*http.ServeMux)(nil)

// Mux 抽象 HTTP 路由器，用于将路由注册与具体框架解耦
//
// pattern 采用 Go net/http 风格："METHOD /path"，路径参数统一使用 {name} 语法，如 "GET /users/{user_id}"
type Mux interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// Middleware HTTP 路由中间件
type Middleware func(http.Handler) http.Handler

// Chain 按传入顺序组合中间件，仅供生成代码调用，禁止业务代码调用
func Chain(handler http.HandlerFunc, middlewares ...Middleware) http.HandlerFunc {
	var next http.Handler = handler
	for i := len(middlewares); i > 0; i-- {
		next = middlewares[i-1](next)
	}
	return next.ServeHTTP
}
