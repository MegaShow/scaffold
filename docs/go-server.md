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

import "api/api.proto";

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

### 实现服务接口

定义类型 ServiceImpl，实现之前声明的服务接口。

```go
type UserService interface {
	GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error)
}

func RegisterUserService(m mux.Mux, service UserService, parser mux.Parser, renderer mux.Renderer) {
	s := wUserService{service: service, parser: parser, renderer: renderer}
	m.HandleFunc("GET /users/{user_id}", s.GetUser)
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

#### http.ServeMux

Golang 标准库的 ServeMux 默认已实现 mux.Mux 接口，仅需要按以下方式接入即可。

```go
serveMux := http.NewServeMux()
parser := &mux.AdaptParser{Form: &mux.FormParser{}, JSON: &mux.JSONParser{}}
renderer := &mux.JSONRenderer{}
gen.RegisterUserService(serveMux, &ServiceImpl{}, parser, renderer)

http.ListenAndServe(":3000", serveMux)
```

#### go-chi/chi

待补充。

## Scaffold Mux

Scaffold 提供了 mux 包，定义与具体路由实现解耦的 Mux、Parser、Renderer 等接口抽象和部分实现。因此，你可以将 Scaffold 与标准库、chi、gin 等路由库一起使用。

在 mux 包中，有以下概念。

- Mux: 路由实现的抽象。
- Parser: 将请求转成接口入参的抽象。
- Renderer: 渲染最终响应的抽象。

mux 包提供了一些常用的 Parser、Renderer，比如 AdaptParser、FormParser、JSONParser，以及 JSONRenderer，你也可以自行定义属于自己的 Parser 和 Renderer。
