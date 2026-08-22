package generator

// File 描述一个待生成的 IDL 文件。
type File struct {
	Dependencies []*Dependency // 生成依赖组件及版本
	Path         string        // IDL 源文件路径，如 example/user.proto
	OutputPrefix string        // 生成文件后缀，如 _scaffold.go
	Package      string        // 包名
	Services     []*Service    // 服务列表
	Messages     []*Message    // 消息类型列表
	Enums        []*Enum       // 枚举类型列表
}

// Dependency 描述生成过程中依赖的组件。
type Dependency struct {
	Name    string // 组件名
	Version string // 组件版本
}

// Service 描述一个服务定义。
type Service struct {
	Name    string    // 服务名
	Methods []*Method // 方法列表
	Comment string    // 注释
}

// Method 描述服务内的一个方法。
type Method struct {
	Name         string     // 方法名
	HTTPMethod   HTTPMethod // HTTP 方法
	Pattern      string     // 路由模式
	Comment      string     // 注释
	RequestType  *Type      // 请求类型
	ResponseType *Type      // 响应类型
}

// HTTPMethod 是 HTTP 方法。
type HTTPMethod string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
	PATCH  HTTPMethod = "PATCH"
)

// Message 描述一个消息类型定义。
type Message struct {
	Name    string   // 类型名
	Comment string   // 注释
	Fields  []*Field // 字段列表
}

// Field 描述消息的一个字段。
type Field struct {
	Name     string // 字段名
	Comment  string // 注释
	Type     *Type  // 字段类型
	Repeated bool   // 是否为列表
}

// Type 描述类型。
type Type struct {
	Kind       FieldKind // 类型种类
	Name       string    // 类型名，仅 KindMessage/KindEnum 时有效
	ImportPath string    // 类型所属包的导入路径，为空表示与生成代码同包
	MapKey     *Type     // KindMap 的键类型
	MapElem    *Type     // KindMap 的值类型
}

// FieldKind 描述字段类型种类，标量种类的取值即对应的 Go 类型名。
type FieldKind string

// 支持的类型种类。
const (
	KindBool    FieldKind = "bool"
	KindInt32   FieldKind = "int32"
	KindInt64   FieldKind = "int64"
	KindUint32  FieldKind = "uint32"
	KindUint64  FieldKind = "uint64"
	KindFloat32 FieldKind = "float32"
	KindFloat64 FieldKind = "float64"
	KindString  FieldKind = "string"
	KindBytes   FieldKind = "bytes"
	KindEnum    FieldKind = "enum"
	KindMessage FieldKind = "message"
	KindMap     FieldKind = "map"
)

// Enum 描述一个枚举类型定义。
type Enum struct {
	Name    string       // 类型名
	Comment string       // 声明前注释，原始文本
	Values  []*EnumValue // 枚举的取值列表
}

// EnumValue 描述枚举的一个取值。
type EnumValue struct {
	Name    string // 常量名
	Comment string // 注释
	Number  int32  // 枚举取值编号
}

// GeneratedFile 描述一份生成的代码文件。
type GeneratedFile struct {
	Path    string // 生成文件的输出路径，如 example/user_scaffold.go
	Content []byte // 生成文件的代码内容
}
