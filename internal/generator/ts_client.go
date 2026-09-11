package generator

import (
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
	return files, nil
}

// generateMessageFile 生成 TypeScript 类型声明文件，对应 ts_message.go.tpl。
func (g *tsClientGenerator) generateMessageFile(file *File) (*GeneratedFile, error) {
	data := toTmplTSMessageFile(file)
	content, err := template.Render("ts_message.go.tpl", data)
	if err != nil {
		return nil, err
	}
	return &GeneratedFile{
		Path:    file.OutputPrefix + ".d.ts",
		Content: content,
	}, nil
}

// tmplTSMessageFile 是 TypeScript 类型定义文件的模板数据，对应 tmplGoMessageFile。
type tmplTSMessageFile struct {
	Source   string     // 源文件路径
	Versions []string   // 版本注释行
	Imports  []string   // import 语句列表
	Messages []*Message // 消息类型列表，按名字字典序排列
	Enums    []*Enum    // 枚举类型列表，按名字字典序排列
}

func toTmplTSMessageFile(file *File) *tmplTSMessageFile {
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

	return &tmplTSMessageFile{
		Source:   file.Path,
		Versions: versionLines(file),
		Messages: messages,
		Enums:    enums,
	}
}
