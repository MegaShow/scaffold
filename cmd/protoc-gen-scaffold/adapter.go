package main

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"go.icytown.com/scaffold/internal/gen/api"
	"go.icytown.com/scaffold/internal/generator"
)

// toFile 把 protobuf 文件描述转换为协议无关的中间模型，
// (api.http) 注解转换为方法的 HTTP 绑定，protoc 版本写入依赖列表。
func toFile(gen *protogen.Plugin, file *protogen.File) *generator.File {
	services := make([]*generator.Service, 0, len(file.Services))
	for _, svc := range file.Services {
		services = append(services, toService(gen, svc, file))
	}
	return &generator.File{
		Path:         file.Desc.Path(),
		OutputPrefix: file.GeneratedFilenamePrefix,
		Package:      string(file.GoPackageName),
		Services:     services,
		Messages:     collectMessages(gen, file),
		Enums:        collectEnums(file),
		Dependencies: []*generator.Dependency{
			{Name: "protoc", Version: protocVersion(gen)},
		},
	}
}

// toService 把 protobuf 服务描述转换为中间模型的服务。
func toService(gen *protogen.Plugin, svc *protogen.Service, file *protogen.File) *generator.Service {
	methods := make([]*generator.Method, 0, len(svc.Methods))
	for _, m := range svc.Methods {
		method := toMethod(gen, m, file)
		if method != nil {
			methods = append(methods, method)
		}
	}
	return &generator.Service{
		Name:    svc.GoName,
		Comment: commentOf(svc.Comments),
		Methods: methods,
	}
}

// toMethod 把 protobuf 方法描述转换为中间模型的方法，
// 读取 (api.http) 注解确定 HTTP 方法与路由模式。
func toMethod(gen *protogen.Plugin, m *protogen.Method, file *protogen.File) *generator.Method {
	httpMethod, pattern := parseHTTPRule(getHTTPRule(m))
	if httpMethod == "" && pattern == "" {
		return nil
	}
	return &generator.Method{
		Name:         m.GoName,
		Comment:      commentOf(m.Comments),
		HTTPMethod:   generator.HTTPMethod(httpMethod),
		Pattern:      pattern,
		RequestType:  toType(gen, m.Input, file),
		ResponseType: toType(gen, m.Output, file),
	}
}

// toType 把 protobuf 消息描述转换为中间模型的类型，
// 与生成代码同包的类型不携带导入路径。
func toType(gen *protogen.Plugin, msg *protogen.Message, file *protogen.File) *generator.Type {
	t := &generator.Type{
		Kind:         generator.KindMessage,
		Name:         msg.GoIdent.GoName,
		OutputPrefix: outputPrefixOf(gen, msg.Desc.ParentFile().Path()),
	}
	if msg.GoIdent.GoImportPath != file.GoImportPath {
		t.ImportPath = string(msg.GoIdent.GoImportPath)
	}
	return t
}

// collectMessages 递归收集文件内的消息定义（含嵌套消息），跳过 map entry 合成消息。
func collectMessages(gen *protogen.Plugin, file *protogen.File) []*generator.Message {
	var messages []*generator.Message
	var walk func(msgs []*protogen.Message)
	walk = func(msgs []*protogen.Message) {
		for _, m := range msgs {
			if m.Desc.IsMapEntry() {
				continue
			}
			fields := make([]*generator.Field, 0, len(m.Fields))
			for _, f := range m.Fields {
				fields = append(fields, toField(gen, f, file))
			}
			messages = append(messages, &generator.Message{
				Name:    m.GoIdent.GoName,
				Comment: commentOf(m.Comments),
				Fields:  fields,
			})
			walk(m.Messages)
		}
	}
	walk(file.Messages)
	return messages
}

// collectEnums 递归收集文件内的枚举定义（含消息内嵌套的枚举）。
func collectEnums(file *protogen.File) []*generator.Enum {
	enums := make([]*generator.Enum, 0, len(file.Enums))
	for _, e := range file.Enums {
		enums = append(enums, toEnum(e))
	}
	var walk func(msgs []*protogen.Message)
	walk = func(msgs []*protogen.Message) {
		for _, m := range msgs {
			for _, e := range m.Enums {
				enums = append(enums, toEnum(e))
			}
			walk(m.Messages)
		}
	}
	walk(file.Messages)
	return enums
}

// toEnum 把 protobuf 枚举描述转换为中间模型的枚举。
func toEnum(e *protogen.Enum) *generator.Enum {
	values := make([]*generator.EnumValue, 0, len(e.Values))
	for _, v := range e.Values {
		values = append(values, &generator.EnumValue{
			Name:    v.GoIdent.GoName,
			Comment: commentOf(v.Comments),
			Number:  int32(v.Desc.Number()),
		})
	}
	return &generator.Enum{Name: e.GoIdent.GoName, Comment: commentOf(e.Comments), Values: values}
}

// toField 把 protobuf 字段描述转换为中间模型的字段，字段名保留 IDL 中的原始名字。
func toField(gen *protogen.Plugin, field *protogen.Field, file *protogen.File) *generator.Field {
	return &generator.Field{
		Name:     string(field.Desc.Name()),
		Comment:  commentOf(field.Comments),
		Type:     toFieldType(gen, field, file),
		Repeated: field.Desc.IsList(),
	}
}

