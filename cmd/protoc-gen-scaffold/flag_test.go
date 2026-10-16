package main

import (
	"testing"

	"go.icytown.com/scaffold/internal/assert"
)

func Test_injectMetaFlag_Set(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		arg     string
		want    injectMetaFlag
		wantErr bool
	}{
		{name: "enable", arg: "true", want: injectMetaFlag{enabled: true}},
		{name: "disable", arg: "false", want: injectMetaFlag{enabled: false}},
		{name: "options", arg: "dev.access,com.example.audit", want: injectMetaFlag{enabled: true, options: []string{"dev.access", "com.example.audit"}}},
		{name: "invalid", arg: "invalid", wantErr: true},
	}
	for _, tt := range testCases {
		value := injectMetaFlag{}
		err := value.Set(tt.arg)
		assert.Equal(t, tt.wantErr, err != nil, tt.name)
		assert.Equal(t, tt.want, value, tt.name)
	}
}

func Test_stringsFlag_Set(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		initial stringsFlag
		arg     string
		want    stringsFlag
	}{
		{name: "first", arg: "first", want: stringsFlag{"first"}},
		{name: "append", initial: stringsFlag{"first"}, arg: "second", want: stringsFlag{"first", "second"}},
		{name: "empty", arg: "", want: stringsFlag{""}},
	}
	for _, tt := range testCases {
		value := tt.initial
		assert.Nil(t, value.Set(tt.arg), tt.name)
		assert.Equal(t, tt.want, value, tt.name)
	}
}
