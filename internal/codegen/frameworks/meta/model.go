package meta

// FrameworkMeta is the Go-optimised metadata for a single macOS framework,
// produced by the scanner and consumed by the purego code generator.
// Serialised as <framework>-<arch>-<sdk>.gometa.json.
type FrameworkMeta struct {
	Framework  string               `json:"framework"`
	SDKVersion string               `json:"sdk_version"`
	Arch       string               `json:"arch"`
	Classes    map[string]Class     `json:"classes"`
	Protocols  map[string]Protocol  `json:"protocols"`
	Enums      map[string]Enum      `json:"enums"`
	Structs    map[string]Struct    `json:"structs"`
	Functions  []Function           `json:"functions"`
	Externs    []Extern             `json:"externs"`
	BlockTypes map[string]BlockType `json:"block_types"`
	Typedefs   map[string]string    `json:"typedefs"`

	IsSwiftOnly     bool     `json:"swift_only,omitempty"`
	ParentFramework string   `json:"parent_framework,omitempty"`
	UmbrellaFor     []string `json:"umbrella_for,omitempty"`

	ForeignExtensions map[string][]Method `json:"foreign_extensions,omitempty"`
	DeclaredImports   map[string]bool     `json:"declared_imports,omitempty"`

	// LinkLib, when non-empty, means this is a plain C library (not an ObjC framework).
	// The generated package uses purego.Dlopen with the library path instead of a
	// framework path.
	LinkLib string `json:"link_lib,omitempty"`

	// SchemaVersion records which generation of the .gometa.json schema wrote
	// the file (stamped by internal/meta.Write at scan time); Read checks it
	// against internal/meta's supported window.
	SchemaVersion int `json:"schema_version,omitempty"`

	// ClangVersion and XcodeVersion are toolchain provenance recorded at scan
	// time. Carried here so tools reading via this package can surface them.
	ClangVersion string `json:"clang_version,omitempty"`
	XcodeVersion string `json:"xcode_version,omitempty"`
}

// Class represents an Objective-C class (@interface).
type Class struct {
	Super         string       `json:"super,omitempty"`
	Protocols     []string     `json:"protocols,omitempty"`
	GenericParams []string     `json:"generic_params,omitempty"`
	Methods       []Method     `json:"methods,omitempty"`
	Properties    []Property   `json:"properties,omitempty"`
	Availability  Availability `json:"availability,omitempty"`
	SDKFile       string       `json:"sdk_file,omitempty"`
	SDKLine       int          `json:"sdk_line,omitempty"`
	SwiftName     string       `json:"swift_name,omitempty"`
	Doc           string       `json:"doc,omitempty"`

	// IsMainThreadRequired is set at load time (not scanned) when the class is
	// isolated to Swift's @MainActor — directly or by inheriting it from an
	// ancestor — meaning its instances must be used on the main thread. The
	// idiomatic emitter wraps such calls in purego.Main. Populated by the
	// mainactor sidecar merge + hierarchy propagation; never serialised to
	// .gometa.json.
	IsMainThreadRequired bool `json:"-"`
}

// Protocol represents an Objective-C @protocol.
type Protocol struct {
	InheritedProtocols []string     `json:"protocols,omitempty"`
	Methods            []Method     `json:"methods,omitempty"`
	Availability       Availability `json:"availability,omitempty"`
}

// Method represents a single Objective-C method selector.
type Method struct {
	Selector             string       `json:"selector"`
	IsClassMethod        bool         `json:"class_method,omitempty"`
	Params               []Param      `json:"args,omitempty"`
	Return               ReturnType   `json:"return"`
	IsInit               bool         `json:"is_init,omitempty"`
	IsNSError            bool         `json:"has_nserror,omitempty"`
	IsVariadic           bool         `json:"variadic,omitempty"`
	Availability         Availability `json:"availability,omitempty"`
	SDKFile              string       `json:"sdk_file,omitempty"`
	SDKLine              int          `json:"sdk_line,omitempty"`
	IsDesignatedInit     bool         `json:"designated_init,omitempty"`
	IsWarnUnused         bool         `json:"warn_unused,omitempty"`
	SwiftName            string       `json:"swift_name,omitempty"`
	Doc                  string       `json:"doc,omitempty"`
	IsOptional           bool         `json:"is_optional,omitempty"`
	IsMainThreadRequired bool         `json:"main_thread_required,omitempty"`

	// IsMainThreadExempt is set at load time (not scanned) when the method is
	// `nonisolated` — it explicitly opts out of its class's @MainActor isolation
	// and must NOT be wrapped in purego.Main even though the class is otherwise
	// main-thread-bound. Populated by the mainactor sidecar merge.
	IsMainThreadExempt bool `json:"-"`
}

// Param is a method argument.
type Param struct {
	Name       string `json:"name"`
	ObjCType   string `json:"objc_type"`
	IsBlock    bool   `json:"is_block,omitempty"`
	BlockRef   string `json:"block_ref,omitempty"`
	IsNullable bool   `json:"nullable,omitempty"`
	IsNoescape bool   `json:"no_escape,omitempty"`
	Direction  string `json:"modifier,omitempty"`
}

