# AGENTS.md

AI 编码代理在本仓库工作时的约定，对人类开发者同样适用。

## 项目结构

- `cmd/protoc-gen-scaffold` — protoc 插件入口，支持插件模式（由 protoc 经 stdin 调起）与 CLI 模式
- `internal/generator` — 协议无关的生成器与中间模型
- `internal/template` — 代码模板
- `internal/gen/api` — `api/api.proto` 的生成产物，禁止手改
- `mux/` — 独立 Go 模块（自带 go.mod）的 HTTP 路由库
- `example/` — 演示项目

## 常用命令

| 任务                        | 命令                      |
| --------------------------- | ------------------------- |
| 构建插件二进制              | `make build`              |
| 重新生成 `internal/gen/api` | `make gen-api`            |
| 重新生成 example 演示代码   | `make gen-example`        |
| Lint                        | `golangci-lint run ./...` |

在 Windows 下执行 make 命令时需要使用 Git Bash 进行。

## 验证代码生成

验证 example 生成效果必须使用 `make gen-example`，禁止手工拼 `go run ./cmd/protoc-gen-scaffold ...` 之类的命令。

## 代码风格

- 注释一律使用中文。
- 禁止直接修改由脚本或工具生成的代码，应当通过调用 make 来更新对应的生成代码。
- 生成代码要求遵循 golangci-lint 的规范。
- 如果没有明确要求，禁止自行生成单测文件，可以在已有单测文件上补充用例。
