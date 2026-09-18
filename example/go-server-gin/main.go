package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"go.icytown.com/scaffold/example/go-server/gen"
	"go.icytown.com/scaffold/mux"
)

type ServiceImpl struct{}

func (s *ServiceImpl) GetUser(ctx context.Context, req *gen.GetUserRequest) (*gen.GetUserResponse, error) {
	return &gen.GetUserResponse{UserID: req.UserID, Name: "User " + req.UserID}, nil
}

type GinMux struct {
	*gin.Engine
}

var _ mux.Mux = (*GinMux)(nil)

func (m *GinMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	method, path, ok := strings.Cut(pattern, " ")
	if !ok {
		panic("invalid route pattern: " + pattern)
	}
	path = strings.NewReplacer("{", ":", "}", "").Replace(path)
	m.Handle(method, path, func(c *gin.Context) {
		for _, param := range c.Params {
			c.Request.SetPathValue(param.Key, param.Value)
		}
		handler(c.Writer, c.Request)
	})
}

func createRouter() *GinMux {
	router := &GinMux{Engine: gin.New()}
	parser := &mux.AdaptParser{Form: &mux.FormParser{}, JSON: &mux.JSONParser{}}
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
