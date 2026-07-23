package macosplatformmetadata

// CurrentSchemaVersion is the .gometa.json schema version written by this
// generator. Bump it whenever FrameworkMeta (or any type it embeds) changes
// in a way that older committed metadata can no longer satisfy — renamed or
// re-tagged fields, changed semantics, removed fields. A bump forces a
// re-scan: Read rejects files outside the supported window with a clear
// "re-scan required" error instead of silently deserialising stale data
// into zero-valued fields.
const CurrentSchemaVersion = 1

// MinSupportedSchemaVersion is the oldest schema version Read still accepts.
// Version 0 denotes legacy metadata written before the field existed; raise
// this past 0 once the committed metadata tree has been fully re-scanned and
// stamped.
const MinSupportedSchemaVersion = 0

// FrameworkMeta is the Go-optimised metadata for a single macOS framework,
// produced by the scanner and consumed by the code generator.
// Serialised as <framework>-<arch>-<sdk>.gometa.json.
type FrameworkMeta struct {
	Framework  string `json:"framework"`
	SDKVersion string `json:"sdk_version"`
	Arch       string `json:"arch"`

	// SchemaVersion records which generation of the .gometa.json schema wrote
	// this file. Stamped automatically by Write; checked by Read. Zero means
	// the file predates schema versioning.
	SchemaVersion int `json:"schema_version,omitempty"`

	// ClangVersion and XcodeVersion record the toolchain that produced this
	// scan (e.g. "Apple clang version 21.0.0 (clang-2100.3.9.2)" and
	// "Xcode 26.0 Build version 17A321"). Clang releases have materially
	// changed AST output before (Clang 21 dropped availability version data,
	// forcing the header-source fallback in scanner/comments.go), so the
	// producing toolchain must be answerable from the artifact itself.
	ClangVersion string               `json:"clang_version,omitempty"`
	XcodeVersion string               `json:"xcode_version,omitempty"`
	Classes      map[string]Class     `json:"classes"`
	Protocols    map[string]Protocol  `json:"protocols"`
	Enums        map[string]Enum      `json:"enums"`
	Structs      map[string]Struct    `json:"structs"`
	Functions    []Function           `json:"functions"`
	Externs      []Extern             `json:"externs"`
	BlockTypes   map[string]BlockType `json:"block_types"`
	Typedefs     map[string]string    `json:"typedefs"`

	// IsSwiftOnly is true when the framework has no ObjC surface area — its API
	// is entirely Swift and cannot be bridged via CGo. The generator emits a
	// documentation-only package in this case.
	IsSwiftOnly bool `json:"swift_only,omitempty"`

	// ParentFramework is set for sub-frameworks (e.g. "Carbon" for HIToolbox).
	// The linker flag must use the parent (-framework Carbon) rather than the
	// child name, which is not a top-level framework in the SDK.
	ParentFramework string `json:"parent_framework,omitempty"`

	// UmbrellaFor lists constituent framework names re-exported by this
	// umbrella framework (e.g. Carbon → HIToolbox, CommonPanels, …).
	// When non-empty the umbrella itself has no own classes; all content
	// lives in the named constituent frameworks.
	UmbrellaFor []string `json:"umbrella_for,omitempty"`

	// ForeignExtensions captures ObjC categories defined in this framework
	// that extend a class owned by a different framework (e.g. AppleScriptObjC
	// adds -loadAppleScriptObjectiveCScripts to NSBundle). In Go these are
	// emitted as package-level functions rather than methods, because Go does
	// not allow adding methods to types from other packages.
	// Key is the foreign class name (e.g. "NSBundle").
	ForeignExtensions map[string][]Method `json:"foreign_extensions,omitempty"`

	// DeclaredImports is the set of other framework names that this framework's
	// own headers directly include via #import/#include. Populated by the scanner
	// from the IncludedFrom chain in the Clang AST: if a header from framework B
	// is directly included by a header belonging to this framework, B is a
	// declared import. Used by the cycle-breaker to prefer cutting undeclared
	// cross-framework edges over declared ones.
	DeclaredImports map[string]bool `json:"declared_imports,omitempty"`

	// LinkLib, when non-empty, overrides the default -framework <Name> linker
	// flag with -l<LinkLib>. Set for C libraries that ship as plain dylibs rather
	// than .framework bundles (e.g. EndpointSecurity → "EndpointSecurity").
	LinkLib string `json:"link_lib,omitempty"`

	// Header is the umbrella header include path relative to the SDK's
	// usr/include directory, set only for C libraries (LinkLib != "").
	// The bridge emitter uses it in the #include directive (e.g.
	// "compression.h", "os/log.h", "bsm/libbsm.h") in place of the
	// framework-style <Name/Name.h> import.
	Header string `json:"header,omitempty"`

	// ShimHeader is the repo-relative path of a hand-maintained prototype
	// header for C libraries that ship no header in the SDK (private dylibs
	// such as IOReport, which has a linkable .tbd stub but no public
	// declarations). Stamped at scan time from the C library registry. The
	// scanner parses this file instead of an SDK header, and the bridge
	// emitter copies it into the generated package's bridge/ directory and
	// includes it with quotes in place of the umbrella include.
	ShimHeader string `json:"shim_header,omitempty"`
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
	// InheritedProtocols lists the protocols this protocol inherits from
	// (ObjCProtocolDecl::protocols() in the Clang AST). Not to be confused
	// with Class.Protocols, which lists protocols a class conforms to.
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

// Param is a method argument (ParmVarDecl in the Clang AST).
type Param struct {
	Name       string `json:"name"`
	ObjCType   string `json:"objc_type"`
	IsBlock    bool   `json:"is_block,omitempty"`
	BlockRef   string `json:"block_ref,omitempty"`
	IsNullable bool   `json:"nullable,omitempty"`
	IsNoescape bool   `json:"no_escape,omitempty"`
	// Direction indicates the parameter's direction convention.
	// "out"   — callee writes, caller reads (e.g. __autoreleasing ** params)
	// "inout" — caller writes then callee overwrites
	// ""      — input (default) or unknown
	Direction string `json:"modifier,omitempty"`
}

// ReturnType describes a method's return value.
type ReturnType struct {
	ObjCType  string `json:"objc_type"`
	IsGeneric bool   `json:"is_generic,omitempty"`
	// GenericParamName is the name of the generic type parameter when IsGeneric
	// is true (e.g. "ObjectType" for -[NSArray<ObjectType> firstObject]).
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
	IsWeak       bool         `json:"weak,omitempty"` // property has the "weak" attribute
	IsCopy       bool         `json:"copy,omitempty"` // property has the "copy" attribute
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
	// the C ABI layout (a packed struct is only safe to surface as a typed value
	// when its natural layout already needs no padding).
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

// BlockType is a unique ObjC block signature, collected so the generator
// can emit a named Go func type and a CGo trampoline.
type BlockType struct {
	Params []Param    `json:"args,omitempty"`
	Return ReturnType `json:"return"`
}

// Availability carries macOS API availability information extracted from
// API_AVAILABLE / API_DEPRECATED attributes in the headers.
type Availability struct {
	MacOSIntroduced string `json:"macos_introduced,omitempty"`
	MacOSDeprecated string `json:"macos_deprecated,omitempty"`
	MacOSObsoleted  string `json:"macos_obsoleted,omitempty"`
	DeprecationMsg  string `json:"deprecation_message,omitempty"`
	ReplacedBy      string `json:"replaced_by,omitempty"`
	// IsUnavailable is true when the symbol is marked API_UNAVAILABLE(macos) in
	// the SDK. Unlike deprecated symbols, unavailable ones generate a hard Clang
	// error if called and must be omitted from the bridge entirely.
	IsUnavailable bool `json:"unavailable,omitempty"`
	// Entitlements lists com.apple.* entitlement keys required to call this API,
	// extracted from SDK header doc comments. Non-empty does NOT mean unavailable —
	// the API is callable once the app has the entitlement provisioned.
	Entitlements []string `json:"entitlements,omitempty"`
}
