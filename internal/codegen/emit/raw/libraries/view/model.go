package view

// ─── Enums ────────────────────────────────────────────────────────────────────

// EnumMemberModel is template data for a single enum constant.
type EnumMemberModel struct {
	ConstName    string // Go identifier for the constant (e.g. NSOrderedAscending)
	Value        string // Raw value expression (e.g. "-1", "0x1")
	ObjCName     string // Original ObjC name; used as the string in String() and Parse()
	CommentBlock string // Pre-rendered comment lines with "\t" prefix (may be empty)
	IsZeroVal    bool   // Bitmask enums: this member represents the zero-value case
}

// EnumModel is template data for a named ObjC enum.
type EnumModel struct {
	GoName        string            // Go type name (e.g. NSComparisonResult)
	GoType        string            // Underlying Go type (e.g. int64, uint64)
	IsBitmask     bool              // Bitmask enums use | combination, not switch
	HasMembers    bool              // Whether helper methods (String, Parse, etc.) should be emitted
	CommentBlock  string            // Pre-rendered type-level comment lines (may be empty)
	Members       []EnumMemberModel // All members for the const block
	UniqueMembers []EnumMemberModel // Deduped members for the String() switch (non-bitmask only)
	FirstConst    string            // First constant name; seeds result in ParseX (non-bitmask only)
	DefaultFmt    string            // Format string for String() default case, e.g. "NSFoo(%d)"
}

// ─── Externs ──────────────────────────────────────────────────────────────────

// ExternItemModel is template data for one extern var declaration.
type ExternItemModel struct {
	GoName       string // Go identifier
	GoType       string // Go type string
	CommentBlock string // Pre-rendered comment lines with "\t" prefix (may be empty)
	// InitExpr is the Go expression assigned in init(), reading the C global
	// through its bridge getter. Empty means the extern's shape is not
	// supported for initialisation and the var stays zero-valued.
	InitExpr string
	// SymbolName is the original C symbol backing the var.
	SymbolName string
}

// ExternsFileModel is template data for a complete _externs.go file.
type ExternsFileModel struct {
	PkgName string
	Imports []string
	Items   []ExternItemModel
	// BridgeInclude is the package-relative bridge header path (e.g.
	// "bridge/machinit_bridge.h"), set when at least one item has an InitExpr
	// so the file carries its own import "C" block.
	BridgeInclude string
	HasInit       bool
}

// BridgeExternGetterModel is template data for one extern-address getter in
// the bridge: a C function returning the address of the extern global so Go
// init() can populate the corresponding package var.
type BridgeExternGetterModel struct {
	SymbolName string // C symbol, e.g. "mach_task_self_"
	GetterName string // bridge function, e.g. "machinit_extern_mach_task_self_"
}

// ─── Structs ──────────────────────────────────────────────────────────────────

// StructFieldModel is template data for one field in a C struct.
type StructFieldModel struct {
	Name   string
	GoType string
}

// StructModel is template data for a regular C struct type definition.
type StructModel struct {
	GoName       string
	CommentBlock string
	Fields       []StructFieldModel
}

// ClassStructModel is template data for an ObjC class's Go struct: a root class
// owns the unsafe.Pointer field and a promoted Ptr() accessor, while a non-root
// class embeds its immediate superclass by value. Both emit a compile-time
// cgo.Object interface assertion.
type ClassStructModel struct {
	// CommentBlock is the wraps/superclass/protocols/swift-name doc plus the
	// shared context comments (column 0, trailing newline).
	CommentBlock string
	// TypeHeader is the type name with any generic constraint, e.g. "NSArray" or
	// "NSArray[T cgo.Object]".
	TypeHeader string
	// EmbedLine is the struct's single line: "ptr unsafe.Pointer" for a root
	// class, otherwise the embedded superclass field.
	EmbedLine string
	// IsRoot emits the ptr field's promoted Ptr() accessor.
	IsRoot bool
	// PtrReceiver is the receiver type for Ptr(), e.g. "NSObject" or
	// "NSArray[T]" (root only).
	PtrReceiver string
	// AssertType is the type in the `var _ cgo.Object = (*…)(nil)` assertion,
	// e.g. "NSObject" or "NSArray[cgo.Object]".
	AssertType string
}

