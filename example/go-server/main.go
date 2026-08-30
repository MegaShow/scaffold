package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.icytown.com/scaffold/example/go-server/gen"
	"go.icytown.com/scaffold/mux"
)

type CustomError struct {
	code int32
	err  error
}

func (e *CustomError) Error() string {
	return e.err.Error()
}

func (e *CustomError) StatusCode() int32 {
	return e.code
}

type ServiceImpl struct {
	mu        sync.RWMutex
	maxUserID int32
	users     map[int32]string
}

func (s *ServiceImpl) CreateUser(ctx context.Context, req *gen.CreateUserRequest) (*gen.CreateUserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.users == nil {
		s.users = make(map[int32]string)
	}
	s.maxUserID++
	s.users[s.maxUserID] = req.Name
	return &gen.CreateUserResponse{UserID: s.maxUserID, Name: req.Name}, nil
}

func (s *ServiceImpl) GetUser(ctx context.Context, req *gen.GetUserRequest) (*gen.GetUserResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if req.UserID == 123 {
		return nil, fmt.Errorf("no permission, ts: %d", req.TS)
	}
	name, ok := s.users[req.UserID]
	if !ok {
		return nil, &CustomError{code: http.StatusNotFound, err: errors.New("user not found")}
	}
	return &gen.GetUserResponse{UserID: req.UserID, Name: name}, nil
}

func createServeMux() *http.ServeMux {
	serveMux := http.NewServeMux()
	parser := &mux.AdaptParser{Form: &mux.FormParser{}, JSON: &mux.JSONParser{}}
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
