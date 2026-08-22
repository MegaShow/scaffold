// Package generator 定义协议无关的脚手架生成器。
package generator

import (
	"fmt"
	"runtime/debug"
)

var (
	injectedVersion string             // 由构建时通过 -ldflags 注入
	scaffoldVersion = resolveVersion() // 实际版本
)

// Version 返回生成器自身的版本号。
func Version() string {
	return scaffoldVersion
}

func resolveVersion() string {
	if injectedVersion != "" {
		return injectedVersion
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	return "dev"
}

// Generator 根据中间模型生成特定语言、特定类型的脚手架代码。
type Generator interface {
	// Generate 为一个 IDL 文件生成脚手架代码。
	Generate(file *File) ([]*GeneratedFile, error)
}

// Language 是生成代码的目标语言。
type Language string

const (
	LanguageGo Language = "go"
)

// Kind 是生成代码的类型。
type Kind string

const (
	KindServer Kind = "server"
	KindClient Kind = "client"
)

// New 根据目标语言和生成类型创建对应的生成器。
func New(lang Language, kind Kind) (Generator, error) {
	switch {
	case lang == LanguageGo && kind == KindServer:
		return &goServerGenerator{}, nil
	default:
		return nil, fmt.Errorf("generator: unsupported language %q with kind %q", lang, kind)
	}
}