// ClassConstructorsModel is template data for a class's core constructors: the
// tracked New<Class> pointer constructor, the untracked <Class>WithPtr value
// constructor (for embedding in subclass literals), and the Cast<Class> helper.
type ClassConstructorsModel struct {
	Name string
	// GenSuffix is "[cgo.Object]" for a generic class, otherwise empty.
	GenSuffix string
	// Chain is the value-chain literal that builds the wrapper, e.g.
	// "&NSMutableArray[cgo.Object]{NSArray[cgo.Object]{NSObject{ptr: ptr}}}".
	Chain string
	// ValueChain is Chain without the leading "&" — the value form returned by
	// <Class>WithPtr.
	ValueChain string
}

// DesignatedInitModel is template data for one New<Class>With<Arg> factory
// generated from a designated initializer. The factory allocates via the class
// alloc bridge, wraps the result untracked, calls the Go init method, and
// converts its result according to Kind.
type DesignatedInitModel struct {
	CtorName  string
	ClassName string
	Selector  string
	ArgList   string // the factory's Go parameter list
	ReturnSig string // "*Class" or "(*Class, error)"
	AllocFn   string // the C alloc bridge function name
	CallExpr  string // "_obj.<Init>(<args>)"
	// Kind selects the result handling: 0 the init already returns *Class;
	// 1 id-return + error; 2 id-return; 3 unsafe.Pointer-return + error;
	// 4 unsafe.Pointer-return.
	Kind int
}

// MethodBodyModel is template data for a CGo method body: the keep-alive
// defers, argument preambles, the C call, exception/NSError handling, and the
// return conversion selected by RetKind.
type MethodBodyModel struct {
	// HasReceiver emits `defer cgo.KeepAlive(<ReceiverVar>)` for instance methods
	// and id<Protocol> proxy methods.
	HasReceiver bool
	// ReceiverVar is the receiver variable name kept alive — "o" for class
	// instance methods, "p" for id<Protocol> proxy methods.
	ReceiverVar string
	// KeepAlives are ObjC-object argument names kept alive across the C call.
	KeepAlives []string
	// Preambles are pre-call setup statements (C-string conversions, block
	// trampolines) whose deferred cleanup runs at method exit.
	Preambles []string
	// HasNSError adds the trailing NSError out-parameter handling.
	HasNSError bool
	// RetKind selects capture + return handling: 0 void, 1 cgo.Object, 2 typed
	// object, 3 value struct, 4 nullable string, 5 scalar/other.
	RetKind int
	// RawCall is the C bridge call expression, e.g. "C.foo_bar(o.Ptr(), &_exc)".
	RawCall string
	// ResultExpr is the converted scalar result captured for RetKind 5.
	ResultExpr string
	// RetType is the Go return type, used for the value-struct zero literal and
	// pointer dereference (RetKind 3).
	RetType string
	// WrapTypedExpr is the constructor reference passed to cgo.WrapTyped for a
	// typed object return (RetKind 2).
	WrapTypedExpr string
}

// ClassMethodModel is template data for one generated instance or class method:
// the doc/annotation preamble, the bridge-ID comment, the signature, and the
// resolved CGo body.
type ClassMethodModel struct {
	// PreambleComment is the doc/context comments plus designated-init,
	// warn-unused, swift-name, blocked-import, and out-parameter notes (column 0).
	PreambleComment string
	// BridgeID is the "// <framework>_<Class>_<selector>…" identifier comment.
	BridgeID string
	// IsClassMethod selects a package-level function over an instance method.
	IsClassMethod bool
	// Receiver is the instance receiver type (instance methods only).
	Receiver string
	// GoName is the instance method name (instance methods only).
	GoName string
	// FuncName is the package-level function name (class methods only).
	FuncName string
	// GoArgs is the rendered parameter list.
	GoArgs string
	// RetStr is the return clause, " T" or empty.
	RetStr string
	// Body is the resolved method body.
	Body MethodBodyModel
	// Skip renders only the preamble comment (no function): a class method whose
	// generated name collides with a package-level type is dropped, but — matching
	// the original — its doc comment is still emitted.
	Skip bool
}

