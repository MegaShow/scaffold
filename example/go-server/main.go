package main

import (
	"context"
	"net/http"
	"time"

	"go.icytown.com/scaffold/example/go-server/gen"
	"go.icytown.com/scaffold/mux"
)

type ServiceImpl struct{}

func (s *ServiceImpl) GetUser(ctx context.Context, req *gen.GetUserRequest) (*gen.GetUserResponse, error) {
	return &gen.GetUserResponse{UserID: req.UserID, Name: "User " + req.UserID}, nil
}

func createServeMux() *http.ServeMux {
	serveMux := http.NewServeMux()
	parser := &mux.AdaptParser{}
	renderer := &mux.JSONRenderer{}
	gen.RegisterUserService(serveMux, &ServiceImpl{}, parser, renderer)
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
