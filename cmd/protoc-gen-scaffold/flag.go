package main

import (
	"flag"
	"fmt"
	"strings"
)

type injectMetaFlag struct {
	enabled bool
	options []string
}

func (f *injectMetaFlag) Set(value string) error {
	switch value {
	case "true":
		f.enabled = true
		return nil
	case "false":
		f.enabled = false
		f.options = nil
		return nil
	}
	for option := range strings.SplitSeq(value, ",") {
		option = strings.TrimSpace(option)
		separator := strings.LastIndexByte(option, '.')
		if separator <= 0 || separator == len(option)-1 {
			return fmt.Errorf("invalid method option %q, expected package.option", option)
		}
		f.options = append(f.options, option)
	}
	f.enabled = true
	return nil
}

func (f *injectMetaFlag) String() string {
	if !f.enabled {
		return "false"
	}
	if len(f.options) == 0 {
		return "true"
	}
	return strings.Join(f.options, ",")
}

type stringsFlag []string

func (f *stringsFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

func (i *stringsFlag) String() string {
	return strings.Join(*i, ",")
}

var (
	format     = flag.String("format", "", "代码格式化工具 (prettier)")
	includes   stringsFlag
	injectMeta injectMetaFlag
	kind       = flag.String("kind", "", "生成代码的类型")
	lang       = flag.String("lang", "", "生成代码的目标语言")
	module     = flag.String("module", "", "Go 模块路径，声明后按 go_package 去除模块前缀生成到对应目录")
	out        = flag.String("out", ".", "生成代码的输出目录")
	prettier   = flag.String("prettier", "", "prettier 可执行文件路径，默认从 PATH 查找")
	protoc     = flag.String("protoc", "", "protoc 可执行文件路径，默认从 PATH 查找")
	version    bool
)

func init() {
	flag.Var(&includes, "include", "proto 包含路径，可重复")
	flag.Var(&injectMeta, "inject-meta", "注入方法元信息，可指定逗号分隔的 package.option")
	flag.BoolVar(&version, "v", false, "输出版本号并退出")
	flag.BoolVar(&version, "version", false, "输出版本号并退出")
}

// validateFormat 校验生成代码的格式化配置
func validateFormat() error {
	switch *format {
	case "":
		return nil
	case "prettier":
		if *prettier == "" {
			*prettier = "prettier"
		}
		return nil
	default:
		return fmt.Errorf("unsupported formatter %q", *format)
	}
}