// CodingMethodsModel is template data for the NSSecureCoding/NSCoding archive
// convenience methods (SerializeToArchive + New<Class>FromArchive).
type CodingMethodsModel struct {
	Name          string
	SerializeFn   string
	DeserializeFn string
}

// GenericHelperModel is template data for the New<Class>T[T] typed constructor a
// generic class exposes so generic subclasses can build it with a preserved type
// parameter (rather than the cgo.Object instantiation).
type GenericHelperModel struct {
	Name   string
	TChain string // the &Class[T]{…} value chain
}

// TypedWithPtrModel is template data for the <Class>TypedWithPtr[T] untracked
// value constructor used by generic subclasses in other packages.
type TypedWithPtrModel struct {
	Name        string
	TValueChain string // the Class[T]{…} value chain (no leading &)
}

// NsStringOverloadModel is template data for a "…Go" Go-string convenience
// overload: a thin wrapper that converts its Go-string arguments and forwards to
// the underlying generated method/function.
type NsStringOverloadModel struct {
	// Signature is the full function signature up to the opening brace.
	Signature string
	// HasReturn is true when the wrapped call's result is returned.
	HasReturn bool
	// CallExpr is the forwarding call into the underlying method/function.
	CallExpr string
}

// CfWrapperModel is template data for a CoreFoundation opaque pointer wrapper type.
type CfWrapperModel struct {
	GoName string // Primary type name, e.g. CFStringRef
}

// CfAliasModel is template data for a CF typedef alias pointing to a primary wrapper.
type CfAliasModel struct {
	GoAlias   string // e.g. CFMutableStringRef
	GoPrimary string // e.g. CFStringRef
}

// StructTypedefModel is template data for a non-CF struct typedef alias.
type StructTypedefModel struct {
	GoAlias  string
	GoTarget string
}

// ─── Protocols ────────────────────────────────────────────────────────────────

// ProtocolMethodModel is template data for one method in a protocol interface.
type ProtocolMethodModel struct {
	GoName string // Go method name
	Params string // Full parameter list string of mapped ObjC args
	Ret    string // Return type string, empty for void
}

// ProtocolModel is template data for one ObjC @protocol → Go interface.
type ProtocolModel struct {
	GoName        string
	ObjCName      string
	AvailComment  string   // e.g. "Introduced: macOS 10.9" — empty when not set
	Embeds        []string // Embedded interface identifiers (same-fw and cross-fw)
	EmbedComments []string // Comment lines for blocked cross-fw embeds
	Methods       []ProtocolMethodModel
}

// ProtocolsFileModel is template data for a complete _protocols.go file.
type ProtocolsFileModel struct {
	PkgName   string
	Imports   []string
	Protocols []ProtocolModel
}

// ─── Functions ────────────────────────────────────────────────────────────────

// FunctionModel is template data for one C function → Go wrapper.
type FunctionModel struct {
	CommentBlock string
	BridgeID     string // e.g. "Foundation/NSStringFromClass"
	IsWarnUnused bool
	GoName       string
	Params       string   // Full parameter list of mapped ObjC args
	Ret          string   // Return type string, empty for void
	KeepAlives   []string // ObjC-object arg names to keep alive across the CGo call
	Preambles    []string // Statements to emit before the CGo call
	// CallBody is the pre-rendered CGo call + exception check + optional return.
	// The complex return-path dispatch (void / cgo.Object / value struct / primitive)
	// is resolved in Go (buildFunctionCallBody) so the template stays structural.
	CallBody string
}

// FunctionsFileModel is template data for a complete _functions.go file.
type FunctionsFileModel struct {
	PkgName   string
	FwLower   string
	Imports   []string
	Functions []FunctionModel
}

// ─── Classes ──────────────────────────────────────────────────────────────────

// ClassFileHeaderModel is template data for the generated header section of a class file:
// the generated-code comment, build tag, package declaration, CGo include, and import block.
type ClassFileHeaderModel struct {
	Framework    string   // e.g. "Foundation"
	PkgName      string   // e.g. "foundation"
	BridgeHeader string   // e.g. "foundation_bridge.h"
	Imports      []string // sorted, deduplicated import paths
}

// ─── Bridge ───────────────────────────────────────────────────────────────────

