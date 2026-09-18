package generator

import (
	"testing"

	"go.icytown.com/scaffold/internal/assert"
)

func Test_collectPathParams(t *testing.T) {
	t.Parallel()
	apiEmpty := &Type{
		Kind:       KindMessage,
		Name:       "Empty",
		ImportPath: "go.icytown.com/scaffold/internal/gen/api",
	}
	testCases := []struct {
		name    string
		file    *File
		method  *Method
		want    []string
		wantErr string
	}{
		{
			name:   "without_params",
			file:   &File{},
			method: &Method{Pattern: "/users"},
		},
		{
			name: "scalar_and_enum_params",
			file: &File{Messages: []*Message{{
				Name: "Request",
				Fields: []*Field{
					{Name: "user_id", Type: &Type{Kind: KindInt64}},
					{Name: "state", Type: &Type{Kind: KindEnum, Name: "State"}},
				},
			}}},
			method: &Method{
				Pattern:     "/users/{user_id}/{state}",
				RequestType: &Type{Kind: KindMessage, Name: "Request"},
			},
			want: []string{"user_id", "state"},
		},
		{
			name:    "invalid_segment",
			file:    &File{},
			method:  &Method{Pattern: "/users/user_id}"},
			wantErr: `invalid path parameter "user_id}"`,
		},
		{
			name:    "unsupported_segment",
			file:    &File{},
			method:  &Method{Pattern: "/users/{user.id}"},
			wantErr: `unsupported path parameter "{user.id}": only {name} is supported`,
		},
		{
			name:    "duplicate_param",
			file:    &File{},
			method:  &Method{Pattern: "/users/{id}/{id}"},
			wantErr: `duplicate path parameter "id"`,
		},
		{
			name:    "empty_request",
			file:    &File{},
			method:  &Method{Pattern: "/users/{id}", RequestType: apiEmpty},
			wantErr: "path parameters are not allowed when request type is api.Empty",
		},
		{
			name:    "nil_request",
			file:    &File{},
			method:  &Method{Pattern: "/users/{id}"},
			wantErr: "path parameters require a request message defined in the current file",
		},
		{
			name:    "external_request",
			file:    &File{},
			method:  &Method{Pattern: "/users/{id}", RequestType: &Type{Kind: KindMessage, Name: "Request", ImportPath: "example.com/api"}},
			wantErr: "path parameters require a request message defined in the current file",
		},
		{
			name:    "request_not_found",
			file:    &File{},
			method:  &Method{Pattern: "/users/{id}", RequestType: &Type{Kind: KindMessage, Name: "Request"}},
			wantErr: `request message "Request" not found`,
		},
		{
			name: "field_not_found",
			file: &File{Messages: []*Message{{Name: "Request"}}},
			method: &Method{
				Pattern:     "/users/{id}",
				RequestType: &Type{Kind: KindMessage, Name: "Request"},
			},
			wantErr: `path parameter "id" has no matching field in request Request`,
		},
		{
			name: "repeated_field",
			file: &File{Messages: []*Message{{Name: "Request", Fields: []*Field{{
				Name: "id", Type: &Type{Kind: KindInt64}, Repeated: true,
			}}}}},
			method: &Method{
				Pattern:     "/users/{id}",
				RequestType: &Type{Kind: KindMessage, Name: "Request"},
			},
			wantErr: `path parameter "id" must be a scalar or enum field`,
		},
		{
			name: "nil_field_type",
			file: &File{Messages: []*Message{{Name: "Request", Fields: []*Field{{Name: "id"}}}}},
			method: &Method{
				Pattern:     "/users/{id}",
				RequestType: &Type{Kind: KindMessage, Name: "Request"},
			},
			wantErr: `path parameter "id" must be a scalar or enum field`,
		},
		{
			name: "message_field",
			file: &File{Messages: []*Message{{Name: "Request", Fields: []*Field{{
				Name: "user", Type: &Type{Kind: KindMessage, Name: "User"},
			}}}}},
			method: &Method{
				Pattern:     "/users/{user}",
				RequestType: &Type{Kind: KindMessage, Name: "Request"},
			},
			wantErr: `path parameter "user" must be a scalar or enum field`,
		},
	}
	for _, tt := range testCases {
		got, err := collectPathParams(tt.file, tt.method)
		if tt.wantErr == "" {
			assert.Nil(t, err, tt.name)
			assert.Equal(t, tt.want, got, tt.name)
			continue
		}
		if !assert.NotNil(t, err, tt.name) {
			continue
		}
		assert.Equal(t, tt.wantErr, err.Error(), tt.name)
	}
}

