package mux

import (
	"encoding/json/v2"
	"errors"
	"net/http"
)

var _ Renderer = (*JSONRenderer)(nil)

// Renderer 抽象向 HTTP 响应写入数据的渲染能力。
type Renderer interface {
	Render(w http.ResponseWriter, v any, err error) error
}

// JSONRenderer 是 JSON 渲染的实现。
//
// err 为 nil 时以 200 渲染 v；否则渲染 {"error": "..."}，
// 状态码优先取 err 的 StatusCode()，未实现时默认 500。
type JSONRenderer struct{}

// Render 将 v 或 err 以 JSON 渲染并写入响应。
func (JSONRenderer) Render(w http.ResponseWriter, v any, err error) error {
	status := http.StatusOK
	var body any
	if err != nil {
		status = http.StatusInternalServerError
		var sc StatusCoder
		if errors.As(err, &sc) {
			status = sc.StatusCode()
		}
		body = map[string]string{"error": err.Error()}
	} else {
		body = v
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.MarshalWrite(w, body)
}
