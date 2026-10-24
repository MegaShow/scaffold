// Package metainfo 定义方法元信息及其请求上下文传递方式
package metainfo

import (
	"context"
	"net/http"
)

// Meta 提供服务、方法元信息
type Meta interface {
	MethodName() string
	MethodOption(optionName string) map[string]any
	ServiceName() string
}

type contextKey struct{}

// GetMeta 从上下文获取方法元信息，未设置时返回 nil
func GetMeta(ctx context.Context) Meta {
	meta, _ := ctx.Value(contextKey{}).(Meta)
	return meta
}

// Middleware 在调用后续处理器前注入方法元信息
func Middleware(meta Meta) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(WithMeta(r.Context(), meta)))
		})
	}
}

// WithMeta 返回携带方法元信息的上下文
func WithMeta(ctx context.Context, meta Meta) context.Context {
	return context.WithValue(ctx, contextKey{}, meta)
}

type meta struct {
	serviceName string
	methodName  string
	options     map[string]map[string]any
}

// New 创建固定的方法元信息，仅供生成代码调用，禁止业务代码调用
func New(serviceName, methodName string, options map[string]map[string]any) Meta {
	return &meta{
		serviceName: serviceName,
		methodName:  methodName,
		options:     options,
	}
}

// MethodName 返回方法名
func (m *meta) MethodName() string {
	return m.methodName
}

// MethodOption 返回指定完整扩展名称对应的消息选项
func (m *meta) MethodOption(optionName string) map[string]any {
	return m.options[optionName]
}

// ServiceName 返回服务名
func (m *meta) ServiceName() string {
	return m.serviceName
}