// BridgeDeclModel is template data for one C function declaration in a bridge header.
type BridgeDeclModel struct {
	Entitlements []string // // Requires entitlement: ... lines
	BridgeID     string   // e.g. "Foundation.NSString/stringWithCString:encoding:"
	Decl         string   // complete C declaration, without trailing semicolon
}

// BridgeAllocModel is template data for one alloc helper (declaration and implementation).
type BridgeAllocModel struct {
	ClassName string // ObjC class name, e.g. "NSWindow"
	FuncName  string // C function name, e.g. "foundation_NSWindow_alloc"
}

// BridgeProtoDeclModel is template data for a protocol proxy bridge declaration.
type BridgeProtoDeclModel struct {
	BridgeID string
	RetType  string
	FuncName string
	Params   string
}

// BridgeHeaderModel is template data for a complete _bridge.h file.
type BridgeHeaderModel struct {
	MethodDecls   []BridgeDeclModel
	AllocHelpers  []BridgeAllocModel
	CodingDecls   string // pre-rendered NSCoding declarations block
	FunctionDecls []BridgeDeclModel
	ForeignDecls  []BridgeDeclModel
	ProtoDecls    []BridgeProtoDeclModel
	ExternGetters []BridgeExternGetterModel
}

// BridgeImplMethodModel is template data for one ObjC bridge function implementation.
type BridgeImplMethodModel struct {
	Entitlements      []string
	BridgeID          string
	RetType           string // C return type, e.g. "void *" or "bool"
	FuncName          string
	Params            string
	IsNSError         bool
	NeedsFormatPragma bool // true for multi-keyword format-variadic methods where @"%@" trick cannot be used
	// TryBody is the pre-rendered content of the @try block: ObjC call + result handling.
	// The complex dispatch (void / object / struct-by-value / primitive + NSError variants)
	// lives in buildBridgeTryBody so the template shows only the @try/@catch structure.
	TryBody     string
	CatchReturn string // zero-value return for @catch, e.g. "return nil;" or ""
}

// BridgeAllocImplModel is template data for one alloc helper implementation.
type BridgeAllocImplModel struct {
	ClassName string
	FuncName  string
}

// BridgeImplModel is template data for a complete _bridge.m file.
type BridgeImplModel struct {
	Framework       string
	ParentFramework string
	// UmbrellaHeader is the C library umbrella include path relative to the
	// SDK's usr/include (e.g. "compression.h", "os/log.h"). When non-empty it
	// replaces the framework-style <Name/Name.h> import.
	UmbrellaHeader string
	// LocalHeader is the file name of a shim prototype header copied into the
	// bridge/ directory (private libraries with no SDK header). When non-empty
	// it is included with quotes and takes precedence over UmbrellaHeader.
	LocalHeader string
	HeaderName  string
	Methods        []BridgeImplMethodModel
	AllocImpls     []BridgeAllocImplModel
	CodingImpls    string // pre-rendered NSCoding implementation block
	Functions      []BridgeImplMethodModel
	ForeignMethods []BridgeImplMethodModel
	ProtoMethods   []BridgeImplMethodModel
	ExternGetters  []BridgeExternGetterModel
}

// BridgeShimModel is template data for the package-root _bridge_impl.m shim file.
// This file includes the real bridge implementation so CGo can find the symbols.
type BridgeShimModel struct {
	Stem string // e.g. "foundation_bridge" (produces bridge/foundation_bridge.m include)
}

// ─── Blocks ───────────────────────────────────────────────────────────────────

// BlockTypeModel is template data for one ObjC block typedef → Go named func type alias.
type BlockTypeModel struct {
	GoName    string // Go type alias name, e.g. CompletionBlock
	Framework string // Framework name for the doc comment, e.g. Foundation
	Sig       string // Complete Go func signature, e.g. func(result uint64, ok bool)
}

// ─── Interfaces ───────────────────────────────────────────────────────────────

// InterfaceMethodModel is template data for one method in a [ClassName]able interface.
type InterfaceMethodModel struct {
	GoName string // Go method name
	Params string // Full parameter list of mapped ObjC args
	Ret    string // Return type string, empty for void
}

