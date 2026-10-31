package main

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/pluginpb"

	"go.icytown.com/scaffold/internal/assert"
	"go.icytown.com/scaffold/internal/gen/api"
	"go.icytown.com/scaffold/internal/generator"
)

func Test_methodOptionExtensions(t *testing.T) {
	t.Parallel()
	gen := &protogen.Plugin{
		Files: []*protogen.File{{
			Extensions: []*protogen.Extension{{Desc: api.E_Http.TypeDescriptor()}},
		}},
	}
	testCases := []struct {
		name  string
		names []string
		want  int
	}{
		{name: "selected", names: []string{"api.http"}, want: 1},
		{name: "unselected", names: []string{"dev.access"}, want: 0},
	}
	for _, tt := range testCases {
		extensions, err := methodOptionExtensions(gen, tt.names)
		assert.Nil(t, err, tt.name)
		assert.Equal(t, tt.want, extensions.NumExtensions(), tt.name)
	}
}

func Test_validateFiles(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"single", "repeated", "nested", "map", "oneof", "get", "response", "direct", "recursive", "imported", "recursive_without_file", "shared_request", "shared_get", "shared_response"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			field := &descriptorpb.FieldDescriptorProto{
				Name: new("archive"), Number: new(int32(1)),
				Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: new(".api.File"),
				Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			}
			request := &descriptorpb.DescriptorProto{Name: new("Request"), Field: []*descriptorpb.FieldDescriptorProto{field}}
			method := &descriptorpb.MethodDescriptorProto{
				Name: new("Upload"), InputType: new(".test.Request"), OutputType: new(".api.Empty"),
				Options: &descriptorpb.MethodOptions{},
			}
			rule := new(api.HttpRule)
			rule.SetPost("/upload")
			file := &descriptorpb.FileDescriptorProto{
				Name: new("test.proto"), Package: new("test"), Syntax: new("proto3"),
				Options:    &descriptorpb.FileOptions{GoPackage: new("example.com/test")},
				Dependency: []string{"scaffold/api.proto"}, MessageType: []*descriptorpb.DescriptorProto{request},
				Service: []*descriptorpb.ServiceDescriptorProto{{Name: new("Service"), Method: []*descriptorpb.MethodDescriptorProto{method}}},
			}
			switch name {
			case "repeated":
				field.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
			case "nested":
				request.Name = new("Nested")
				file.MessageType = append(file.MessageType, &descriptorpb.DescriptorProto{Name: new("Request"), Field: []*descriptorpb.FieldDescriptorProto{{
					Name: new("nested"), Number: new(int32(1)), Type: field.Type, TypeName: new(".test.Nested"), Label: field.Label,
				}}})
			case "map":
				request.NestedType = []*descriptorpb.DescriptorProto{{Name: new("FilesEntry"), Options: &descriptorpb.MessageOptions{MapEntry: new(true)}, Field: []*descriptorpb.FieldDescriptorProto{
					{Name: new("key"), Number: new(int32(1)), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: field.Label},
					{Name: new("value"), Number: new(int32(2)), Type: field.Type, TypeName: field.TypeName, Label: field.Label},
				}}}
				field.Name = new("files")
				field.TypeName = new(".test.Request.FilesEntry")
				field.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
			case "oneof":
				request.OneofDecl = []*descriptorpb.OneofDescriptorProto{{Name: new("content")}}
				field.OneofIndex = new(int32(0))
			case "get":
				rule = new(api.HttpRule)
				rule.SetGet("/upload")
			case "response":
				method.OutputType = method.InputType
			case "direct":
				method.InputType = new(".api.File")
			case "recursive":
				request.Field = append(request.Field, &descriptorpb.FieldDescriptorProto{
					Name: new("child"), Number: new(int32(2)), Type: field.Type, TypeName: method.InputType, Label: field.Label,
				})
			case "recursive_without_file":
				field.TypeName = method.InputType
			}
			proto.SetExtension(method.Options, api.E_Http, rule)
			if name == "shared_request" || name == "shared_get" || name == "shared_response" {
				shared := &descriptorpb.MethodDescriptorProto{
					Name: new("Reuse"), InputType: method.InputType, OutputType: method.OutputType,
					Options: &descriptorpb.MethodOptions{},
				}
				proto.SetExtension(shared.Options, api.E_Http, rule)
				switch name {
				case "shared_get":
					getRule := new(api.HttpRule)
					getRule.SetGet("/reuse")
					proto.SetExtension(shared.Options, api.E_Http, getRule)
				case "shared_response":
					shared.OutputType = shared.InputType
				}
				file.Service[0].Method = append(file.Service[0].Method, shared)
			}
			var files []*descriptorpb.FileDescriptorProto
			if name == "imported" {
				files = append(files, &descriptorpb.FileDescriptorProto{
					Name: new("shared.proto"), Package: file.Package, Syntax: file.Syntax, Options: file.Options,
					Dependency: file.Dependency, MessageType: file.MessageType,
				})
				file.MessageType = nil
				file.Dependency = append(file.Dependency, "shared.proto")
			}
			files = append(files, file)
			plugin := newAdapterTestPlugin(t, files...)
			err := validateFiles(plugin.FilesByPath["test.proto"])
			switch name {
			case "single", "repeated", "recursive_without_file", "shared_request":
				assert.Nil(t, err)
			default:
				if !assert.NotNil(t, err) {
					t.FailNow()
				}
				if name == "imported" {
					assert.True(t, strings.Contains(err.Error(), "request with file fields must be declared in the service file"))
				} else {
					assert.True(t, strings.Contains(err.Error(), "top-level request field"))
				}
			}
		})
	}
}

