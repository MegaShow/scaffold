package mux

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"strings"

	"go.icytown.com/scaffold/mux/form"
)

var (
	_ Parser = (*AdaptParser)(nil)
	_ Parser = (*FormParser)(nil)
	_ Parser = (*JSONParser)(nil)
)

// Parser 抽象从 HTTP 请求读取数据的解析能力。
type Parser interface {
	// Parse 将请求数据解码到 v；v 必须是指向已分配请求对象的非 nil 一级指针。
	// 请求对象由调用方创建，解析器负责填充其字段，不负责分配顶层请求对象。
	Parse(r *http.Request, v any) error
}

// AdaptParser 按请求方法和 Content-Type 选择解析器，未指定时使用 FormParser 和 JSONParser
type AdaptParser struct {
	Form Parser
	JSON Parser
}

// Parse 按请求的 Content-Type 选择解析实现。
func (p *AdaptParser) Parse(r *http.Request, v any) error {
	contentType, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
	contentType = strings.TrimSpace(strings.ToLower(contentType))

	var parser Parser
	switch {
	case r.Method == http.MethodGet || contentType == "application/x-www-form-urlencoded":
		parser = p.Form
		if parser == nil {
			parser = FormParser{}
		}
	case contentType == "application/json":
		parser = p.JSON
		if parser == nil {
			parser = JSONParser{}
		}
	default:
		return errUnsupportedMediaType
	}
	return parser.Parse(r, v)
}

// FormParser 是表单解析器的实现。
type FormParser struct{}

// Parse 将请求 query 或 body 以表单格式解析到 v。
func (FormParser) Parse(r *http.Request, v any) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	if formDecoder, ok := v.(form.Decodable); ok {
		return formDecoder.DecodeForm(r.Form)
	}
	return errors.New("not implementation form.Decodable") // 不应该走到这个分支
}

// JSONParser 是 JSON 解析器的实现。
type JSONParser struct{}

// Parse 将请求 body 以 JSON 解析到 v。
func (JSONParser) Parse(r *http.Request, v any) error {
	return json.UnmarshalRead(r.Body, v)
}