// ClassInterfaceModel is template data for one [ClassName]able Go interface.
type ClassInterfaceModel struct {
	GoName      string // ObjC class name; the interface is GoName+"able"
	EmbedLine   string // Embedded type: "cgo.Object", "pkg.Superabled", or "Superable"
	Methods     []InterfaceMethodModel
	HasNSCoding bool
}

// ClassInterfacesFileModel is template data for a complete _interfaces.go file.
type ClassInterfacesFileModel struct {
	PkgName    string
	Imports    []string // sorted, deduplicated import paths
	UsesUnsafe bool     // true when any method references unsafe.Pointer
	UsesObjc   bool     // true when any method or embed references cgo.*
	Interfaces []ClassInterfaceModel
}

// ─── Variadic Wrappers ────────────────────────────────────────────────────────

// VariadicWrappersFileModel is template data for the Foundation variadic wrappers file.
type VariadicWrappersFileModel struct {
	PkgName string // Go package name, e.g. "foundation"
}

// ─── Block Trampolines ────────────────────────────────────────────────────────

// BlockTrampolineSigModel holds all pre-rendered strings for one block signature.
// It is used by the block_trampolines_go_file, block_trampolines_h_file, and
// block_trampolines_m_file templates.
type BlockTrampolineSigModel struct {
	Name string // canonical name: "bool", "void_ptr"
	// Go file fields
	GoParams    string // CGo param list: "key C.uint64_t, arg0 unsafe.Pointer"
	RetDecl     string // "" for void, " (result C.bool)" for non-void
	ClosureType string // "func(unsafe.Pointer)", "func() bool"
	CallBody    string // pre-rendered call statement(s), tab-indented
	// C file fields
	CRetType          string // "void", "bool", "int64_t"
	CForwardParams    string // "uint64_t key, void * arg0"
	CTrampolineParams string // "struct GoBlock *block, void * arg0"
	CTrampolineCall   string // "goCallBlock_bool(block->goKey);" or "return (bool)...;"
}

// BlockTrampolinesGoFileModel is template data for blocks_generated.go.
type BlockTrampolinesGoFileModel struct {
	PkgName string
	Sigs    []BlockTrampolineSigModel
}

// BlockTrampolinesHFileModel is template data for block_trampolines_generated.h.
type BlockTrampolinesHFileModel struct {
	Sigs []BlockTrampolineSigModel
}

// BlockTrampolinesMFileModel is template data for block_trampolines_generated.m.
type BlockTrampolinesMFileModel struct {
	Sigs []BlockTrampolineSigModel
}

// ─── Method Trampolines ───────────────────────────────────────────────────────

// MethodTrampolineSigModel holds all pre-rendered strings for one method IMP signature.
// It is used by the method_trampolines_go_file, method_trampolines_h_file, and
// method_trampolines_m_file templates.
type MethodTrampolineSigModel struct {
	Name    string // canonical name: "bool", "void_ptr"
	ObjCEnc string // ObjC type encoding: "v@:", "B@:@"
	// Go file fields
	GoParams string // "key C.uint64_t, self unsafe.Pointer, arg0 C.bool"
	RetDecl  string // "" for void, " (result C.bool)" for non-void
	CallBody string // pre-rendered call statement(s), tab-indented
	// C header fields
	IMPCDecl    string // "bool goIMP_bool(id self, SEL _cmd)"
	GoCallCDecl string // "bool goCallIMP_bool(uint64_t key, void* self)"
	// C impl fields
	IMPBody string // pre-rendered function body lines (without braces)
}

// MethodTrampolinesGoFileModel is template data for callbacks_generated.go.
type MethodTrampolinesGoFileModel struct {
	PkgName string
	Sigs    []MethodTrampolineSigModel
}

// MethodTrampolinesHFileModel is template data for method_trampolines_generated.h.
type MethodTrampolinesHFileModel struct {
	Sigs []MethodTrampolineSigModel
}

// MethodTrampolinesMFileModel is template data for method_trampolines_generated.m.
type MethodTrampolinesMFileModel struct {
	Sigs []MethodTrampolineSigModel
}

// ─── Subclass Factories ───────────────────────────────────────────────────────

