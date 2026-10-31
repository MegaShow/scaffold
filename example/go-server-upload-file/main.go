package main

import (
	"context"
	"net/http"
	"time"

	"go.icytown.com/scaffold/example/go-server-upload-file/gen"
	"go.icytown.com/scaffold/mux"
)

// statusError 为示例业务错误提供 HTTP 状态码
type statusError struct {
	status  int
	message string
}

func (e *statusError) Error() string {
	return e.message
}

func (e *statusError) StatusCode() int {
	return e.status
}

type ServiceImpl struct{}

func (s *ServiceImpl) UploadFile(_ context.Context, req *gen.UploadFileRequest) (*gen.UploadFileResponse, error) {
	if req.File == nil {
		return nil, &statusError{status: http.StatusBadRequest, message: "file is required"}
	}
	response := &gen.UploadFileResponse{
		Name:  req.Name,
		File:  &gen.FileInfo{Filename: req.File.Filename, Size: req.File.Size},
		Files: make([]*gen.FileInfo, 0, len(req.Files)),
	}
	for _, file := range req.Files {
		response.Files = append(response.Files, &gen.FileInfo{Filename: file.Filename, Size: file.Size})
	}
	return response, nil
}

func createServeMux() *http.ServeMux {
	serveMux := http.NewServeMux()
	parser := &mux.AdaptParser{Multipart: mux.MultipartParser{
		MaxMemory: 8 << 20,   // 内存阈值 8 MiB
		MaxBytes:  128 << 20, // 请求体上限 128 MiB
	}}
	gen.RegisterFileService(serveMux, &ServiceImpl{}, parser, mux.JSONRenderer{})
	return serveMux
}

func main() {
	server := &http.Server{
		Addr:              ":3000",
		Handler:           createServeMux(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	_ = server.ListenAndServe()
}
