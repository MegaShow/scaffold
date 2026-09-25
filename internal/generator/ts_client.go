package generator

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"go.icytown.com/scaffold/internal/template"
)

// tsClientGenerator 生成 TypeScript 客户端脚手架代码。
type tsClientGenerator struct{}

func (g *tsClientGenerator) Generate(file *File) ([]*GeneratedFile, error) {
	if len(file.Messages) == 0 && len(file.Enums) == 0 && len(file.Services) == 0 {
		return nil, nil
	}

	var files []*GeneratedFile
	if len(file.Messages) > 0 || len(file.Enums) > 0 {
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

// generateMessageFile 生成 TypeScript 类型声明文件，对应 ts_message.go.tpl。
func (g *tsClientGenerator) generateMessageFile(file *File) (*GeneratedFile, error) {
	data, err := toTmplTSMessageFile(file)
	if err != nil {
		return nil, err
	}
	content, err := template.Render("ts_message.go.tpl", data)
	if err != nil {
		return nil, err
	}
	return &GeneratedFile{
		Path:    file.OutputPrefix + ".d.ts",
		Content: content,
	}, nil
}

// generateServiceFile 生成 TypeScript 服务客户端文件。
func (g *tsClientGenerator) generateServiceFile(file *File) (*GeneratedFile, error) {
	data, err := toTmplTSServiceFile(file)
	if err != nil {
		return nil, err
	}
	content, err := template.Render("ts_service_client.go.tpl", data)
	if err != nil {
		return nil, err
	}
	return &GeneratedFile{
		Path:    file.OutputPrefix + ".service.ts",
		Content: content,
	}, nil
}

// tmplTSMessageFile 是 TypeScript 类型定义文件的模板数据，对应 tmplGoMessageFile。
type tmplTSMessageFile struct {
	Source   string          // 源文件路径
	Versions []string        // 版本注释行
	Imports  []*tmplTSImport // TypeScript 类型导入，按模块路径排列
	Messages []*Message      // 消息类型列表，按名字字典序排列
	Enums    []*Enum         // 枚举类型列表，按名字字典序排列
}

func toTmplTSMessageFile(file *File) (*tmplTSMessageFile, error) {
	messages := make([]*Message, len(file.Messages))
	copy(messages, file.Messages)
	slices.SortFunc(messages, func(a, b *Message) int {
		return strings.Compare(a.Name, b.Name)
	})

	enums := make([]*Enum, len(file.Enums))
	copy(enums, file.Enums)
	slices.SortFunc(enums, func(a, b *Enum) int {
		return strings.Compare(a.Name, b.Name)
	})

	var types []*Type
	for _, message := range file.Messages {
		for _, field := range message.Fields {
			types = append(types, field.Type)
		}
	}
	imports, err := collectTSImports(file.OutputPrefix, false, types...)
	if err != nil {
		return nil, err
	}

	return &tmplTSMessageFile{
		Source:   file.Path,
		Versions: versionLines(file),
		Imports:  imports,
		Messages: messages,
		Enums:    enums,
	}, nil
}

// tmplTSServiceFile 是 TypeScript 服务客户端文件的模板数据。
type tmplTSServiceFile struct {
	Source   string           // 源文件路径
	Versions []string         // 版本注释行
	Imports  []*tmplTSImport  // TypeScript 类型导入，导入名字按字典序排列
	Services []*tmplTSService // 服务列表，按名字字典序排列
}

// tmplTSImport 描述一条 TypeScript 类型导入。
type tmplTSImport struct {
	Path  string   // 模块路径
	Names []string // 导入名字列表
}

// tmplTSService 是 TypeScript 服务客户端的模板数据。
type tmplTSService struct {
	*Service
	Methods []*tmplTSMethod // 方法列表，按名字字典序排列
}

// tmplTSMethod 是 TypeScript 服务方法的模板数据。
type tmplTSMethod struct {
	*Method
	PathParams []string // 路径参数名
}

func toTmplTSServiceFile(file *File) (*tmplTSServiceFile, error) {
	services := make([]*tmplTSService, 0, len(file.Services))
	var types []*Type
	for _, service := range file.Services {
		tmplService := &tmplTSService{Service: service}
		for _, method := range service.Methods {
			params, err := collectPathParams(file, method)
			if err != nil {
				return nil, fmt.Errorf("generator: service %s method %s: %w", service.Name, method.Name, err)
			}
			types = append(types, method.RequestType, method.ResponseType)
			tmplService.Methods = append(tmplService.Methods, &tmplTSMethod{
				Method:     method,
				PathParams: params,
			})
		}
		slices.SortFunc(tmplService.Methods, func(a, b *tmplTSMethod) int {
			return strings.Compare(toLowerCamelCase(a.Name), toLowerCamelCase(b.Name))
		})
		services = append(services, tmplService)
	}
	slices.SortFunc(services, func(a, b *tmplTSService) int {
		return strings.Compare(a.Name, b.Name)
	})

	typeImports, err := collectTSImports(file.OutputPrefix, true, types...)
	if err != nil {
		return nil, err
	}
	imports := make([]*tmplTSImport, 0, 1+len(typeImports))
	imports = append(imports, &tmplTSImport{
		Path:  "@icytown/scaffold-client",
		Names: []string{"Client"},
	})
	imports = append(imports, typeImports...)
	slices.SortFunc(imports, func(a, b *tmplTSImport) int {
		return strings.Compare(a.Path, b.Path)
	})

	return &tmplTSServiceFile{
		Source:   file.Path,
		Versions: versionLines(file),
		Imports:  imports,
		Services: services,
	}, nil
}

// collectTSImports 收集命名类型引用，并按目标模块合并、排序 TypeScript 类型导入。
func collectTSImports(currentPrefix string, includeCurrent bool, types ...*Type) ([]*tmplTSImport, error) {
	importPaths := make(map[string]string)
	namesByPath := make(map[string]map[string]struct{})
	var collect func(*Type) error
	collect = func(t *Type) error {
		if t == nil || isEmptyType(t) {
			return nil
		}
		if t.Kind == KindMap {
			if err := collect(t.MapKey); err != nil {
				return err
			}
			return collect(t.MapElem)
		}
		if t.Kind != KindMessage && t.Kind != KindEnum {
			return nil
		}
		targetPrefix := t.OutputPrefix
		if targetPrefix == "" {
			targetPrefix = currentPrefix
		}
		if !includeCurrent && targetPrefix == currentPrefix {
			return nil
		}
		importPath, ok := importPaths[targetPrefix]
		if !ok {
			var err error
			importPath, err = relativeTSImportPath(currentPrefix, targetPrefix)
			if err != nil {
				return err
			}
			importPaths[targetPrefix] = importPath
		}
		if namesByPath[importPath] == nil {
			namesByPath[importPath] = make(map[string]struct{})
		}
		namesByPath[importPath][t.Name] = struct{}{}
		return nil
	}
	for _, t := range types {
		if err := collect(t); err != nil {
			return nil, err
		}
	}

	imports := make([]*tmplTSImport, 0, len(namesByPath))
	for importPath, nameSet := range namesByPath {
		names := make([]string, 0, len(nameSet))
		for name := range nameSet {
			names = append(names, name)
		}
		slices.Sort(names)
		imports = append(imports, &tmplTSImport{Path: importPath, Names: names})
	}
	slices.SortFunc(imports, func(a, b *tmplTSImport) int {
		return strings.Compare(a.Path, b.Path)
	})
	return imports, nil
}

// relativeTSImportPath 计算生成文件之间的 TypeScript ESM 相对导入路径。
func relativeTSImportPath(currentPrefix, targetPrefix string) (string, error) {
	relative, err := filepath.Rel(filepath.Dir(currentPrefix), targetPrefix)
	if err != nil {
		return "", fmt.Errorf("generator: typescript import from %q to %q: %w", currentPrefix, targetPrefix, err)
	}
	relative = filepath.ToSlash(relative)
	if !strings.HasPrefix(relative, ".") {
		relative = "./" + relative
	}
	return relative, nil
}
