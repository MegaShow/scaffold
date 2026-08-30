package generator

import (
	"errors"
	"fmt"
	"go/format"
	"slices"
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
	data, err := toTmplGoMessageFile(file)
	if err != nil {
		return nil, err
	}
	content, err := template.Render("go_message.go.tpl", data)
	if err != nil {
		return nil, err
	}
	formatted, fmtErr := format.Source(content)
	if fmtErr != nil {
		// 格式化失败时打印原始模板输出便于排查。
		return nil, fmt.Errorf("generator: message: format source: %w\n==== raw template output ====\n%s\n==== end ====", fmtErr, content)
	}
	return &GeneratedFile{
		Path:    file.OutputPrefix + ".go",
		Content: formatted,
	}, nil
}

func (g *goServerGenerator) generateServiceFile(file *File) (*GeneratedFile, error) {
	data, err := toTmplGoServiceFile(file)
	if err != nil {
		return nil, err
	}
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
	Package  string           // 包名
	Source   string           // 源文件路径
	Versions []string         // 版本注释行
	Imports  [][]string       // 分组排序后的导入路径列表，标准库在前，组间以空行分隔
	Messages []*tmplGoMessage // 消息类型列表
	Enums    []*Enum          // 枚举类型列表
}

type tmplGoMessage struct {
	*Message
	PathFields    []*Field // 被任一路由使用的路径字段，按消息字段顺序排列
	usedInRequest bool     // 是否在请求类型中使用
}

// IsUsedInRequest 判断消息是否被当前文件中的请求类型直接或间接使用。
func (m *tmplGoMessage) IsUsedInRequest() bool {
	return m.usedInRequest
}

type tmplGoServiceFile struct {
	Package  string           // 包名
	Source   string           // 源文件路径
	Versions []string         // 版本注释行
	Imports  [][]string       // 分组排序后的导入路径列表，标准库在前，组间以空行分隔
	Services []*tmplGoService // 服务列表
}

type tmplGoService struct {
	*Service
	Methods []*tmplGoMethod // 带路径绑定信息的方法列表
}

type tmplGoMethod struct {
	*Method
	PathParams []string // 按路由顺序排列的路径参数名
}

func toTmplGoMessageFile(file *File) (*tmplGoMessageFile, error) {
	usedInRequest, requestImports := collectRequestUsage(file)
	pathFields, pathImports, err := collectGoPathFields(file)
	if err != nil {
		return nil, err
	}
	requestImports = append(requestImports, pathImports...)
	messages := make([]*tmplGoMessage, 0, len(file.Messages))
	for _, message := range file.Messages {
		messages = append(messages, &tmplGoMessage{
			Message:       message,
			PathFields:    pathFields[message.Name],
			usedInRequest: usedInRequest[message.Name],
		})
	}
	// 按消息名字典序生成类型及其方法，保留消息内部字段和输入模型的顺序。
	slices.SortFunc(messages, func(a, b *tmplGoMessage) int {
		return strings.Compare(a.Name, b.Name)
	})
	return &tmplGoMessageFile{
		Package:  file.Package,
		Source:   file.Path,
		Versions: versionLines(file),
		Imports:  groupGoImports(collectGoMessageImports(file, requestImports)),
		Messages: messages,
		Enums:    file.Enums,
	}, nil
}

// collectRequestUsage 收集请求直接或间接使用的消息，以及生成 DecodeForm 所需的导入。
func collectRequestUsage(file *File) (map[string]bool, []string) {
	messages := make(map[string]*Message, len(file.Messages))
	for _, message := range file.Messages {
		messages[message.Name] = message
	}

	used := make(map[string]bool, len(file.Messages))
	imports := map[string]struct{}{}
	var collectType func(*Type)
	collectType = func(t *Type) {
		if t == nil {
			return
		}
		switch t.Kind {
		case KindMap:
			imports["go.icytown.com/scaffold/mux/conv"] = struct{}{}
			collectType(t.MapKey)
			collectType(t.MapElem)
		case KindMessage:
			if t.ImportPath != "" {
				return
			}
			message := messages[t.Name]
			if message == nil || used[t.Name] {
				return
			}
			used[t.Name] = true
			imports["net/url"] = struct{}{}
			imports["go.icytown.com/scaffold/mux/form"] = struct{}{}
			for _, field := range message.Fields {
				collectType(field.Type)
			}
		}
	}

	for _, service := range file.Services {
		for _, method := range service.Methods {
			collectType(method.RequestType)
		}
	}
	importPaths := make([]string, 0, len(imports))
	for path := range imports {
		importPaths = append(importPaths, path)
	}
	return used, importPaths
}