func Test_toService(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"empty", "unannotated", "mixed"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			desc := adapterTestFile()
			switch name {
			case "empty":
				desc.Service[0].Method = nil
			case "unannotated":
				desc.Service[0].Method = desc.Service[0].Method[:1]
			}
			plugin := newAdapterTestPlugin(t, desc)
			file := plugin.FilesByPath["test.proto"]
			request := collectMessages(plugin, file)[0]
			service, err := toService(plugin, file.Services[0], file, nil, map[string]*generator.Message{"Request": request})
			assert.Nil(t, err)
			if name != "mixed" {
				assert.Nil(t, service)
				return
			}
			if !assert.NotNil(t, service) {
				t.FailNow()
			}
			assert.Equal(t, "Service", service.Name)
			if assert.Equal(t, 1, len(service.Methods)) {
				assert.Equal(t, "Call", service.Methods[0].Name)
				assert.True(t, request == service.Methods[0].RequestMessage)
			}
		})
	}
}

func Test_toMethod(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"unannotated", "empty_binding", "post", "metadata", "unmatched_metadata", "external_request", "external_response"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			desc := adapterTestFile()
			switch name {
			case "empty_binding":
				proto.SetExtension(desc.Service[0].Method[1].Options, api.E_Http, new(api.HttpRule))
			case "external_request":
				desc.Service[0].Method[1].InputType = new(".api.Empty")
			case "external_response":
				desc.Service[0].Method[1].OutputType = new(".api.Empty")
			}
			plugin := newAdapterTestPlugin(t, desc)
			file := plugin.FilesByPath["test.proto"]
			method := file.Services[0].Methods[1]
			var extensions *protoregistry.Types
			switch name {
			case "unannotated":
				method = file.Services[0].Methods[0]
			case "metadata", "unmatched_metadata":
				optionName := "api.http"
				if name == "unmatched_metadata" {
					optionName = "test.missing"
				}
				var err error
				extensions, err = methodOptionExtensions(plugin, []string{optionName})
				if !assert.Nil(t, err) {
					t.FailNow()
				}
			}
			request := collectMessages(plugin, file)[0]
			response := &generator.Message{Name: "Child"}
			model, err := toMethod(plugin, method, file, extensions, map[string]*generator.Message{
				"Request": request, "Child": response, "Empty": request,
			})
			assert.Nil(t, err)
			if name == "unannotated" || name == "empty_binding" {
				assert.Nil(t, model)
				return
			}
			if !assert.NotNil(t, model) {
				t.FailNow()
			}
			assert.Equal(t, "Call", model.Name)
			assert.Equal(t, generator.HTTPMethod("POST"), model.HTTPMethod)
			assert.Equal(t, "/call", model.Pattern)
			if name == "external_request" {
				assert.Equal(t, "Empty", model.RequestType.Name)
				assert.Nil(t, model.RequestMessage)
			} else {
				assert.Equal(t, "Request", model.RequestType.Name)
				assert.True(t, request == model.RequestMessage)
			}
			if name == "external_response" {
				assert.Equal(t, "Empty", model.ResponseType.Name)
				assert.Nil(t, model.ResponseMessage)
			} else {
				assert.Equal(t, "Child", model.ResponseType.Name)
				assert.True(t, response == model.ResponseMessage)
			}
			if name == "metadata" {
				assert.Equal(t, map[string]map[string]any{"api.http": {"post": "/call"}}, model.Options)
			} else {
				assert.Nil(t, model.Options)
			}
		})
	}
}

