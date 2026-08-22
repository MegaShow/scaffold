// Package template 提供脚手架生成器的模板渲染设施：内嵌模板文件、自定义函数注册与模板执行。
package template

import (
	"bytes"
	"embed"
	"fmt"
	"maps"
	"sync"
	"text/template"
)

//go:embed *.tpl
var templateFS embed.FS

var (
	templateMu    sync.Mutex
	templateFuncs = template.FuncMap{}
	parsed        *template.Template
)

// RegisterFuncs 注册渲染需要的函数。
func RegisterFuncs(f template.FuncMap) {
	templateMu.Lock()
	defer templateMu.Unlock()
	maps.Copy(templateFuncs, f)
	parsed = nil // 置空让渲染模板重新加载
}

// Render 渲染指定模板并返回内容。
func Render(name string, data any) ([]byte, error) {
	templateMu.Lock()
	defer templateMu.Unlock()
	if parsed == nil {
		t, err := template.New("").Funcs(templateFuncs).ParseFS(templateFS, "*.tpl")
		if err != nil {
			return nil, fmt.Errorf("template: parse templates: %w", err)
		}
		parsed = t
	}

	var buf bytes.Buffer
	if err := parsed.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, fmt.Errorf("template: render template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}