// commentOf 返回声明的备注文本：前导注释优先，缺失时回退到行尾注释。
func commentOf(set protogen.CommentSet) string {
	if set.Leading != "" {
		return strings.TrimSpace(string(set.Leading))
	}
	return strings.TrimSpace(string(set.Trailing))
}

// toFieldType 把 protobuf 字段类型转换为中间模型的字段类型。
func toFieldType(gen *protogen.Plugin, field *protogen.Field, file *protogen.File) *generator.Type {
	if field.Desc.IsMap() {
		// map 字段关联的合成 entry 消息：字段 1 为 key、字段 2 为 value。
		entry := field.Message
		var keyField, valField *protogen.Field
		for _, f := range entry.Fields {
			switch f.Desc.Number() {
			case 1:
				keyField = f
			case 2:
				valField = f
			}
		}
		return &generator.Type{
			Kind:    generator.KindMap,
			MapKey:  toFieldType(gen, keyField, file),
			MapElem: toFieldType(gen, valField, file),
		}
	}

	switch field.Desc.Kind() {
	case protoreflect.MessageKind:
		return namedFieldType(gen, field.Message.GoIdent, generator.KindMessage, field.Message.Desc.ParentFile().Path(), file)
	case protoreflect.EnumKind:
		return namedFieldType(gen, field.Enum.GoIdent, generator.KindEnum, field.Enum.Desc.ParentFile().Path(), file)
	default:
		return &generator.Type{Kind: scalarKindMap[field.Desc.Kind()]}
	}
}

// namedFieldType 构造引用命名类型的字段类型，与生成代码同包时不携带导入路径。
func namedFieldType(
	gen *protogen.Plugin,
	id protogen.GoIdent,
	kind generator.FieldKind,
	sourcePath string,
	file *protogen.File,
) *generator.Type {
	t := &generator.Type{Kind: kind, Name: id.GoName, OutputPrefix: outputPrefixOf(gen, sourcePath)}
	if id.GoImportPath != file.GoImportPath {
		t.ImportPath = string(id.GoImportPath)
	}
	return t
}

// outputPrefixOf 返回声明文件对应的生成文件前缀。
func outputPrefixOf(gen *protogen.Plugin, sourcePath string) string {
	file := gen.FilesByPath[sourcePath]
	if file == nil {
		return ""
	}
	return file.GeneratedFilenamePrefix
}

// scalarKindMap 把 protobuf 标量类型映射为中间模型类型种类。
var scalarKindMap = map[protoreflect.Kind]generator.FieldKind{
	protoreflect.DoubleKind:   generator.KindFloat64,
	protoreflect.FloatKind:    generator.KindFloat32,
	protoreflect.Int64Kind:    generator.KindInt64,
	protoreflect.Sint64Kind:   generator.KindInt64,
	protoreflect.Sfixed64Kind: generator.KindInt64,
	protoreflect.Uint64Kind:   generator.KindUint64,
	protoreflect.Fixed64Kind:  generator.KindUint64,
	protoreflect.Int32Kind:    generator.KindInt32,
	protoreflect.Sint32Kind:   generator.KindInt32,
	protoreflect.Sfixed32Kind: generator.KindInt32,
	protoreflect.Uint32Kind:   generator.KindUint32,
	protoreflect.Fixed32Kind:  generator.KindUint32,
	protoreflect.BoolKind:     generator.KindBool,
	protoreflect.StringKind:   generator.KindString,
	protoreflect.BytesKind:    generator.KindBytes,
}

// protocVersion 返回发起本次生成的 protoc 版本。
func protocVersion(gen *protogen.Plugin) string {
	v := gen.Request.GetCompilerVersion()
	if v == nil {
		return "(unknown)"
	}
	version := fmt.Sprintf("v%d.%d.%d", v.GetMajor(), v.GetMinor(), v.GetPatch())
	if suffix := v.GetSuffix(); suffix != "" {
		version += "-" + suffix
	}
	return version
}

// getHTTPRule 读取方法上的 (api.http) 注解，未声明时返回 nil。
func getHTTPRule(method *protogen.Method) *api.HttpRule {
	opts, ok := method.Desc.Options().(*descriptorpb.MethodOptions)
	if !ok || opts == nil {
		return nil
	}
	if !proto.HasExtension(opts, api.E_Http) {
		return nil
	}
	rule, ok := proto.GetExtension(opts, api.E_Http).(*api.HttpRule)
	if !ok || rule == nil {
		return nil
	}
	return rule
}

// parseHTTPRule 解析注解中的 HTTP 方法与路由模式，未声明时默认 POST /。
func parseHTTPRule(rule *api.HttpRule) (string, string) {
	if rule == nil {
		return "", ""
	}
	if v := rule.GetGet(); v != "" {
		return "GET", v
	}
	if v := rule.GetPost(); v != "" {
		return "POST", v
	}
	if v := rule.GetPut(); v != "" {
		return "PUT", v
	}
	if v := rule.GetDelete(); v != "" {
		return "DELETE", v
	}
	if v := rule.GetPatch(); v != "" {
		return "PATCH", v
	}
	return "", ""
}