// collectGoPathFields 汇总各请求被路由使用的字段，为复用请求生成按名称选择的路径解码。
func collectGoPathFields(file *File) (map[string][]*Field, []string, error) {
	used := make(map[string]map[string]bool)
	for _, service := range file.Services {
		for _, method := range service.Methods {
			params, err := collectGoPathParams(file, method)
			if err != nil {
				return nil, nil, fmt.Errorf("generator: service %s method %s: %w", service.Name, method.Name, err)
			}
			if len(params) == 0 {
				continue
			}
			name := method.RequestType.Name
			if used[name] == nil {
				used[name] = make(map[string]bool)
			}
			for _, param := range params {
				used[name][param] = true
			}
		}
	}
	fields := make(map[string][]*Field, len(used))
	var imports []string
	if len(used) > 0 {
		imports = append(imports, "net/http", "go.icytown.com/scaffold/mux/path")
	}
	for _, message := range file.Messages {
		for _, field := range message.Fields {
			if !used[message.Name][field.Name] {
				continue
			}
			fields[message.Name] = append(fields[message.Name], field)
		}
	}
	return fields, imports, nil
}

// collectGoPathParams 提取单段路径参数，并校验请求中存在可解码的同名字段。
func collectGoPathParams(file *File, method *Method) ([]string, error) {
	var params []string
	seen := make(map[string]bool)
	for segment := range strings.SplitSeq(method.Pattern, "/") {
		if !strings.ContainsAny(segment, "{}") {
			continue
		}
		name, ok := strings.CutPrefix(segment, "{")
		if !ok {
			return nil, fmt.Errorf("invalid path parameter %q", segment)
		}
		name, ok = strings.CutSuffix(name, "}")
		if !ok || name == "" || strings.ContainsAny(name, "{}.$") {
			return nil, fmt.Errorf("unsupported path parameter %q: only {name} is supported", segment)
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate path parameter %q", name)
		}
		seen[name] = true
		params = append(params, name)
	}
	if len(params) == 0 {
		return nil, nil
	}
	if isEmptyType(method.RequestType) {
		return nil, errors.New("path parameters are not allowed when request type is api.Empty")
	}
	if method.RequestType == nil || method.RequestType.Kind != KindMessage || method.RequestType.ImportPath != "" {
		return nil, errors.New("path parameters require a request message defined in the current file")
	}
	var request *Message
	for _, message := range file.Messages {
		if message.Name == method.RequestType.Name {
			request = message
			break
		}
	}
	if request == nil {
		return nil, fmt.Errorf("request message %q not found", method.RequestType.Name)
	}
	fields := make(map[string]*Field, len(request.Fields))
	for _, field := range request.Fields {
		fields[field.Name] = field
	}
	for _, name := range params {
		field := fields[name]
		if field == nil {
			return nil, fmt.Errorf("path parameter %q has no matching field in request %s", name, request.Name)
		}
		if field.Repeated || field.Type == nil {
			return nil, fmt.Errorf("path parameter %q must be a scalar or enum field", name)
		}
		switch field.Type.Kind {
		case KindBool, KindInt32, KindInt64, KindUint32, KindUint64, KindFloat32, KindFloat64, KindString, KindBytes, KindEnum:
		default:
			return nil, fmt.Errorf("path parameter %q must be a scalar or enum field", name)
		}
	}
	return params, nil
}

func toTmplGoServiceFile(file *File) (*tmplGoServiceFile, error) {
	services := make([]*tmplGoService, 0, len(file.Services))
	imports := collectGoServiceImports(file)
	for _, service := range file.Services {
		s := &tmplGoService{Service: service}
		for _, method := range service.Methods {
			params, err := collectGoPathParams(file, method)
			if err != nil {
				return nil, fmt.Errorf("generator: service %s method %s: %w", service.Name, method.Name, err)
			}
			s.Methods = append(s.Methods, &tmplGoMethod{Method: method, PathParams: params})
		}
		// 按方法名字典序生成接口、路由注册和处理函数，不改变输入模型的顺序。
		slices.SortFunc(s.Methods, func(a, b *tmplGoMethod) int {
			return strings.Compare(a.Name, b.Name)
		})
		services = append(services, s)
	}
	return &tmplGoServiceFile{
		Package:  file.Package,
		Source:   file.Path,
		Versions: versionLines(file),
		Imports:  groupGoImports(imports),
		Services: services,
	}, nil
}

// collectGoMessageImports 收集消息字段引用的跨包导入及 DecodeForm 的运行时依赖。
func collectGoMessageImports(file *File, requestImports []string) []string {
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
	for _, path := range requestImports {
		paths[path] = struct{}{}
	}
	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	return result
}

// collectGoServiceImports 收集 Service 接口需要的导入路径并排序。
func collectGoServiceImports(file *File) []string {
	paths := map[string]struct{}{
		"context":                     {},
		"net/http":                    {},
		"go.icytown.com/scaffold/mux": {},
	}
	for _, service := range file.Services {
		for _, method := range service.Methods {
			for _, t := range []*Type{method.RequestType, method.ResponseType} {
				if !isEmptyType(t) && t.ImportPath != "" {
					paths[t.ImportPath] = struct{}{}
				}
			}
		}
	}
	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	return result
}

// groupGoImports 把导入路径按标准库/非标准库分组，并按字典序排列各组。
func groupGoImports(paths []string) [][]string {
	var std, others []string
	for _, p := range paths {
		if isStdlibImport(p) {
			std = append(std, p)
		} else {
			others = append(others, p)
		}
	}
	sort.Strings(std)
	sort.Strings(others)
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
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}
