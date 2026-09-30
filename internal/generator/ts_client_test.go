package generator

import (
	"testing"

	"go.icytown.com/scaffold/internal/assert"
)

func Test_collectTSImports(t *testing.T) {
	t.Parallel()
	sharedMessage := &Type{Kind: KindMessage, Name: "SharedMessage", OutputPrefix: "example/gen/shared"}
	types := []*Type{
		sharedMessage,
		sharedMessage,
		{Kind: KindEnum, Name: "SharedState", OutputPrefix: "example/gen/shared"},
		{Kind: KindMessage, Name: "LocalMessage", OutputPrefix: "example/gen/user"},
		{
			Kind:    KindMap,
			MapKey:  &Type{Kind: KindString},
			MapElem: &Type{Kind: KindMessage, Name: "OtherMessage", OutputPrefix: "example/gen/other/other"},
		},
	}

	imports, err := collectTSImports("example/gen/user", false, types...)
	assert.Nil(t, err)
	assert.Equal(t, []*tmplTSImport{
		{Path: "./other/other", Names: []string{"OtherMessage"}},
		{Path: "./shared", Names: []string{"SharedMessage", "SharedState"}},
	}, imports)

	imports, err = collectTSImports("example/gen/user", true, types...)
	assert.Nil(t, err)
	assert.Equal(t, []*tmplTSImport{
		{Path: "./other/other", Names: []string{"OtherMessage"}},
		{Path: "./shared", Names: []string{"SharedMessage", "SharedState"}},
		{Path: "./user", Names: []string{"LocalMessage"}},
	}, imports)
}

func Test_relativeTSImportPath(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		from string
		to   string
		want string
	}{
		{name: "same_directory", from: "example/gen/user", to: "example/gen/shared", want: "./shared"},
		{name: "child_directory", from: "example/gen/user", to: "example/gen/other/other", want: "./other/other"},
		{name: "parent_directory", from: "example/gen/other/user", to: "example/gen/shared", want: "../shared"},
	}
	for _, tt := range testCases {
		got, err := relativeTSImportPath(tt.from, tt.to)
		assert.Nil(t, err, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func Test_toTmplTSServiceFile(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name         string
		request      string
		response     string
		wantRequest  string
		wantResponse string
		wantImports  []string
	}{
		{"reserved_names", "Request", "Response", "Request_", "Response_", []string{"Request as Request_", "Response as Response_"}},
		{"ordinary_names", "Input", "Output", "Input", "Output", []string{"Input", "Output"}},
		{"existing_underscore", "Request", "Request_", "Request__", "Request_", []string{"Request_", "Request as Request__"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			request := &Type{Kind: KindMessage, Name: tt.request}
			response := &Type{Kind: KindMessage, Name: tt.response}
			file := &File{OutputPrefix: "example/gen/user", Services: []*Service{{
				Name: "UserService", Methods: []*Method{{Name: "Get", HTTPMethod: GET, Pattern: "/users", RequestType: request, ResponseType: response}},
			}}}
			data, err := toTmplTSServiceFile(file)
			assert.Nil(t, err)
			assert.Equal(t, []*tmplTSImport{
				{Path: "@icytown/scaffold-client", Names: []string{"Client", "Request", "RequestComposer", "ResponseResolver"}},
				{Path: "./user", Names: tt.wantImports},
			}, data.Imports)
			assert.Equal(t, tt.wantRequest, data.Aliases[[2]string{"", tt.request}])
			assert.Equal(t, tt.wantResponse, data.Aliases[[2]string{"", tt.response}])
			assert.Equal(t, tt.wantRequest, data.Aliases[[2]string{file.OutputPrefix, tt.request}])
			assert.Equal(t, tt.wantResponse, data.Aliases[[2]string{file.OutputPrefix, tt.response}])
			assert.Equal(t, tt.request, request.Name)
			assert.Equal(t, tt.response, response.Name)
		})
	}
}
