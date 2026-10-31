package generator

import (
	"errors"
	"fmt"
	"math"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"go.icytown.com/scaffold/internal/template"
)

func init() {
	template.RegisterFuncs(map[string]any{
		"contains":          strings.Contains,
		"isEmptyType":       isEmptyType,
		"renderDocComment":  renderDocComment,
		"renderGoName":      renderGoName,
		"renderGoNamedType": renderGoNamedType,
		"renderGoType":      renderGoType,
		"renderGoValue":     renderGoValue,
		"renderLineComment": renderLineComment,
		"renderTSType":      renderTSType,
		"toLowerCamelCase":  toLowerCamelCase,
		"trimPrefix":        strings.TrimPrefix,
	})
}

// collectPathParams 按路由顺序提取单段路径参数对应的字段，并校验字段为标量或枚举
func collectPathParams(file *File, method *Method) ([]*Field, error) {
	var params []string
	seen := make(map[string]bool)
	for segment := range strings.SplitSeq(method.Pattern, "/") {
		if !strings.ContainsAny(segment, "{}") {
			continue
		}
		name, ok := strings.CutPrefix(segment, "{")
		if !ok {
			return nil, fmt.Errorf("invalid path parameter %q", segment)
		}
		name, ok = strings.CutSuffix(name, "}")
		if !ok || name == "" || strings.ContainsAny(name, "{}.$") {
			return nil, fmt.Errorf("unsupported path parameter %q: only {name} is supported", segment)
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate path parameter %q", name)
		}
		seen[name] = true
		params = append(params, name)
	}
	if len(params) == 0 {
		return nil, nil
	}
	if isEmptyType(method.RequestType) {
		return nil, errors.New("path parameters are not allowed when request type is api.Empty")
	}
	if method.RequestType == nil || method.RequestType.Kind != KindMessage || method.RequestType.ImportPath != "" {
		return nil, errors.New("path parameters require a request message defined in the current file")
	}
	var request *Message
	for _, message := range file.Messages {
		if message.Name == method.RequestType.Name {
			request = message
			break
		}
	}
	if request == nil {
		return nil, fmt.Errorf("request message %q not found", method.RequestType.Name)
	}
	fields := make(map[string]*Field, len(request.Fields))
	for _, field := range request.Fields {
		fields[field.Name] = field
	}
	pathFields := make([]*Field, 0, len(params))
	for _, name := range params {
		field := fields[name]
		if field == nil {
			return nil, fmt.Errorf("path parameter %q has no matching field in request %s", name, request.Name)
		}
		if field.Repeated || field.Type == nil {
			return nil, fmt.Errorf("path parameter %q must be a scalar or enum field", name)
		}
		switch field.Type.Kind {
		case KindBool, KindInt32, KindInt64, KindUint32, KindUint64, KindFloat32, KindFloat64, KindString, KindBytes, KindEnum:
		default:
			return nil, fmt.Errorf("path parameter %q must be a scalar or enum field", name)
		}
		pathFields = append(pathFields, field)
	}
	return pathFields, nil
}

// isEmptyType 判断类型是否为 api.Empty，不将其他无字段消息视为空类型。
func isEmptyType(t *Type) bool {
	return t != nil && t.Kind == KindMessage && t.Name == "Empty" &&
		t.ImportPath == "go.icytown.com/scaffold/internal/gen/api"
}

// renderDocComment 把原始注释文本格式化为文档注释块。
func renderDocComment(comment string) string {
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return ""
	}
	if !strings.Contains(comment, "\n") {
		return "/** " + comment + " */"
	}
	var b strings.Builder
	b.WriteString("/**\n")
	for line := range strings.SplitSeq(comment, "\n") {
		b.WriteString(" *")
		if line != "" {
			b.WriteByte(' ')
			b.WriteString(strings.TrimPrefix(line, " "))
		}
		b.WriteByte('\n')
	}
	b.WriteString(" */")
	return b.String()
}