func Test_isEmptyType(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		arg  *Type
		want bool
	}{
		{name: "api_empty", arg: &Type{
			Kind:       KindMessage,
			Name:       "Empty",
			ImportPath: "go.icytown.com/scaffold/internal/gen/api",
		}, want: true},
		{name: "nil_type", arg: nil, want: false},
		{name: "empty_without_import_path", arg: &Type{Kind: KindMessage, Name: "Empty"}, want: false},
		{name: "empty_with_other_import_path", arg: &Type{
			Kind:       KindMessage,
			Name:       "Empty",
			ImportPath: "example.com/other",
		}, want: false},
		{name: "other_message_in_api", arg: &Type{
			Kind:       KindMessage,
			Name:       "User",
			ImportPath: "go.icytown.com/scaffold/internal/gen/api",
		}, want: false},
		{name: "empty_with_enum_kind", arg: &Type{
			Kind:       KindEnum,
			Name:       "Empty",
			ImportPath: "go.icytown.com/scaffold/internal/gen/api",
		}, want: false},
	}
	for _, tt := range testCases {
		assert.Equal(t, tt.want, isEmptyType(tt.arg), tt.name)
	}
}

func Test_renderDocComment(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		arg  string
		want string
	}{
		{name: "empty", arg: "", want: ""},
		{name: "whitespace_only", arg: "   ", want: ""},
		{name: "single_line", arg: "user service", want: "/** user service */"},
		{name: "single_line_trims_spaces", arg: "  user service  ", want: "/** user service */"},
		{name: "multi_line", arg: "line one\nline two", want: "/**\n * line one\n * line two\n */"},
		{name: "multi_line_with_blank_line", arg: "line one\n\nline two", want: "/**\n * line one\n *\n * line two\n */"},
		{name: "multi_line_strips_one_leading_space", arg: "line one\n  line two", want: "/**\n * line one\n *  line two\n */"},
	}
	for _, tt := range testCases {
		assert.Equal(t, tt.want, renderDocComment(tt.arg), tt.name)
	}
}

func Test_renderLineComment(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		arg  string
		want string
	}{
		{name: "empty", arg: "", want: ""},
		{name: "whitespace_only", arg: "   ", want: ""},
		{name: "single_line", arg: "user service", want: "// user service"},
		{name: "single_line_trims_spaces", arg: "  user service  ", want: "// user service"},
		{name: "multi_line", arg: "line one\nline two", want: "// line one\n// line two\n"},
		{name: "multi_line_with_blank_line", arg: "line one\n\nline two", want: "// line one\n//\n// line two\n"},
		{name: "multi_line_strips_one_leading_space", arg: "line one\n  line two", want: "// line one\n//  line two\n"},
	}
	for _, tt := range testCases {
		assert.Equal(t, tt.want, renderLineComment(tt.arg), tt.name)
	}
}

func Test_renderGoName(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		arg  string
		want string
	}{
		{name: "single_segment", arg: "name", want: "Name"},
		{name: "snake_case", arg: "user_name", want: "UserName"},
		{name: "initialism_segment", arg: "user_id", want: "UserID"},
		{name: "whole_initialism", arg: "api", want: "API"},
		{name: "initialism_prefix", arg: "api_key", want: "APIKey"},
		{name: "initialism_suffix", arg: "http_method", want: "HTTPMethod"},
		{name: "initialism_mixed_case_segment", arg: "user_Id", want: "UserID"},
		{name: "empty", arg: "", want: ""},
		{name: "leading_underscore", arg: "_name", want: "Name"},
		{name: "trailing_underscore", arg: "name_", want: "Name"},
		{name: "double_underscore", arg: "user__name", want: "UserName"},
		{name: "only_underscores", arg: "___", want: ""},
		{name: "single_char_segments", arg: "a_b", want: "AB"},
	}
	for _, tt := range testCases {
		assert.Equal(t, tt.want, renderGoName(tt.arg), tt.name)
	}
}

func Test_renderGoNamedType(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		arg  *Type
		want string
	}{
		{name: "same_package", arg: &Type{Kind: KindMessage, Name: "User"}, want: "User"},
		{name: "cross_package", arg: &Type{
			Kind:       KindMessage,
			Name:       "Empty",
			ImportPath: "go.icytown.com/scaffold/internal/gen/api",
		}, want: "api.Empty"},
		{name: "nested_package_path", arg: &Type{
			Kind:       KindEnum,
			Name:       "Status",
			ImportPath: "example.com/foo/bar",
		}, want: "bar.Status"},
	}
	for _, tt := range testCases {
		assert.Equal(t, tt.want, renderGoNamedType(tt.arg), tt.name)
	}
}