// SubclassGoFileModel is template data for a *_subclass_generated.go file.
type SubclassGoFileModel struct {
	ClassName       string   // e.g. "NSCharacterSet"
	GoClassName     string   // e.g. "GoOrinSubNSCharacterSet"
	PackageName     string   // e.g. "foundation"
	IMPSigs         []string // sorted unique IMP sig names
	OverridesFields string   // pre-rendered struct field lines (tab-indented, ends with \n)
	AddMethodBody   string   // pre-rendered per-method add_method if-blocks (inside "if overrides != nil")
	BindMethodBody  string   // pre-rendered per-method BindMethod if-blocks (inside "if overrides != nil")
}

// SubclassHFileModel is template data for a *_subclass_generated.h file.
type SubclassHFileModel struct {
	GoClassName string
	IMPSigs     []string
}

// SubclassMFileModel is template data for a *_subclass_generated.m file.
type SubclassMFileModel struct {
	Framework   string // e.g. "Foundation"
	ClassName   string // e.g. "NSCharacterSet"
	GoClassName string // e.g. "GoOrinSubNSCharacterSet"
	HeaderFile  string // e.g. "NSCharacterSet_subclass_generated.h"
	IMPSigs     []string
}

// ─── Protocol Impls ───────────────────────────────────────────────────────────

// ProtocolImplGoFileModel is template data for a *_protocol_callback.go file.
type ProtocolImplGoFileModel struct {
	PackageName         string   // e.g. "foundation"
	GoClassName         string   // e.g. "goBridgeProto_NSCacheDelegate"
	ProtoName           string   // e.g. "NSCacheDelegate"
	CallbacksStructName string   // e.g. "NSCacheDelegateCallbacks"
	FactoryName         string   // e.g. "NewNSCacheDelegateProtocolCallback"
	IMPSigs             []string // sorted unique IMP sig names
	NeedsObjc           bool     // true when return type is cgo.Object (no proxy)
	ImplFields          string   // pre-rendered struct field lines (tab-indented)
	FactoryComment      string   // pre-rendered factory function doc comment block
	ReturnType          string   // "*NSCacheDelegateIDProtocol" or "cgo.Object"
	AddMethodBody       string   // pre-rendered per-method addMethod if-blocks (no outer wrapper)
	BindMethodBody      string   // pre-rendered per-method BindMethod if-blocks
	ReturnBody          string   // pre-rendered return statement(s) (tab-indented)
}

// ProtocolImplHFileModel is template data for a *_impl_generated.h file.
type ProtocolImplHFileModel struct {
	GoClassName string
	IMPSigs     []string
}

// ProtocolImplMFileModel is template data for a *_impl_generated.m file.
type ProtocolImplMFileModel struct {
	Framework   string
	ProtoName   string
	GoClassName string
	HeaderFile  string // e.g. "NSCacheDelegate_impl_generated.h"
	IMPSigs     []string
}

// ─── Protocol Proxies ─────────────────────────────────────────────────────────

// ProtocolProxiesFileModel is template data for a complete _protocol_proxies.go file.
type ProtocolProxiesFileModel struct {
	PkgName      string
	BridgeHeader string   // e.g. "foundation_bridge.h"
	Imports      []string // sorted, deduplicated import paths
	ProxyTypes   []ProtocolProxyTypeModel
}

// ProtocolProxyMethodModel is template data for one method on a protocol proxy type.
type ProtocolProxyMethodModel struct {
	GoName       string // Go method name
	Params       string // full parameter list of mapped ObjC args
	Ret          string // return type string, empty for void
	AvailComment string // availability comment, empty if none
	BodyLines    string // pre-rendered method body lines (tab-indented)
}

// ProtocolProxyTypeModel is template data for one id<Protocol> wrapper type.
type ProtocolProxyTypeModel struct {
	IDTypeName      string // e.g. "NSCacheDelegateIDProtocol"
	ProtoName       string // e.g. "NSCacheDelegate"
	NSObjectEmbed   string // "NSObject" or "foundation.NSObject"
	NSObjectWithPtr string // "NSObjectWithPtr" or "foundation.NSObjectWithPtr"
	AvailComment    string // availability comment, empty if none
	Methods         []ProtocolProxyMethodModel
}
