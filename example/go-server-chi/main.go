package main

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"go.icytown.com/scaffold/example/go-server/gen"
	"go.icytown.com/scaffold/mux"
)

type ServiceImpl struct{}

func (s *ServiceImpl) GetUser(ctx context.Context, req *gen.GetUserRequest) (*gen.GetUserResponse, error) {
	return &gen.GetUserResponse{UserID: req.UserID, Name: "User " + req.UserID}, nil
}

type ChiMux struct {
	chi.Router
}

var _ mux.Mux = (*ChiMux)(nil)

func (m *ChiMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.Router.HandleFunc(pattern, handler)
}

func createRouter() *ChiMux {
	router := &ChiMux{Router: chi.NewRouter()}
	parser := &mux.AdaptParser{}
	renderer := &mux.JSONRenderer{}
	gen.RegisterUserService(router, &ServiceImpl{}, parser, renderer)
	return router
}

func main() {
	server := &http.Server{
		Addr:              ":3000",
		Handler:           createRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	_ = server.ListenAndServe()
}