func Test_renderGoType(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		arg  *Type
		want string
	}{
		{name: "bytes", arg: &Type{Kind: KindBytes}, want: "[]byte"},
		{name: "scalar", arg: &Type{Kind: KindString}, want: "string"},
		{name: "message_same_package", arg: &Type{Kind: KindMessage, Name: "User"}, want: "*User"},
		{name: "message_cross_package", arg: &Type{
			Kind:       KindMessage,
			Name:       "Empty",
			ImportPath: "go.icytown.com/scaffold/internal/gen/api",
		}, want: "*api.Empty"},
		{name: "enum_same_package", arg: &Type{Kind: KindEnum, Name: "Status"}, want: "Status"},
		{name: "enum_cross_package", arg: &Type{
			Kind:       KindEnum,
			Name:       "Status",
			ImportPath: "go.icytown.com/scaffold/internal/gen/api",
		}, want: "api.Status"},
		{name: "map_scalar", arg: &Type{
			Kind:    KindMap,
			MapKey:  &Type{Kind: KindString},
			MapElem: &Type{Kind: KindInt32},
		}, want: "map[string]int32"},
		{name: "map_with_message_value", arg: &Type{
			Kind:    KindMap,
			MapKey:  &Type{Kind: KindString},
			MapElem: &Type{Kind: KindMessage, Name: "User"},
		}, want: "map[string]*User"},
		{name: "map_with_bytes_value", arg: &Type{
			Kind:    KindMap,
			MapKey:  &Type{Kind: KindString},
			MapElem: &Type{Kind: KindBytes},
		}, want: "map[string][]byte"},
		{name: "nested_map", arg: &Type{
			Kind:   KindMap,
			MapKey: &Type{Kind: KindString},
			MapElem: &Type{
				Kind:    KindMap,
				MapKey:  &Type{Kind: KindInt64},
				MapElem: &Type{Kind: KindBool},
			},
		}, want: "map[string]map[int64]bool"},
	}
	for _, tt := range testCases {
		assert.Equal(t, tt.want, renderGoType(tt.arg), tt.name)
	}
}

func Test_renderTSType(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		arg  *Type
		want string
	}{
		{name: "nil", arg: nil, want: "unknown"},
		{name: "bool", arg: &Type{Kind: KindBool}, want: "boolean"},
		{name: "int32", arg: &Type{Kind: KindInt32}, want: "number"},
		{name: "int64", arg: &Type{Kind: KindInt64}, want: "number"},
		{name: "uint32", arg: &Type{Kind: KindUint32}, want: "number"},
		{name: "uint64", arg: &Type{Kind: KindUint64}, want: "number"},
		{name: "float32", arg: &Type{Kind: KindFloat32}, want: "number"},
		{name: "float64", arg: &Type{Kind: KindFloat64}, want: "number"},
		{name: "string", arg: &Type{Kind: KindString}, want: "string"},
		{name: "bytes", arg: &Type{Kind: KindBytes}, want: "Uint8Array"},
		{name: "enum", arg: &Type{Kind: KindEnum, Name: "State"}, want: "State"},
		{name: "message", arg: &Type{Kind: KindMessage, Name: "User"}, want: "User"},
		{name: "map", arg: &Type{
			Kind:    KindMap,
			MapKey:  &Type{Kind: KindString},
			MapElem: &Type{Kind: KindMessage, Name: "User"},
		}, want: "Record<string, User>"},
		{name: "unknown", arg: &Type{Kind: FieldKind("unknown")}, want: "unknown"},
	}
	for _, tt := range testCases {
		assert.Equal(t, tt.want, renderTSType(tt.arg), tt.name)
	}
}

func Test_toLowerCamelCase(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		arg  string
		want string
	}{
		{name: "empty", arg: "", want: ""},
		{name: "camel_case", arg: "GetUser", want: "getUser"},
		{name: "initialism", arg: "GetURL", want: "getUrl"},
		{name: "leading_initialism", arg: "URLValue", want: "urlValue"},
		{name: "initialism_only", arg: "URL", want: "url"},
		{name: "multiple_initialisms", arg: "HTTPServerURL", want: "httpServerUrl"},
	}
	for _, tt := range testCases {
		assert.Equal(t, tt.want, toLowerCamelCase(tt.arg), tt.name)
	}
}

func Test_versionLines(t *testing.T) {
	t.Parallel()
	t.Run("no_dependencies", func(t *testing.T) {
		t.Parallel()
		lines := versionLines(&File{})
		assert.Equal(t, []string{"scaffold " + scaffoldVersion}, lines)
	})
	t.Run("dependencies_shorter_than_scaffold", func(t *testing.T) {
		t.Parallel()
		file := &File{
			Dependencies: []*Dependency{
				{Name: "gorm", Version: "v1.25.0"},
				{Name: "protobuf", Version: "v1.36.5"},
			},
		}
		lines := versionLines(file)
		assert.Equal(t, []string{
			"scaffold " + scaffoldVersion,
			"gorm     v1.25.0",
			"protobuf v1.36.5",
		}, lines)
	})
	t.Run("dependency_longer_than_scaffold", func(t *testing.T) {
		t.Parallel()
		file := &File{
			Dependencies: []*Dependency{
				{Name: "grpc-go", Version: "v1.65.0"},
				{Name: "protobuf-go", Version: "v1.36.5"},
			},
		}
		lines := versionLines(file)
		assert.Equal(t, []string{
			"scaffold    " + scaffoldVersion,
			"grpc-go     v1.65.0",
			"protobuf-go v1.36.5",
		}, lines)
	})
}