// renderLineComment 把原始注释文本格式化为行注释块。
func renderLineComment(comment string) string {
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

// goInitialisms 是 Go 命名惯例中的常见首字母缩写词，命中时整段大写。
// 来源 staticcheck 默认配置: https://staticcheck.dev/docs/configuration/options/#initialisms
var (
	goInitialisms = []string{
		"ACL", "API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "QPS",
		"RAM", "RPC", "SLA", "SMTP", "SQL", "SSH", "TCP", "TLS", "TTL", "UDP", "UI", "GID", "UID", "UUID", "URI", "URL",
		"UTF8", "VM", "XML", "XMPP", "XSRF", "XSS", "SIP", "RTP", "AMQP", "DB", "TS",
	}
	goInitialismIndex = func() map[string]string {
		m := make(map[string]string, len(goInitialisms))
		for _, s := range goInitialisms {
			m[strings.ToLower(s)] = s
		}
		return m
	}()
)

// renderGoName 把 IDL 中的原始字段名转换为导出的 Go 字段名，
// 按下划线分段后大写每段首字母，如 user_name 转为 UserName；
// 命中 Go 惯用缩写词的分段整体大写，如 user_id 转为 UserID。
func renderGoName(name string) string {
	var b strings.Builder
	for seg := range strings.SplitSeq(name, "_") {
		if seg == "" {
			continue
		}
		if initialism := goInitialismIndex[strings.ToLower(seg)]; initialism != "" {
			b.WriteString(initialism)
			continue
		}
		b.WriteString(strings.ToUpper(seg[:1]))
		b.WriteString(seg[1:])
	}
	return b.String()
}

// renderGoType 渲染 Go 类型，不支持数组。
func renderGoType(t *Type) string {
	switch t.Kind {
	case KindFile:
		return "*multipart.FileHeader"
	case KindBytes:
		return "[]byte"
	case KindEnum:
		return renderGoNamedType(t)
	case KindMap:
		return "map[" + renderGoType(t.MapKey) + "]" + renderGoType(t.MapElem)
	case KindMessage:
		return "*" + renderGoNamedType(t)
	default:
		return string(t.Kind)
	}
}

// renderGoNamedType 渲染消息或枚举的类型名，跨包时添加包名前缀，不添加指针符号。
func renderGoNamedType(t *Type) string {
	if t.ImportPath == "" {
		return t.Name
	}
	return path.Base(t.ImportPath) + "." + t.Name
}

// renderGoValue 将基础值及嵌套集合转换为 Go 代码，可指定换行的集合层数，默认不换行
func renderGoValue(value any, lineBreakDepth ...int) (string, error) {
	depth := 0
	if len(lineBreakDepth) > 0 {
		depth = lineBreakDepth[0]
	}
	switch v := value.(type) {
	case nil:
		return "nil", nil
	case bool:
		return strconv.FormatBool(v), nil
	case string:
		return strconv.Quote(v), nil
	case int:
		return "int(" + strconv.FormatInt(int64(v), 10) + ")", nil
	case int8:
		return "int8(" + strconv.FormatInt(int64(v), 10) + ")", nil
	case int16:
		return "int16(" + strconv.FormatInt(int64(v), 10) + ")", nil
	case int32:
		return fmt.Sprintf("int32(%d)", v), nil
	case int64:
		return fmt.Sprintf("int64(%d)", v), nil
	case uint:
		return "uint(" + strconv.FormatUint(uint64(v), 10) + ")", nil
	case uint8:
		return "uint8(" + strconv.FormatUint(uint64(v), 10) + ")", nil
	case uint16:
		return "uint16(" + strconv.FormatUint(uint64(v), 10) + ")", nil
	case uint32:
		return fmt.Sprintf("uint32(%d)", v), nil
	case uint64:
		return fmt.Sprintf("uint64(%d)", v), nil
	case uintptr:
		return "uintptr(" + strconv.FormatUint(uint64(v), 10) + ")", nil
	case float32:
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return "", fmt.Errorf("unsupported Go value %v", v)
		}
		return "float32(" + strconv.FormatFloat(float64(v), 'g', -1, 32) + ")", nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return "", fmt.Errorf("unsupported Go value %v", v)
		}
		return "float64(" + strconv.FormatFloat(v, 'g', -1, 64) + ")", nil
	case []byte:
		if v == nil {
			return "nil", nil
		}
		return "[]byte(" + strconv.Quote(string(v)) + ")", nil
	case []any:
		if v == nil {
			return "nil", nil
		}
		parts := make([]string, len(v))
		for i, item := range v {
			literal, err := renderGoValue(item, depth-1)
			if err != nil {
				return "", err
			}
			parts[i] = literal
		}
		if len(parts) == 0 {
			return "[]any{}", nil
		}
		if depth > 0 {
			return "[]any{\n" + strings.Join(parts, ",\n") + ",\n}", nil
		}
		return "[]any{" + strings.Join(parts, ", ") + "}", nil
	case map[string]any:
		if v == nil {
			return "nil", nil
		}
		return renderGoMap(v, "map[string]any", depth)
	case map[string]map[string]any:
		if v == nil {
			return "nil", nil
		}
		values := make(map[string]any, len(v))
		for key, item := range v {
			values[key] = item
		}
		return renderGoMap(values, "map[string]map[string]any", depth)
	default:
		return "", fmt.Errorf("unsupported Go value %T", value)
	}
}

// renderGoMap 按键名字典序渲染 map 字面量
func renderGoMap(values map[string]any, typeName string, depth int) (string, error) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		literal, err := renderGoValue(values[key], depth-1)
		if err != nil {
			return "", err
		}
		parts = append(parts, strconv.Quote(key)+": "+literal)
	}
	if len(parts) == 0 {
		return typeName + "{}", nil
	}
	if depth > 0 {
		return typeName + "{\n" + strings.Join(parts, ",\n") + ",\n}", nil
	}
	return typeName + "{" + strings.Join(parts, ", ") + "}", nil
}

// renderTSType 渲染 TypeScript 类型，可选别名按模块路径和原类型名查找，数组由调用方追加 []
func renderTSType(t *Type, aliases ...map[[2]string]string) string {
	if t == nil {
		return "unknown"
	}
	switch t.Kind {
	case KindFile:
		return "File"
	case KindBool:
		return "boolean"
	case KindInt32, KindInt64, KindUint32, KindUint64, KindFloat32, KindFloat64:
		return "number"
	case KindString:
		return "string"
	case KindBytes:
		return "Uint8Array"
	case KindEnum, KindMessage:
		if len(aliases) > 0 && !isEmptyType(t) {
			module := t.OutputPrefix
			if module != "" {
				module = filepath.ToSlash(filepath.Clean(module))
			}
			if name, ok := aliases[0][[2]string{module, t.Name}]; ok {
				return name
			}
		}
		return t.Name
	case KindMap:
		return "Record<" + renderTSType(t.MapKey, aliases...) + ", " + renderTSType(t.MapElem, aliases...) + ">"
	default:
		return "unknown"
	}
}

// toLowerCamelCase 把导出的标识符按单词转换为 lowerCamelCase。
func toLowerCamelCase(name string) string {
	runes := []rune(name)
	previousUpper := false
	for i, char := range runes {
		if !unicode.IsUpper(char) {
			previousUpper = false
			continue
		}
		if i == 0 || previousUpper && (i+1 == len(runes) || unicode.IsUpper(runes[i+1])) {
			runes[i] = unicode.ToLower(char)
		}
		previousUpper = true
	}
	return string(runes)
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
