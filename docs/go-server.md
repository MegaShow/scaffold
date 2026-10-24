# Go Server 使用文档

本文档介绍如何使用 Scaffold 生成和使用 Golang 服务代码。

## 快速开始

### 声明 IDL 和生成服务代码

创建一个 proto 文件，用于定义 Web 应用所支持的 HTTP 接口。

```proto
// user.proto
edition = "2024";

package com.example.user;

option go_package = "example.com/user";

import "scaffold/api.proto";

message GetUserRequest {
  uint32 user_id = 1;
}

message GetUserResponse {
  uint32 user_id = 1;
  string name = 2;
}

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (api.http) = {
      get: "/users/{user_id}"
    };
  }
}
```

运行 Scaffold 生成 Golang 代码。

```sh
protoc-gen-scaffold -lang=go -kind=server user.proto
```

### 安装依赖

安装 Scaffold 的 Golang 依赖。

```sh
go get go.icytown.com/scaffold/mux
```

### 实现服务接口

定义类型 ServiceImpl，实现之前声明的服务接口。

```go
type ServiceImpl struct{}

func (s *ServiceImpl) GetUser(ctx context.Context, req *gen.GetUserRequest) (*gen.GetUserResponse, error) {
	return &gen.GetUserResponse{UserID: req.UserID, Name: "User " + req.UserID}, nil
}
```

### 在 Web 框架中注册服务

Scaffold 支持实现 mux.Mux 接口的 Golang Web 框架，如果暂不支持以下接口，可自行封装。

```go
// Mux 抽象 HTTP 路由器，用于将路由注册与具体框架解耦。
//
// pattern 采用 Go net/http 风格："METHOD /path"，路径参数统一使用 {name} 语法，如 "GET /users/{user_id}"。
type Mux interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}
```

#### net/http

Golang 标准库的 ServeMux 默认已实现 mux.Mux 接口，仅需要按以下方式接入即可。

```go
serveMux := http.NewServeMux()
parser := &mux.AdaptParser{}
renderer := &mux.JSONRenderer{}
gen.RegisterUserService(serveMux, &ServiceImpl{}, parser, renderer)

http.ListenAndServe(":3000", serveMux)
```

#### github.com/go-chi/chi

chi 的 HandleFunc 支持 `"METHOD /path"` 格式，但它接收的处理函数类型是 `http.HandlerFunc`，与 mux.Mux 中的裸函数类型签名不完全一致，因此需要添加一个简单的类型适配器。

```go
type ChiMux struct {
	chi.Router
}

var _ mux.Mux = (*ChiMux)(nil)

func (m *ChiMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.Router.HandleFunc(pattern, handler)
}
```

创建 ChiMux，并按照与标准库相同的方式注册生成的服务。

```go
router := &ChiMux{Router: chi.NewRouter()}
parser := &mux.AdaptParser{}
renderer := &mux.JSONRenderer{}
gen.RegisterUserService(router, &ServiceImpl{}, parser, renderer)

http.ListenAndServe(":3000", router)
```

#### github.com/gin-gonic/gin

gin 使用 `Handle(method, path, handlers...)` 注册路由，并使用 `:name` 表示路径参数，因此需要适配路由格式。Scaffold 通过标准库的 `Request.PathValue` 读取路径参数，适配器还需要将 gin 解析出的参数写入 request。

```go
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
```

创建 GinMux，并注册生成的服务。

```go
router := &GinMux{Engine: gin.Default()}
parser := &mux.AdaptParser{}
renderer := &mux.JSONRenderer{}
gen.RegisterUserService(router, &ServiceImpl{}, parser, renderer)

http.ListenAndServe(":3000", router)
```

## Scaffold Mux

Scaffold 提供了 mux 包，定义与具体路由实现解耦的 Mux、Parser、Renderer 等接口抽象和部分实现。因此，你可以将 Scaffold 与标准库、chi、gin 等路由库一起使用。

在 mux 包中，有以下概念。

- Mux: 路由实现的抽象。
- Parser: 将请求转成接口入参的抽象。
- Renderer: 渲染最终响应的抽象。

mux 包提供了一些常用的 Parser、Renderer，比如 AdaptParser、FormParser、JSONParser，以及 JSONRenderer，你也可以自行定义属于自己的 Parser 和 Renderer。

## 常见用法

### 指定 HTTP 状态码

mux.JSONRenderer 在接口返回 error 时，默认使用 HTTP 500 状态码。如果 error 实现了 mux.StatusCoder 接口，则使用该 error 指定的状态码。

```go
type HTTPError struct {
	statusCode int
	err        error
}

func (e *HTTPError) Error() string {
	return e.err.Error()
}

func (e *HTTPError) StatusCode() int {
	return e.statusCode
}

func (e *HTTPError) Unwrap() error {
	return e.err
}
```

