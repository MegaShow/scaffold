// protoc-gen-scaffold 是 protoc 插件，也可直接作为命令行工具使用：
//   - 插件模式：由 protoc 经 stdin 传入 CodeGeneratorRequest 调起，生成脚手架代码；
//   - CLI 模式：命令行直接调用并携带 .proto 文件，读取 lang/kind 等参数后组装
//     protoc 命令并启动子进程，由 protoc 以插件方式再次调起本程序完成生成。
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"go.icytown.com/scaffold/api"
	"go.icytown.com/scaffold/internal/generator"
)

type stringsFlag []string

func (f *stringsFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

func (i *stringsFlag) String() string { return strings.Join(*i, ",") }

var (
	includes stringsFlag
	format   = flag.String("format", "", "代码格式化工具 (prettier)")
	kind     = flag.String("kind", "", "生成代码的类型")
	lang     = flag.String("lang", "", "生成代码的目标语言")
	module   = flag.String("module", "", "Go 模块路径，声明后按 go_package 去除模块前缀生成到对应目录")
	out      = flag.String("out", ".", "生成代码的输出目录")
	prettier = flag.String("prettier", "", "prettier 可执行文件路径，默认从 PATH 查找")
	protoc   = flag.String("protoc", "", "protoc 可执行文件路径，默认从 PATH 查找")
	version  bool
)

func init() {
	flag.Var(&includes, "include", "proto 包含路径，可重复")
	flag.BoolVar(&version, "v", false, "输出版本号并退出")
	flag.BoolVar(&version, "version", false, "输出版本号并退出")
}

func main() {
	flag.Parse()

	// 输出版本号后退出
	if version {
		_, _ = fmt.Fprintf(os.Stdout, "protoc-gen-scaffold %s\n", generator.Version())
		return
	}

	// 携带 .proto 文件参数时进入 CLI 模式
	if files := flag.Args(); len(files) > 0 {
		if err := runCLI(files); err != nil {
			fmt.Fprintf(os.Stderr, "protoc-gen-scaffold: %s\n", err.Error())
			os.Exit(1)
		}
		return
	}

	// 无文件参数且 stdin 是终端，视为用户直接运行，提示用法后退出
	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		fmt.Fprintln(os.Stderr, "Usage: protoc-gen-scaffold [flags] file.proto...")
		fmt.Fprintln(os.Stderr, "Plugin mode: invoked by protoc via --scaffold_out")
		flag.PrintDefaults()
		os.Exit(2)
	}

	// 插件模式
	protogen.Options{ParamFunc: flag.CommandLine.Set}.Run(func(gen *protogen.Plugin) error {
		if err := validateFormat(); err != nil {
			return fmt.Errorf("protoc-gen-scaffold: %w", err)
		}
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL | pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)
		gen.SupportedEditionsMinimum = descriptorpb.Edition_EDITION_PROTO3
		gen.SupportedEditionsMaximum = descriptorpb.Edition_EDITION_2024
		g, err := generator.New(generator.Language(*lang), generator.Kind(*kind))
		if err != nil {
			return fmt.Errorf("protoc-gen-scaffold: %w", err)
		}
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			if err := generateFile(gen, g, f); err != nil {
				return err
			}
		}
		return nil
	})
}

