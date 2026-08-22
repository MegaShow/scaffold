# Scaffold

[![GitHub Action](https://github.com/megashow/scaffold/actions/workflows/main.yaml/badge.svg)](https://github.com/megashow/scaffold/actions/workflows/main.yaml)
[![CodeCov](https://codecov.io/gh/megashow/scaffold/graph/badge.svg)](https://codecov.io/gh/megashow/scaffold)
[![GitHub Release](https://img.shields.io/github/v/release/megashow/scaffold)](https://github.com/megashow/scaffold/releases)

Scaffold 是一个基于 ProtoBuf 语法的 HTTP IDL 代码生成脚手架工具，用于生成 Server/Client 端的接口定义和使用代码。

IDL 定义虽然使用了 ProtoBuf 协议和语法，但最终生成代码并不依赖 ProtoBuf 协议运行，也不支持使用 ProtoBuf 协议进行通信。

## 语言支持

- Server
  - Golang
- Client
  - TypeScript

## 前置依赖

Scaffold 的 IDL 代码生成依赖于 protoc，因此在使用前需要安装好 protoc。

- protoc: https://github.com/protocolbuffers/protobuf

## 安装

通过源代码编译安装：

```sh
go install go.icytown.com/scaffold@latest
```

或者直接前往 [GitHub Release](https://github.com/megashow/scaffold/releases) 下载对应二进制文件。

## 使用

创建一个 proto 文件。

```proto
// user.proto
edition = "2024";

package com.example.user;

option go_package = "example.com/user";

import "api/api.proto";

message GetUserRequest {
  string user_id = 1;
}

message GetUserResponse {
  string user_id = 1;
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

执行命令。

```sh
protoc-gen-scaffold -lang=go -kind=server -include=. user.proto
```
