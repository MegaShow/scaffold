package main

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"

	"go.icytown.com/scaffold/example/go-server/gen"
)

type ServiceImpl struct {
	maxUserID atomic.Int32
}

func (s *ServiceImpl) CreateUser(ctx context.Context, req *gen.CreateUserRequest) (*gen.CreateUserResponse, error) {
	userID := s.maxUserID.Add(1)
	return &gen.CreateUserResponse{UserId: userID, Name: req.Name}, nil
}

func (s *ServiceImpl) GetUser(ctx context.Context, req *gen.GetUserRequest) (*gen.GetUserResponse, error) {
	return &gen.GetUserResponse{UserId: req.UserId, Name: fmt.Sprintf("User %d", req.UserId)}, nil
}

func main() {
	serveMux := http.NewServeMux()
	gen.RegisterUserService(serveMux, &ServiceImpl{})
}