func Test_toFieldType(t *testing.T) {
	t.Parallel()
	scalars := []struct {
		name   string
		typeOf descriptorpb.FieldDescriptorProto_Type
		want   generator.FieldKind
	}{
		{name: "double", typeOf: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, want: generator.KindFloat64},
		{name: "float", typeOf: descriptorpb.FieldDescriptorProto_TYPE_FLOAT, want: generator.KindFloat32},
		{name: "int64", typeOf: descriptorpb.FieldDescriptorProto_TYPE_INT64, want: generator.KindInt64},
		{name: "sint64", typeOf: descriptorpb.FieldDescriptorProto_TYPE_SINT64, want: generator.KindInt64},
		{name: "sfixed64", typeOf: descriptorpb.FieldDescriptorProto_TYPE_SFIXED64, want: generator.KindInt64},
		{name: "uint64", typeOf: descriptorpb.FieldDescriptorProto_TYPE_UINT64, want: generator.KindUint64},
		{name: "fixed64", typeOf: descriptorpb.FieldDescriptorProto_TYPE_FIXED64, want: generator.KindUint64},
		{name: "int32", typeOf: descriptorpb.FieldDescriptorProto_TYPE_INT32, want: generator.KindInt32},
		{name: "sint32", typeOf: descriptorpb.FieldDescriptorProto_TYPE_SINT32, want: generator.KindInt32},
		{name: "sfixed32", typeOf: descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, want: generator.KindInt32},
		{name: "uint32", typeOf: descriptorpb.FieldDescriptorProto_TYPE_UINT32, want: generator.KindUint32},
		{name: "fixed32", typeOf: descriptorpb.FieldDescriptorProto_TYPE_FIXED32, want: generator.KindUint32},
		{name: "bool", typeOf: descriptorpb.FieldDescriptorProto_TYPE_BOOL, want: generator.KindBool},
		{name: "string", typeOf: descriptorpb.FieldDescriptorProto_TYPE_STRING, want: generator.KindString},
		{name: "bytes", typeOf: descriptorpb.FieldDescriptorProto_TYPE_BYTES, want: generator.KindBytes},
	}
	desc := adapterTestFile()
	desc.MessageType[1].Field = nil
	for i, scalar := range scalars {
		desc.MessageType[1].Field = append(desc.MessageType[1].Field, &descriptorpb.FieldDescriptorProto{
			Name: new(scalar.name), Number: new(int32(i + 1)), Type: scalar.typeOf.Enum(),
		})
	}
	plugin := newAdapterTestPlugin(t, desc)
	file := plugin.FilesByPath["test.proto"]
	for i, scalar := range scalars {
		t.Run(scalar.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, &generator.Type{Kind: scalar.want}, toFieldType(plugin, file.Messages[1].Fields[i], file))
		})
	}
	want := map[string]*generator.Type{
		"name":  {Kind: generator.KindString},
		"count": {Kind: generator.KindInt32},
		"tags":  {Kind: generator.KindString},
		"state": {Kind: generator.KindEnum, Name: "State", OutputPrefix: file.GeneratedFilenamePrefix},
		"child": {Kind: generator.KindMessage, Name: "Child", OutputPrefix: file.GeneratedFilenamePrefix},
		"file":  {Kind: generator.KindFile},
		"files": {Kind: generator.KindFile},
		"labels": {Kind: generator.KindMap,
			MapKey: &generator.Type{Kind: generator.KindString}, MapElem: &generator.Type{Kind: generator.KindString}},
	}
	for _, field := range file.Messages[0].Fields {
		t.Run(string(field.Desc.Name()), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, want[string(field.Desc.Name())], toFieldType(plugin, field, file))
		})
	}
}

func Test_toField(t *testing.T) {
	t.Parallel()
	plugin := newAdapterTestPlugin(t, adapterTestFile())
	file := plugin.FilesByPath["test.proto"]
	for _, field := range file.Messages[0].Fields {
		t.Run(string(field.Desc.Name()), func(t *testing.T) {
			t.Parallel()
			field.Comments.Leading = " field comment "
			model := toField(plugin, field, file)
			assert.Equal(t, string(field.Desc.Name()), model.Name)
			assert.Equal(t, "field comment", model.Comment)
			assert.Equal(t, field.Desc.Name() == "tags" || field.Desc.Name() == "files", model.Repeated)
		})
	}
}

