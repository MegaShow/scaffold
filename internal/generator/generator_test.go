package generator

import (
	"testing"

	"go.icytown.com/scaffold/internal/assert"
)

func TestVersion(t *testing.T) {
	t.Parallel()
	assert.Equal(t, scaffoldVersion, Version())
}

func Test_resolveVersion(t *testing.T) { //nolint:paralleltest // 测试会修改全局的 injectedVersion，不能并行执行。
	original := injectedVersion
	t.Cleanup(func() {
		injectedVersion = original
	})

	injectedVersion = "v1.2.3"
	assert.Equal(t, "v1.2.3", resolveVersion(), "injected_version")

	injectedVersion = ""
	assert.Equal(t, "dev", resolveVersion(), "development_version")
}

func TestNew(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		lang     Language
		kind     Kind
		wantType string
		wantErr  string
	}{
		{name: "go_server", lang: LanguageGo, kind: KindServer, wantType: "go_server"},
		{name: "typescript_client", lang: LanguageTypeScript, kind: KindClient, wantType: "typescript_client"},
		{
			name:    "go_client",
			lang:    LanguageGo,
			kind:    KindClient,
			wantErr: `generator: unsupported language "go" with kind "client"`,
		},
		{
			name:    "typescript_server",
			lang:    LanguageTypeScript,
			kind:    KindServer,
			wantErr: `generator: unsupported language "typescript" with kind "server"`,
		},
		{
			name:    "unknown_language",
			lang:    Language("unknown"),
			kind:    KindServer,
			wantErr: `generator: unsupported language "unknown" with kind "server"`,
		},
		{
			name:    "unknown_kind",
			lang:    LanguageGo,
			kind:    Kind("unknown"),
			wantErr: `generator: unsupported language "go" with kind "unknown"`,
		},
	}
	for _, tt := range testCases {
		got, err := New(tt.lang, tt.kind)
		if tt.wantErr != "" {
			assert.Nil(t, got, tt.name)
			if !assert.NotNil(t, err, tt.name) {
				continue
			}
			assert.Equal(t, tt.wantErr, err.Error(), tt.name)
			continue
		}
		assert.Nil(t, err, tt.name)
		gotType := "unknown"
		switch got.(type) {
		case *goServerGenerator:
			gotType = "go_server"
		case *tsClientGenerator:
			gotType = "typescript_client"
		}
		assert.Equal(t, tt.wantType, gotType, tt.name)
	}
}
