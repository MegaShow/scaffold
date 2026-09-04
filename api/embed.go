// Package api 通过 go:embed 内嵌 api.proto，使 protoc-gen-scaffold 在 CLI 模式下
// 可将其释放到可执行文件同目录，供 protoc 解析 import "api/api.proto"。
// 这样即便通过 go install 安装到 GOPATH/bin，也无需手动拷贝 api 目录。
package api

import _ "embed"

// Proto 是 api/api.proto 的原始内容，在编译期内嵌进二进制。
//
//go:embed api.proto
var Proto []byte