// runCLI 以命令行模式运行：定位 protoc 与自身路径，组装并执行 protoc 命令，
// 由 protoc 以插件方式调起本程序完成代码生成。
func runCLI(files []string) error {
	// 提前校验生成器组合，避免启动 protoc 子进程后才报错。
	if _, err := generator.New(generator.Language(*lang), generator.Kind(*kind)); err != nil {
		return err
	}
	if err := validateFormat(); err != nil {
		return err
	}

	protocPath := *protoc
	if protocPath == "" {
		var err error
		protocPath, err = exec.LookPath("protoc")
		if err != nil {
			return fmt.Errorf("protoc not found in PATH, specify it with -protoc: %w", err)
		}
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve own executable path: %w", err)
	}

	// 确保可执行文件同目录下存在 scaffold/api.proto，供 protoc 通过 -I 解析
	// import "scaffold/api.proto"。go install 安装场景下该文件不会随二进制分发，
	// 因此从内嵌内容释放；已存在且内容一致时跳过，内容过期则同步为当前版本
	if err := ensureApiProto(filepath.Dir(self)); err != nil {
		return err
	}

	// 声明 module 时按 go_package 剥离模块前缀定位输出目录，否则镜像 proto 源文件相对路径
	paths := "source_relative"
	if *module != "" {
		paths = "import,module=" + *module
	}

	options := []string{"paths=" + paths, "lang=" + *lang, "kind=" + *kind, "format=" + *format}
	if *format == "prettier" {
		options = append(options, "prettier="+*prettier)
	}

	args := make([]string, 0, len(includes)+len(files)+3)
	args = append(args,
		"--plugin=protoc-gen-scaffold="+self,
		"--scaffold_out="+*out,
		"--scaffold_opt="+strings.Join(options, ","),
		"-I.",
		"-I"+filepath.Dir(self),
	)
	for _, inc := range includes {
		args = append(args, "-I"+inc)
	}
	args = append(args, files...)

	//nolint:gosec,noctx // protoc 路径来自用户 -protoc 参数或 PATH，属 CLI 核心功能，子进程生命周期与自身一致无需 context
	cmd := exec.Command(protocPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// validateFormat 校验生成代码的格式化配置。
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

// ensureApiProto 确保 selfDir 下存在 scaffold/api.proto 且内容与内嵌版本一致
// 文件缺失时创建；存在但与内嵌内容不一致时（例如旧版本遗留）覆盖为当前版本，
// 保证 protoc 解析到的 api.proto 与当前二进制匹配
func ensureApiProto(selfDir string) error {
	apiDir := filepath.Join(selfDir, "scaffold")
	target := filepath.Join(apiDir, "api.proto")
	content := apiProtoContent()

	if existing, err := os.ReadFile(target); err == nil { //nolint:gosec // target 由 os.Executable() 目录拼接固定子路径得到，非用户输入
		if bytes.Equal(existing, content) {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", target, err)
	}

	if err := os.MkdirAll(apiDir, 0o750); err != nil {
		return fmt.Errorf("create api dir %s: %w", apiDir, err)
	}
	if err := os.WriteFile(target, content, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	return nil
}

// apiProtoContent 返回包含当前生成器版本的 API IDL 内容。
func apiProtoContent() []byte {
	header := fmt.Sprintf("// Scaffold API version: %s\n\n", generator.Version())
	content := make([]byte, 0, len(header)+len(api.Proto))
	content = append(content, header...)
	return append(content, api.Proto...)
}

// generateFile 把单个 protobuf 文件转换为中间模型并生成脚手架代码，
// 生成结果写回插件响应。
func generateFile(gen *protogen.Plugin, g generator.Generator, file *protogen.File) error {
	files, err := g.Generate(toFile(gen, file))
	if err != nil {
		return err
	}

	for _, f := range files {
		if err := formatGeneratedFile(f); err != nil {
			return err
		}
		out := gen.NewGeneratedFile(f.Path, file.GoImportPath)
		if _, err := out.Write(f.Content); err != nil {
			return err
		}
	}
	return nil
}

// formatGeneratedFile 按配置选择生成代码的格式化方式。
func formatGeneratedFile(file *generator.GeneratedFile) error {
	switch *format {
	case "":
		return nil
	case "prettier":
		return formatWithPrettier(file)
	default:
		return fmt.Errorf("unsupported formatter %q", *format)
	}
}

// formatWithPrettier 使用 Prettier 格式化生成的 TypeScript 代码。
func formatWithPrettier(file *generator.GeneratedFile) error {
	ext := strings.ToLower(filepath.Ext(file.Path))
	if ext != ".ts" && ext != ".tsx" {
		return nil
	}

	var stdout, stderr bytes.Buffer
	//nolint:gosec,noctx // Prettier 路径由用户显式配置，子进程生命周期与生成器一致。
	cmd := exec.Command(*prettier, "--stdin-filepath", file.Path)
	cmd.Stdin = bytes.NewReader(file.Content)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return fmt.Errorf("format %s with prettier: %w", file.Path, err)
		}
		return fmt.Errorf("format %s with prettier: %w: %s", file.Path, err, message)
	}
	file.Content = stdout.Bytes()
	return nil
}