在服务实现中返回 HTTPError，即可指定响应的 HTTP 状态码。

```go
func (s *ServiceImpl) GetUser(ctx context.Context, req *gen.GetUserRequest) (*gen.GetUserResponse, error) {
	if req.UserID != 1 {
		return nil, &HTTPError{
			statusCode: http.StatusNotFound,
			err:        errors.New("user not found"),
		}
	}
	return &gen.GetUserResponse{UserID: req.UserID, Name: "Alice"}, nil
}
```

上述错误由 mux.JSONRenderer 渲染后，HTTP 状态码为 404，响应体为 `{"error":"user not found"}`。JSONRenderer 使用 errors.As 查找 mux.StatusCoder，因此使用 `%w` 包装该错误后，状态码仍然有效。

### 通用 Response 封装

如果希望所有接口返回统一的 JSON 结构，可以实现自定义 mux.Renderer。下面的示例同时保留 HTTP 状态码语义，并将状态码、消息和业务数据封装到响应体中。

Render 接收当前请求 `r *http.Request`，可通过 `r.Context()` 读取请求上下文，或根据请求头调整响应。下例不需要请求信息，因此使用 `_` 忽略该参数。

```go
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type ResponseRenderer struct{}

func (ResponseRenderer) Render(w http.ResponseWriter, _ *http.Request, v any, err error) error {
	status := http.StatusOK
	response := &Response{
		Code:    status,
		Message: "success",
		Data:    v,
	}
	if err != nil {
		status = http.StatusInternalServerError
		var statusCoder mux.StatusCoder
		if errors.As(err, &statusCoder) {
			status = statusCoder.StatusCode()
		}
		response = &Response{
			Code:    status,
			Message: err.Error(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.MarshalWrite(w, response)
}
```

将注册服务时使用的 Renderer 替换为自定义实现即可。

```go
renderer := &ResponseRenderer{}
gen.RegisterUserService(serveMux, &ServiceImpl{}, parser, renderer)
```

成功响应示例如下。

```json
{ "code": 200, "message": "success", "data": { "user_id": 1, "name": "Alice" } }
```

错误响应示例如下。

```json
{ "code": 404, "message": "user not found" }
```

### 传递方法元信息

生成服务代码时设置 `-inject-meta=true`，生成的路由就会在调用中间件和业务方法前将服务名和方法名写入请求的 `context.Context`：

```sh
protoc-gen-scaffold -lang=go -kind=server -inject-meta=true user.proto
```

业务方法通过 metainfo.GetMeta(ctx) 读取元信息。ServiceName() 和 MethodName() 返回不含 proto 包名的名称；未启用注入时，GetMeta(ctx) 返回 `nil`。

如果还要读取 proto 中定义的业务选项，可以在前面的 user.proto 中增加消息类型的方法扩展：

```proto
import "google/protobuf/descriptor.proto";

message AccessRule {
  string role = 1;
  bool audit = 2;
}

extend google.protobuf.MethodOptions {
  AccessRule access = 1000;
}
```

然后在 GetUser 的 RPC 定义中设置该选项。

```proto
option (com.example.user.access) = {
  role: "admin"
  audit: true
};
```

生成时传入选项的完整名称，多个选项用逗号分隔，例如 `-inject-meta=com.example.user.access,com.example.user.audit`。

```sh
protoc-gen-scaffold -lang=go -kind=server -inject-meta=com.example.user.access user.proto
```

生成的路由处理链会用 `metainfo.WithMeta` 将选中的选项写入请求上下文。业务代码可以按完整名称读取该选项的消息字段，未选中的选项不会进入生成代码：

```go
func (s *ServiceImpl) GetUser(ctx context.Context, req *gen.GetUserRequest) (*gen.GetUserResponse, error) {
	meta := metainfo.GetMeta(ctx)
	if meta != nil {
		if access := meta.MethodOption("com.example.user.access"); access != nil && access["audit"] == true {
			fmt.Printf("calling %s.%s\n", meta.ServiceName(), meta.MethodName())
		}
	}
	return &gen.GetUserResponse{UserID: req.UserID, Name: "Alice"}, nil
}
```

### 使用中间件

注册 service 时可以传入一个或多个 `mux.Middleware` 中间件，它们按传入顺序执行，并且只作用于该 service 的路由。中间件的签名是 `func(http.Handler) http.Handler`。下面的示例打印请求处理前后的信息。

```go
func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("before: %s %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		fmt.Printf("after: %s %s\n", r.Method, r.URL.Path)
	})
}

gen.RegisterUserService(serveMux, service, parser, renderer, logRequest)
```
