package generator

import (
	"strings"

	"go.icytown.com/scaffold/internal/template"
)

func init() {
	template.RegisterFuncs(map[string]any{
		"contains":        strings.Contains,
		"renderGoComment": renderGoComment,
		"renderGoType":    renderGoType,
	})
}

// renderGoComment 把原始注释文本格式化为 Go 行注释块。
func renderGoComment(comment string) string {
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return ""
	}
	if !strings.Contains(comment, "\n") {
		return "// " + comment
	}
	var b strings.Builder
	for line := range strings.SplitSeq(comment, "\n") {
		if line == "" {
			b.WriteString("//\n")
			continue
		}
		b.WriteString("// ")
		b.WriteString(strings.TrimPrefix(line, " "))
		b.WriteByte('\n')
	}
	return b.String()
}

// renderGoFieldType 渲染消息字段类型。
func renderGoType(t *Type, repeated bool) string {
	s := baseGoType(t)
	if repeated {
		return "[]" + s
	}
	return s
}

// baseGoType 渲染字段的基础 Go 类型，不含 Repeated 修饰。
func baseGoType(t *Type) string {
	switch t.Kind {
	case KindMessage:
		return "*" + baseGoValueType(t)
	case KindEnum:
		return baseGoValueType(t)
	case KindMap:
		return "map[" + baseGoType(t.MapKey) + "]" + baseGoType(t.MapElem)
	default:
		return string(t.Kind)
	}
}

// baseGoValueType 渲染字段的值类型。
func baseGoValueType(t *Type) string {
	if t.ImportPath == "" {
		return t.Name
	}
	pkg := t.ImportPath
	if idx := strings.LastIndex(pkg, "/"); idx >= 0 {
		pkg = pkg[idx+1:]
	}
	return pkg + "." + t.Name
}

// versionLines 计算版本注释行：scaffold 自身与各依赖组件按名字长度对齐，组件名列取最长名字对齐后跟一个空格，保证版本号在同一列左对齐。
func versionLines(file *File) []string {
	entries := make([][2]string, 0, len(file.Dependencies)+1)
	entries = append(entries, [2]string{"scaffold", scaffoldVersion})
	for _, dep := range file.Dependencies {
		entries = append(entries, [2]string{dep.Name, dep.Version})
	}

	maxLen := len(entries[0][0])
	for _, e := range entries {
		if len(e[0]) > maxLen {
			maxLen = len(e[0])
		}
	}

	lines := make([]string, 0, len(entries))
	for _, e := range entries {
		lines = append(lines, e[0]+strings.Repeat(" ", maxLen-len(e[0]))+" "+e[1])
	}
	return lines
}