func Test_toType(t *testing.T) {
	t.Parallel()
	plugin := newAdapterTestPlugin(t, adapterTestFile())
	file := plugin.FilesByPath["test.proto"]
	apiFile := plugin.FilesByPath["scaffold/api.proto"]
	testCases := []struct {
		name string
		msg  *protogen.Message
		want *generator.Type
	}{
		{name: "local", msg: file.Messages[0], want: &generator.Type{Kind: generator.KindMessage, Name: "Request", OutputPrefix: file.GeneratedFilenamePrefix}},
		{name: "imported", msg: apiFile.Messages[0], want: &generator.Type{Kind: generator.KindMessage, Name: "Empty", ImportPath: string(apiFile.GoImportPath), OutputPrefix: apiFile.GeneratedFilenamePrefix}},
		{name: "file", msg: file.Messages[0].Fields[5].Message, want: &generator.Type{Kind: generator.KindFile}},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, toType(plugin, tt.msg, file))
		})
	}
}

func Test_hasFileFields(t *testing.T) {
	t.Parallel()
	plugin := newAdapterTestPlugin(t, adapterTestFile())
	file := plugin.FilesByPath["test.proto"]
	assert.True(t, hasFileFields(file.Messages[0]))
	assert.False(t, hasFileFields(file.Messages[1]))
	assert.False(t, hasFileFields(&protogen.Message{Fields: []*protogen.Field{file.Messages[0].Fields[4]}}))
}

func Test_isFile(t *testing.T) {
	t.Parallel()
	plugin := newAdapterTestPlugin(t, adapterTestFile())
	file := plugin.FilesByPath["test.proto"]
	assert.False(t, isFile(nil))
	assert.False(t, isFile(file.Messages[0]))
	assert.True(t, isFile(file.Messages[0].Fields[5].Message))
	lookalike, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name: new("other.proto"), Package: new("api"), Syntax: new("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{Name: new("File")}},
	}, nil)
	if !assert.Nil(t, err) {
		t.FailNow()
	}
	assert.False(t, isFile(&protogen.Message{Desc: lookalike.Messages().Get(0)}))
}

func Test_parseHTTPRule(t *testing.T) {
	t.Parallel()
	for _, method := range []string{"nil", "empty", "GET", "POST", "PUT", "DELETE", "PATCH"} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			rule := new(api.HttpRule)
			switch method {
			case "nil":
				rule = nil
			case "GET":
				rule.SetGet("/test")
			case "POST":
				rule.SetPost("/test")
			case "PUT":
				rule.SetPut("/test")
			case "DELETE":
				rule.SetDelete("/test")
			case "PATCH":
				rule.SetPatch("/test")
			}
			gotMethod, gotPattern := parseHTTPRule(rule)
			if method == "nil" || method == "empty" {
				assert.Equal(t, "", gotMethod)
				assert.Equal(t, "", gotPattern)
			} else {
				assert.Equal(t, method, gotMethod)
				assert.Equal(t, "/test", gotPattern)
			}
		})
	}
}

func Test_getHTTPRule(t *testing.T) {
	t.Parallel()
	plugin := newAdapterTestPlugin(t, adapterTestFile())
	methods := plugin.FilesByPath["test.proto"].Services[0].Methods
	assert.Nil(t, getHTTPRule(methods[0]))
	rule := getHTTPRule(methods[1])
	if assert.NotNil(t, rule) {
		assert.Equal(t, "/call", rule.GetPost())
	}
}

func Test_methodOptions(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"selected", "unselected", "no_options"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			plugin := newAdapterTestPlugin(t, adapterTestFile())
			optionName := "api.http"
			if name == "unselected" {
				optionName = "test.missing"
			}
			extensions, err := methodOptionExtensions(plugin, []string{optionName})
			if !assert.Nil(t, err) {
				t.FailNow()
			}
			methods := plugin.FilesByPath["test.proto"].Services[0].Methods
			method := methods[1]
			if name == "no_options" {
				method = methods[0]
			}
			options, err := methodOptions(method, extensions)
			assert.Nil(t, err)
			if name == "selected" {
				assert.Equal(t, map[string]map[string]any{"api.http": {"post": "/call"}}, options)
			} else {
				assert.Nil(t, options)
			}
		})
	}
}

