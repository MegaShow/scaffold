package generator

import (
	"fmt"
	"go/format"
	"sort"
	"strings"

	"go.icytown.com/scaffold/internal/template"
)

// goServerGenerator 生成 Golang 服务端脚手架代码。
type goServerGenerator struct{}

func (g *goServerGenerator) Generate(file *File) ([]*GeneratedFile, error) {
	if len(file.Services) == 0 && len(file.Messages) == 0 && len(file.Enums) == 0 {
		return nil, nil
	}

	var files []*GeneratedFile
	if len(file.Messages) > 0 {
		f, err := g.generateMessageFile(file)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if len(file.Services) > 0 {
		f, err := g.generateServiceFile(file)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func (g *goServerGenerator) generateMessageFile(file *File) (*GeneratedFile, error) {
	data := toTmplGoMessageFile(file)
	content, err := template.Render("go_message.go.tpl", data)
	if err != nil {
		return nil, err
	}
	content, err = format.Source(content)
	if err != nil {
		return nil, fmt.Errorf("generator: message: format source: %w", err)
	}
	return &GeneratedFile{
		Path:    file.OutputPrefix + ".go",
		Content: content,
	}, nil
}

func (g *goServerGenerator) generateServiceFile(file *File) (*GeneratedFile, error) {
	data := toTmplGoServiceFile(file)
	content, err := template.Render("go_service_server.go.tpl", data)
	if err != nil {
		return nil, err
	}
	content, err = format.Source(content)
	if err != nil {
		return nil, fmt.Errorf("generator: service: format source: %w", err)
	}
	return &GeneratedFile{
		Path:    file.OutputPrefix + ".service.go",
		Content: content,
	}, nil
}

type tmplGoMessageFile struct {
	Package  string     // 包名
	Source   string     // 源文件路径
	Versions []string   // 版本注释行
	Imports  [][]string // 分组排序后的导入路径列表，标准库在前，组间以空行分隔
	Messages []*Message // 消息类型列表
	Enums    []*Enum    // 枚举类型列表
}

type tmplGoServiceFile struct {
	Package  string     // 包名
	Source   string     // 源文件路径
	Versions []string   // 版本注释行
	Imports  [][]string // 分组排序后的导入路径列表，标准库在前，组间以空行分隔
	Services []*Service // 服务列表
}

func toTmplGoMessageFile(file *File) *tmplGoMessageFile {
	return &tmplGoMessageFile{
		Package:  file.Package,
		Source:   file.Path,
		Versions: versionLines(file),
		Imports:  groupGoImports(collectGoMessageImports(file)),
		Messages: file.Messages,
		Enums:    file.Enums,
	}
}

func toTmplGoServiceFile(file *File) *tmplGoServiceFile {
	return &tmplGoServiceFile{
		Package:  file.Package,
		Source:   file.Path,
		Versions: versionLines(file),
		Imports:  groupGoImports(collectGoServiceImports(file)),
		Services: file.Services,
	}
}

// collectGoMessageImports 收集消息字段引用的跨包导入路径并排序。
func collectGoMessageImports(file *File) []string {
	paths := map[string]struct{}{}
	var addType func(t *Type)
	addType = func(t *Type) {
		if t == nil {
			return
		}
		if t.ImportPath != "" {
			paths[t.ImportPath] = struct{}{}
		}
		addType(t.MapKey)
		addType(t.MapElem)
	}
	for _, msg := range file.Messages {
		for _, f := range msg.Fields {
			addType(f.Type)
		}
	}
	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

// collectGoServiceImports 收集 Service 接口需要的导入路径并排序。
func collectGoServiceImports(file *File) []string {
	paths := map[string]struct{}{
		"context":                     {},
		"go.icytown.com/scaffold/mux": {},
	}
	for _, service := range file.Services {
		for _, method := range service.Methods {
			for _, t := range []*Type{method.RequestType, method.ResponseType} {
				if t.ImportPath != "" {
					paths[t.ImportPath] = struct{}{}
				}
			}
		}
	}
	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

// groupGoImports 把排序后的导入路径按标准库/非标准库分为两组，组内保持原有顺序。
func groupGoImports(paths []string) [][]string {
	var std, others []string
	for _, p := range paths {
		if isStdlibImport(p) {
			std = append(std, p)
		} else {
			others = append(others, p)
		}
	}
	groups := make([][]string, 0, 2)
	if len(std) > 0 {
		groups = append(groups, std)
	}
	if len(others) > 0 {
		groups = append(groups, others)
	}
	return groups
}

// isStdlibImport 判断导入路径是否属于标准库：首段路径不含 "." 即视为标准库。
func isStdlibImport(path string) bool {
	first := path
	if idx := strings.IndexByte(path, '/'); idx >= 0 {
		first = path[:idx]
	}
	return !strings.Contains(first, ".")
}
