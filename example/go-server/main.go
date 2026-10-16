package main

import (
	"context"
	"net/http"
	"time"

	"go.icytown.com/scaffold/example/go-server/gen"
	"go.icytown.com/scaffold/mux"
	"go.icytown.com/scaffold/mux/metainfo"
)

type ServiceImpl struct{}

func (s *ServiceImpl) GetUser(ctx context.Context, req *gen.GetUserRequest) (*gen.GetUserResponse, error) {
	return &gen.GetUserResponse{UserID: req.UserID, Name: "User " + req.UserID}, nil
}

func middleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := metainfo.GetMeta(r.Context())
		if meta == nil {
			http.Error(w, "method metadata is missing", http.StatusInternalServerError)
			return
		}
		w.Header().Set("X-Scaffold-Method", meta.MethodName())
		next(w, r)
	})
}

func createServeMux() *http.ServeMux {
	serveMux := http.NewServeMux()
	parser := &mux.AdaptParser{}
	renderer := &mux.JSONRenderer{}
	gen.RegisterUserService(serveMux, &ServiceImpl{}, parser, renderer, middleware)
	return serveMux
}

func main() {
	serveMux := createServeMux()
	server := &http.Server{
		Addr:              ":3000",
		Handler:           serveMux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	_ = server.ListenAndServe()
}