func Test_commentOf(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		set  protogen.CommentSet
		want string
	}{
		{name: "empty"},
		{name: "leading", set: protogen.CommentSet{Leading: " leading \n", Trailing: " trailing "}, want: "leading"},
		{name: "trailing", set: protogen.CommentSet{Trailing: " trailing \n"}, want: "trailing"},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, commentOf(tt.set))
		})
	}
}

func Test_collectMessages(t *testing.T) {
	t.Parallel()
	plugin := newAdapterTestPlugin(t, adapterTestFile())
	messages := collectMessages(plugin, plugin.FilesByPath["test.proto"])
	names := make([]string, 0, len(messages))
	for _, message := range messages {
		names = append(names, message.Name)
	}
	assert.Equal(t, []string{"Request", "Request_Inner", "Child"}, names)
}

func Test_collectEnums(t *testing.T) {
	t.Parallel()
	plugin := newAdapterTestPlugin(t, adapterTestFile())
	enums := collectEnums(plugin.FilesByPath["test.proto"])
	names := make([]string, 0, len(enums))
	for _, enum := range enums {
		names = append(names, enum.Name)
	}
	assert.Equal(t, []string{"State", "Request_Mode"}, names)
}

func Test_toEnum(t *testing.T) {
	t.Parallel()
	plugin := newAdapterTestPlugin(t, adapterTestFile())
	enum := plugin.FilesByPath["test.proto"].Enums[0]
	enum.Comments.Leading = " enum comment "
	enum.Values[1].Comments.Leading = " value comment "
	assert.Equal(t, &generator.Enum{
		Name: "State", Comment: "enum comment",
		Values: []*generator.EnumValue{{Name: "State_UNKNOWN", Number: 0}, {Name: "State_READY", Number: 1, Comment: "value comment"}},
	}, toEnum(enum))
}

func Test_namedFieldType(t *testing.T) {
	t.Parallel()
	plugin := newAdapterTestPlugin(t, adapterTestFile())
	file := plugin.FilesByPath["test.proto"]
	for _, name := range []string{"same_package", "different_package", "unknown_file"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			id := protogen.GoIdent{GoName: "State", GoImportPath: file.GoImportPath}
			path := "test.proto"
			want := &generator.Type{Kind: generator.KindEnum, Name: "State", OutputPrefix: file.GeneratedFilenamePrefix}
			switch name {
			case "different_package":
				id.GoImportPath = "example.com/other"
				want.ImportPath = "example.com/other"
			case "unknown_file":
				path = "missing.proto"
				want.OutputPrefix = ""
			}
			assert.Equal(t, want, namedFieldType(plugin, id, generator.KindEnum, path, file))
		})
	}
}

func Test_outputPrefixOf(t *testing.T) {
	t.Parallel()
	plugin := newAdapterTestPlugin(t, adapterTestFile())
	assert.Equal(t, plugin.FilesByPath["test.proto"].GeneratedFilenamePrefix, outputPrefixOf(plugin, "test.proto"))
	assert.Equal(t, "", outputPrefixOf(plugin, "missing.proto"))
}

func Test_protocVersion(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		version *pluginpb.Version
		want    string
	}{
		{name: "missing", want: "(unknown)"},
		{name: "release", version: &pluginpb.Version{Major: new(int32(36)), Minor: new(int32(2)), Patch: new(int32(1))}, want: "v36.2.1"},
		{name: "suffix", version: &pluginpb.Version{Major: new(int32(36)), Suffix: new("rc1")}, want: "v36.0.0-rc1"},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			plugin := &protogen.Plugin{Request: &pluginpb.CodeGeneratorRequest{CompilerVersion: tt.version}}
			assert.Equal(t, tt.want, protocVersion(plugin))
		})
	}
}

func Test_optionScalar(t *testing.T) {
	t.Parallel()
	plugin := newAdapterTestPlugin(t, adapterTestFile())
	fields := plugin.FilesByPath["test.proto"].Messages[0].Desc.Fields()
	assert.Equal[any](t, "value", optionScalar(fields.ByName("name"), protoreflect.ValueOfString("value")))
	assert.Equal[any](t, int32(1), optionScalar(fields.ByName("state"), protoreflect.ValueOfEnum(1)))
	child := dynamicpb.NewMessage(fields.ByName("child").Message())
	child.Set(child.Descriptor().Fields().ByName("value"), protoreflect.ValueOfString("text"))
	assert.Equal[any](t, map[string]any{"value": "text"}, optionScalar(fields.ByName("child"), protoreflect.ValueOfMessage(child)))
}

