# AGENTS.md

AI 编码代理在本仓库工作时的约定，对人类开发者同样适用。

## 项目结构

- `cmd/protoc-gen-scaffold` — protoc 插件入口，支持插件模式（由 protoc 经 stdin 调起）与 CLI 模式
- `internal/generator` — 协议无关的生成器与中间模型
- `internal/template` — 代码模板
- `internal/gen/api` — `api/api.proto` 的生成产物，禁止手改
- `go/mux/` — 独立 Go 模块（自带 go.mod）的 HTTP 路由库
- `example/` — 独立 Go 模块的演示项目，包含 Web 框架集成测试

## 常用命令

| 任务                        | 命令               |
| --------------------------- | ------------------ |
| 构建插件二进制              | `make build`       |
| 重新生成 `internal/gen/api` | `make gen-api`     |
| 重新生成 example 演示代码   | `make gen-example` |
| Lint                        | `make lint`        |
| 测试所有 Go 模块            | `make test`        |

在 Windows 下执行 make 命令时需要使用 Git Bash 进行。

## 依赖工具配置

- 禁止修改或显式指定 Go、Node.js 及相关依赖工具的环境、缓存、镜像或全局配置，包括但不限于 `GOCACHE`、`GOMODCACHE`、`GOPATH`、`GOENV`、`NODE_PATH` 以及 npm、pnpm、yarn 的 cache、store、registry 配置。
- 必须使用用户与运行环境已有的默认配置；默认配置因权限、网络或其他原因不可用时，立即停止相关操作并向用户报告，禁止自行改用仓库目录、临时目录或其他路径。
- 本约束不限制按任务要求维护项目自身的依赖清单与锁文件，例如 `go.mod`、`go.sum`、`package.json` 和 `pnpm-lock.yaml`。

## 验证代码生成

验证 example 生成效果必须使用 `make gen-example`，禁止手工拼 `go run ./cmd/protoc-gen-scaffold ...` 之类的命令。

## 代码约束

- 代码注释一律使用中文；代码中的命名、错误信息、日志、测试名称、断言失败提示等非注释文案一律使用英文。
- 禁止直接修改由脚本或工具生成的代码，应当通过调用 make 来更新对应的生成代码。
- 生成代码要求遵循 golangci-lint 的规范。
- 生成代码函数声明在文件内按函数名的字典序排列。

### 单测组织与命名

- 按被测函数或方法组织单测，每个被测函数或方法单独设置测试入口，不得将多个函数或方法合并为一个行为测试入口。
- 单测优先采用 `internal/assert` 进行测试。
- 避免仅为复用少量代码引入多层回调或泛型测试封装。公共准备和校验逻辑确有必要时可以复用，但每个测试入口只针对对应的被测函数或方法。
- 包级导出函数使用 `Test<函数名>`，例如 `ParseBool` 对应 `TestParseBool`。
- 包级未导出函数使用 `Test_<函数名>`，例如 `parseBool` 对应 `Test_parseBool`。
- 类型方法使用 `Test<类型名>_<方法名>`，例如 `Decoder.Bool` 对应 `TestDecoder_Bool`，`Decoder.scalar` 对应 `TestDecoder_scalar`。
- 类型名、函数名和方法名保留原始大小写；类型名以小写字母开头时，在 `Test` 后增加下划线，例如 `muxError.Error` 对应 `Test_muxError_Error`，确保 Go 能识别测试入口。

### assert 断言库

`internal/assert` 与 `go/mux/internal/assert` 是同一份代码的两份拷贝（因 mux 是独立 Go 模块，无法导入主模块的 internal 包）。修改任一份时必须同步修改另一份，保持两个文件内容完全一致。

## Git 操作约束

- 未经用户明确确认，禁止执行任何会改变仓库状态的 git 命令，包括但不限于 `stash`、`reset`、`checkout`、`clean`、`gc`、`repack`、`commit`、`push`、`rm`。只读命令（`status`、`log`、`diff`、`show`、`cat-file`、`ls-tree`、`fsck`）不受限制。
- 工作区可能存在用户未提交或已暂存的内容。涉及 git 的写操作前，必须先执行 `git status` 向用户报告现状，说明操作目的、影响范围和风险，获得确认后才能执行。
- 需要查看或对比历史版本时，使用 `git show`、`git diff` 等只读命令读取内容，禁止用 `stash` 临时回退工作区。
- 禁止在沙箱内执行 `git stash`、`git gc` 等写入 ref 的操作。
- git 仓库出现异常时，立即停止一切写操作并向用户报告，未经确认不得自行修复。
