package meta

// Availability holds the macOS platform availability for a Swift declaration.
type Availability struct {
	MacOSIntroduced  string // e.g. "15.0"
	MacOSUnavailable bool
}

// EnumCase is a single case in a Swift enum.
type EnumCase struct {
	Name         string // Swift case name, e.g. "installed"
	RawValue     string // explicit raw value string, e.g. "0", "\"foo\""; empty if implicit
	Availability Availability
}

// Enum represents a Swift enum whose cases can be represented in Go as typed constants.
// This covers both plain enums (iota) and raw-value enums (Swift.Int, Swift.String, etc.).
type Enum struct {
	// Name is the flattened Go type name, e.g. "LanguageAvailabilityStatus"
	// (nested type "Status" inside parent "LanguageAvailability" is flattened).
	Name string
	// RawGoType is the Go backing type for raw-value enums: "int64", "string", etc.
	// Empty string means plain enum — cases are assigned iota.
	RawGoType    string
	Cases        []EnumCase
	Availability Availability
}

// StaticCode is a named sentinel value (static let) on an error struct.
type StaticCode struct {
	Name         string // Swift member name, e.g. "unsupportedSourceLanguage"
	Availability Availability
}

// ErrorStruct represents a Swift struct that conforms to Error or LocalizedError.
// Each static let member becomes a named Go error sentinel.
type ErrorStruct struct {
	Name         string // Go type name, e.g. "TranslationError"
	StaticCodes  []StaticCode
	Availability Availability
}

// StaticValue is a named constant on a value-enum struct (opaque type with named instances).
type StaticValue struct {
	Name         string // Swift member name, e.g. "highFidelity"
	Availability Availability
}

// StructField is a stored property on a Swift struct.
type StructField struct {
	Name         string // Go field name (PascalCase applied by emitter)
	GoType       string // resolved Go type, e.g. "string", "*string", "int64", "[]byte"
	Optional     bool   // true when original Swift type had trailing ?
	Availability Availability
}

// Struct represents a Swift struct with stored properties.
// IsValueEnum is true for structs that have no stored fields but have static-let
// named instances (the Swift pattern for opaque value types like TranslationSession.Strategy).
type Struct struct {
	Name         string
	Fields       []StructField
	StaticValues []StaticValue
	IsValueEnum  bool
	Availability Availability
}

// Param is a single parameter in a Swift method signature.
type Param struct {
	Label string // external call-site label; "_" means no label
	Name  string // internal parameter name
	Type  string // raw Swift type string, e.g. "Swift.String", "Foundation.Locale.Language?"
}

// SwiftReturn is the return type of a Swift method.
type SwiftReturn struct {
	Type     string // raw Swift type string; "" means Void
	Optional bool   // true when Type ends with "?"
	IsAsync  bool   // true when the getter is declared "get async"
}

// Method is a Swift method or initializer declaration on a class.
type Method struct {
	Name         string  // Swift method name, e.g. "translate"; "" for unnamed operators
	Params       []Param // positional parameters
	RawParams    string  // raw parameter list string for verbatim shim generation
	Return       SwiftReturn
	IsAsync      bool
	IsThrows     bool
	IsStatic     bool
	IsInit       bool // true for init(...) declarations
	IsMainActor  bool // @_Concurrency.MainActor annotation on the declaration line
	Availability Availability
}

// Property is a Swift property on a class (stored or computed).
type Property struct {
	Name         string
	Type         string // raw Swift type string
	Optional     bool
	Writable     bool // has both get and set accessors, or is a stored var
	IsAsync      bool // declared "{ get async }"
	IsStatic     bool
	IsMainActor  bool
	Availability Availability
}

// Class is a Swift class or actor declaration.
type Class struct {
	Name         string
	IsActor      bool
	IsFinal      bool
	IsOpen       bool
	SuperClass   string   // first conformance that is a class (not a protocol), or ""
	Conformances []string // raw conformance/protocol list
	Methods      []Method
	Properties   []Property
	Availability Availability
}

// FrameworkMeta is the complete parsed metadata for a Swift framework.
type FrameworkMeta struct {
	Framework    string
	Classes      []*Class
	Enums        []*Enum
	ErrorStructs []*ErrorStruct
	Structs      []*Struct
}

// HasContent reports whether the metadata contains any generated types.
func (framework *FrameworkMeta) HasContent() bool {
	return len(framework.Classes) > 0 || len(framework.Enums) > 0 || len(framework.ErrorStructs) > 0 || len(framework.Structs) > 0
}