func Test_optionValue(t *testing.T) {
	t.Parallel()
	plugin := newAdapterTestPlugin(t, adapterTestFile())
	message := dynamicpb.NewMessage(plugin.FilesByPath["test.proto"].Messages[0].Desc)
	fields := message.Descriptor().Fields()
	tags := message.Mutable(fields.ByName("tags")).List()
	tags.Append(protoreflect.ValueOfString("first"))
	tags.Append(protoreflect.ValueOfString("second"))
	labels := message.Mutable(fields.ByName("labels")).Map()
	labels.Set(protoreflect.ValueOfString("key").MapKey(), protoreflect.ValueOfString("value"))
	assert.Equal[any](t, []any{"first", "second"}, optionValue(fields.ByName("tags"), protoreflect.ValueOfList(tags)))
	assert.Equal[any](t, map[string]any{"key": "value"}, optionValue(fields.ByName("labels"), protoreflect.ValueOfMap(labels)))
	assert.Equal[any](t, int32(42), optionValue(fields.ByName("count"), protoreflect.ValueOfInt32(42)))
}

func adapterTestFile() *descriptorpb.FileDescriptorProto {
	rule := new(api.HttpRule)
	rule.SetPost("/call")
	options := new(descriptorpb.MethodOptions)
	proto.SetExtension(options, api.E_Http, rule)
	return &descriptorpb.FileDescriptorProto{
		Name: new("test.proto"), Package: new("test"), Syntax: new("proto3"),
		Options: &descriptorpb.FileOptions{GoPackage: new("example.com/test")}, Dependency: []string{"scaffold/api.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: new("Request"), Field: []*descriptorpb.FieldDescriptorProto{
				{Name: new("name"), Number: new(int32(1)), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				{Name: new("count"), Number: new(int32(2)), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()},
				{Name: new("tags"), Number: new(int32(3)), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()},
				{Name: new("state"), Number: new(int32(4)), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: new(".test.State")},
				{Name: new("child"), Number: new(int32(5)), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: new(".test.Child")},
				{Name: new("file"), Number: new(int32(6)), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: new(".api.File")},
				{Name: new("files"), Number: new(int32(7)), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: new(".api.File"), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()},
				{Name: new("labels"), Number: new(int32(8)), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: new(".test.Request.LabelsEntry"), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()},
			}, NestedType: []*descriptorpb.DescriptorProto{
				{Name: new("LabelsEntry"), Options: &descriptorpb.MessageOptions{MapEntry: new(true)}, Field: []*descriptorpb.FieldDescriptorProto{
					{Name: new("key"), Number: new(int32(1)), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: new("value"), Number: new(int32(2)), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				}},
				{Name: new("Inner")},
			}, EnumType: []*descriptorpb.EnumDescriptorProto{{Name: new("Mode"), Value: []*descriptorpb.EnumValueDescriptorProto{{Name: new("DEFAULT"), Number: new(int32(0))}}}}},
			{Name: new("Child"), Field: []*descriptorpb.FieldDescriptorProto{
				{Name: new("value"), Number: new(int32(1)), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
			}},
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{{Name: new("State"), Value: []*descriptorpb.EnumValueDescriptorProto{
			{Name: new("UNKNOWN"), Number: new(int32(0))}, {Name: new("READY"), Number: new(int32(1))},
		}}},
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: new("Service"), Method: []*descriptorpb.MethodDescriptorProto{
			{Name: new("Hidden"), InputType: new(".test.Request"), OutputType: new(".test.Child")},
			{Name: new("Call"), InputType: new(".test.Request"), OutputType: new(".test.Child"), Options: options},
		}}},
	}
}

func newAdapterTestPlugin(t *testing.T, files ...*descriptorpb.FileDescriptorProto) *protogen.Plugin {
	t.Helper()
	apiFile := protodesc.ToFileDescriptorProto(api.File_api_api_proto)
	apiFile.Name = new("scaffold/api.proto")
	descriptors := make([]*descriptorpb.FileDescriptorProto, 0, 2+len(files))
	descriptors = append(descriptors, protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto), apiFile)
	descriptors = append(descriptors, files...)
	plugin, err := (protogen.Options{}).New(&pluginpb.CodeGeneratorRequest{ProtoFile: descriptors, FileToGenerate: []string{"test.proto"}})
	if !assert.Nil(t, err) {
		t.FailNow()
	}
	return plugin
}
