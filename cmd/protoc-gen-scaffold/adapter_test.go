package main

import (
	"testing"

	"google.golang.org/protobuf/compiler/protogen"

	"go.icytown.com/scaffold/internal/assert"
	"go.icytown.com/scaffold/internal/gen/api"
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