// ReturnType describes a method's return value.
type ReturnType struct {
	ObjCType          string `json:"objc_type"`
	IsGeneric         bool   `json:"is_generic,omitempty"`
	GenericParamName  string `json:"generic_param_name,omitempty"`
	IsInstancetype    bool   `json:"instancetype,omitempty"`
	IsAlreadyRetained bool   `json:"already_retained,omitempty"`
	IsNullable        bool   `json:"nullable,omitempty"`
}

// Property is an Objective-C @property declaration.
type Property struct {
	Name         string       `json:"name"`
	ObjCType     string       `json:"objc_type"`
	IsReadOnly   bool         `json:"readonly,omitempty"`
	IsWeak       bool         `json:"weak,omitempty"`
	IsCopy       bool         `json:"copy,omitempty"`
	Getter       string       `json:"getter,omitempty"`
	Setter       string       `json:"setter,omitempty"`
	Availability Availability `json:"availability,omitempty"`
	SDKFile      string       `json:"sdk_file,omitempty"`
	SDKLine      int          `json:"sdk_line,omitempty"`
	Doc          string       `json:"doc,omitempty"`
}

// Enum is an Objective-C enum (NS_ENUM, NS_OPTIONS, or plain enum).
type Enum struct {
	GoType       string       `json:"go_type"`
	Members      []EnumMember `json:"members,omitempty"`
	IsAnon       bool         `json:"is_anon,omitempty"`
	Availability Availability `json:"availability,omitempty"`
	SDKFile      string       `json:"sdk_file,omitempty"`
	SDKLine      int          `json:"sdk_line,omitempty"`
	IsBitmask    bool         `json:"is_bitmask,omitempty"`
	IsExtensible bool         `json:"is_extensible,omitempty"`
	Doc          string       `json:"doc,omitempty"`
}

// EnumMember is a single constant within an enum.
type EnumMember struct {
	Name         string       `json:"name"`
	Value        string       `json:"value"`
	Availability Availability `json:"availability,omitempty"`
	Doc          string       `json:"doc,omitempty"`
}

// Struct is a C struct exposed by the framework.
type Struct struct {
	Fields []StructField `json:"fields,omitempty"`
	// Packed is true when the C struct carries __attribute__((packed)). The
	// idiomatic emitter uses it to decide whether a plain Go struct reproduces
	// the C ABI layout: a packed struct is only safe to surface as a typed Go
	// value when its natural layout already needs no padding.
	Packed bool `json:"packed,omitempty"`
	// Size is the total struct size in bytes from clang's authoritative record
	// layout (0 when unknown). Used with each field's Offset to cross-check that
	// the emitted Go struct reproduces the C ABI.
	Size         int          `json:"size,omitempty"`
	Availability Availability `json:"availability,omitempty"`
	SDKFile      string       `json:"sdk_file,omitempty"`
	SDKLine      int          `json:"sdk_line,omitempty"`
	Doc          string       `json:"doc,omitempty"`
}

// StructField is one field within a struct.
type StructField struct {
	Name     string `json:"name"`
	ObjCType string `json:"objc_type"`
	GoType   string `json:"go_type,omitempty"`
	// Offset is the field's byte offset from clang's authoritative record layout.
	Offset int `json:"offset,omitempty"`
}

// Function is a plain C function declared in the framework headers.
type Function struct {
	Name         string       `json:"name"`
	Params       []Param      `json:"args,omitempty"`
	Return       ReturnType   `json:"return"`
	IsInline     bool         `json:"inline,omitempty"`
	IsVariadic   bool         `json:"variadic,omitempty"`
	Availability Availability `json:"availability,omitempty"`
	SDKFile      string       `json:"sdk_file,omitempty"`
	SDKLine      int          `json:"sdk_line,omitempty"`
	IsWarnUnused bool         `json:"warn_unused,omitempty"`
	Doc          string       `json:"doc,omitempty"`
}

// Extern is an extern symbol (global constant or variable).
type Extern struct {
	Name         string       `json:"name"`
	ObjCType     string       `json:"objc_type"`
	GoType       string       `json:"go_type,omitempty"`
	Availability Availability `json:"availability,omitempty"`
	SDKFile      string       `json:"sdk_file,omitempty"`
	SDKLine      int          `json:"sdk_line,omitempty"`
	Doc          string       `json:"doc,omitempty"`
}

// BlockType is a unique ObjC block signature.
type BlockType struct {
	Params []Param    `json:"args,omitempty"`
	Return ReturnType `json:"return"`
}

// Availability carries macOS API availability information.
type Availability struct {
	MacOSIntroduced string   `json:"macos_introduced,omitempty"`
	MacOSDeprecated string   `json:"macos_deprecated,omitempty"`
	MacOSObsoleted  string   `json:"macos_obsoleted,omitempty"`
	DeprecationMsg  string   `json:"deprecation_message,omitempty"`
	ReplacedBy      string   `json:"replaced_by,omitempty"`
	IsUnavailable   bool     `json:"unavailable,omitempty"`
	Entitlements    []string `json:"entitlements,omitempty"`
}
