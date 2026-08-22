package mux

import (
	"encoding/json/v2"
	"net/http"
	"strings"
)

var (
	_ Parser = (*AdaptParser)(nil)
	_ Parser = (*JSONParser)(nil)
)

// Parser 抽象从 HTTP 请求读取数据的解析能力。
type Parser interface {
	Parse(r *http.Request, v any) error
}

// AdaptParser 是多种解析器的封装，会根据请求的 Content-Type 使用不同解析器。
type AdaptParser struct {
	JSON Parser
}

// Parse 按请求的 Content-Type 选择解析实现。
func (p *AdaptParser) Parse(r *http.Request, v any) error {
	contentType, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	switch contentType {
	case "application/json":
		if p.JSON != nil {
			return p.JSON.Parse(r, v)
		}
	}
	return errUnsupportedMediaType
}

// JSONParser 是 JSON 解析器的实现。
type JSONParser struct{}

// Parse 将请求 Body 以 JSON 解析到 v。
func (JSONParser) Parse(r *http.Request, v any) error {
	return json.UnmarshalRead(r.Body, v)
}
