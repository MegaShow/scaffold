package mux

import (
	"encoding/json/v2"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"go.icytown.com/scaffold/mux/form"
)

var (
	_ Parser = (*AdaptParser)(nil)
	_ Parser = (*FormParser)(nil)
	_ Parser = (*JSONParser)(nil)
	_ Parser = (*MultipartParser)(nil)
)

// Parser 抽象从 HTTP 请求读取数据的解析能力
type Parser interface {
	// Parse 将请求数据解码到 v；v 必须是指向已分配请求对象的非 nil 一级指针
	// 请求对象由调用方创建，解析器负责填充其字段，不负责分配顶层请求对象
	Parse(r *http.Request, v any) error
}

// AdaptParser 按请求方法和 Content-Type 选择解析器，未指定时使用内置解析器
type AdaptParser struct {
	Form      Parser
	JSON      Parser
	Multipart Parser
}

// Parse 按请求的 Content-Type 选择解析实现
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
	case contentType == "multipart/form-data":
		parser = p.Multipart
		if parser == nil {
			parser = MultipartParser{}
		}
	default:
		return errUnsupportedMediaType
	}
	return parser.Parse(r, v)
}

// FormParser 是表单解析器的实现
type FormParser struct{}

// Parse 将请求 query 或 body 以表单格式解析到 v
func (FormParser) Parse(r *http.Request, v any) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	if formDecoder, ok := v.(form.Decodable); ok {
		return formDecoder.DecodeForm(r.Form)
	}
	return errors.New("not implementation DecodeForm") // 不应该走到这个分支
}

// JSONParser 是 JSON 解析器的实现
type JSONParser struct{}

// Parse 将请求 body 以 JSON 解析到 v
func (JSONParser) Parse(r *http.Request, v any) error {
	return json.UnmarshalRead(r.Body, v)
}

// multipartDecodable 由生成的上传请求实现，用于解码表单字段和文件
type multipartDecodable interface {
	DecodeMultipart(form *multipart.Form) error
}

// MultipartParser 解析上传表单
type MultipartParser struct {
	// MaxMemory 是文件内容的内存阈值，超出后写入临时文件，零值使用 32 MiB
	MaxMemory int64
	// MaxBytes 是整个请求体的字节限制，零值不限制；也可在外层使用 http.MaxBytesHandler
	MaxBytes int64
}

// Parse 解析字段和文件
func (p MultipartParser) Parse(r *http.Request, v any) error {
	// 解析失败时清理 multipart 临时文件
	parsed := false
	defer func() {
		if !parsed && r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	if p.MaxBytes > 0 {
		if r.ContentLength > p.MaxBytes {
			return errContentTooLarge
		}
		r.Body = http.MaxBytesReader(nil, r.Body, p.MaxBytes)
	}
	memory := p.MaxMemory
	if memory == 0 {
		memory = 32 << 20 // 默认 32 MiB
	}
	err := r.ParseMultipartForm(memory)
	if err == nil && p.MaxBytes > 0 {
		_, err = io.Copy(io.Discard, r.Body)
	}
	if err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return errContentTooLarge
		}
		if errors.Is(err, multipart.ErrMessageTooLarge) {
			return errContentTooLarge
		}
		return err
	}

	switch decoder := v.(type) {
	case multipartDecodable:
		err = decoder.DecodeMultipart(r.MultipartForm)
	case form.Decodable:
		err = decoder.DecodeForm(r.MultipartForm.Value)
	default:
		return errors.New("not implementation DecodeMultipart or DecodeForm") // 不应该走到这个分支
	}
	if err != nil {
		return err
	}
	parsed = true
	return nil
}
