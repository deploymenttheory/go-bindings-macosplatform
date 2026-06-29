package raw

// ─── Enums ────────────────────────────────────────────────────────────────────

// enumMemberModel is template data for a single enum constant.
type enumMemberModel struct {
	ConstName    string // Go identifier for the constant (e.g. NSOrderedAscending)
	Value        string // Raw value expression (e.g. "-1", "0x1")
	ObjCName     string // Original ObjC name; used as the string in String() and Parse()
	CommentBlock string // Pre-rendered comment lines with "\t" prefix (may be empty)
	IsZeroVal    bool   // Bitmask enums: this member represents the zero-value case
}

// enumModel is template data for a named ObjC enum.
type enumModel struct {
	GoName        string            // Go type name (e.g. NSComparisonResult)
	GoType        string            // Underlying Go type (e.g. int64, uint64)
	IsBitmask     bool              // Bitmask enums use | combination, not switch
	HasMembers    bool              // Whether helper methods (String, Parse, etc.) should be emitted
	CommentBlock  string            // Pre-rendered type-level comment lines (may be empty)
	Members       []enumMemberModel // All members for the const block
	UniqueMembers []enumMemberModel // Deduped members for the String() switch (non-bitmask only)
	FirstConst    string            // First constant name; seeds result in ParseX (non-bitmask only)
	DefaultFmt    string            // Format string for String() default case, e.g. "NSFoo(%d)"
}

// ─── Externs ──────────────────────────────────────────────────────────────────

// externItemModel is template data for one extern var declaration.
type externItemModel struct {
	GoName       string // Go identifier
	GoType       string // Go type string
	CommentBlock string // Pre-rendered comment lines with "\t" prefix (may be empty)
}

// externsFileModel is template data for a complete _externs.go file.
type externsFileModel struct {
	PkgName string
	Imports []string
	Items   []externItemModel
}

// ─── Structs ──────────────────────────────────────────────────────────────────

// structFieldModel is template data for one field in a C struct.
type structFieldModel struct {
	Name   string
	GoType string
}

// structModel is template data for a regular C struct type definition.
type structModel struct {
	GoName       string
	CommentBlock string
	Fields       []structFieldModel
}

// classStructModel is template data for an ObjC class's Go struct: a root class
// owns the unsafe.Pointer field and a promoted Ptr() accessor, while a non-root
// class embeds its immediate superclass by value. Both emit a compile-time
// cgo.Object interface assertion.
type classStructModel struct {
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

// classConstructorsModel is template data for a class's core constructors: the
// tracked New<Class> pointer constructor, the untracked <Class>WithPtr value
// constructor (for embedding in subclass literals), and the Cast<Class> helper.
type classConstructorsModel struct {
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

// designatedInitModel is template data for one New<Class>With<Arg> factory
// generated from a designated initializer. The factory allocates via the class
// alloc bridge, wraps the result untracked, calls the Go init method, and
// converts its result according to Kind.
type designatedInitModel struct {
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

// methodBodyModel is template data for a CGo method body: the keep-alive
// defers, argument preambles, the C call, exception/NSError handling, and the
// return conversion selected by RetKind.
type methodBodyModel struct {
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

// classMethodModel is template data for one generated instance or class method:
// the doc/annotation preamble, the bridge-ID comment, the signature, and the
// resolved CGo body.
type classMethodModel struct {
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
	Body methodBodyModel
	// Skip renders only the preamble comment (no function): a class method whose
	// generated name collides with a package-level type is dropped, but — matching
	// the original — its doc comment is still emitted.
	Skip bool
}

// codingMethodsModel is template data for the NSSecureCoding/NSCoding archive
// convenience methods (SerializeToArchive + New<Class>FromArchive).
type codingMethodsModel struct {
	Name          string
	SerializeFn   string
	DeserializeFn string
}

// genericHelperModel is template data for the New<Class>T[T] typed constructor a
// generic class exposes so generic subclasses can build it with a preserved type
// parameter (rather than the cgo.Object instantiation).
type genericHelperModel struct {
	Name   string
	TChain string // the &Class[T]{…} value chain
}

// typedWithPtrModel is template data for the <Class>TypedWithPtr[T] untracked
// value constructor used by generic subclasses in other packages.
type typedWithPtrModel struct {
	Name        string
	TValueChain string // the Class[T]{…} value chain (no leading &)
}

// nsStringOverloadModel is template data for a "…Go" Go-string convenience
// overload: a thin wrapper that converts its Go-string arguments and forwards to
// the underlying generated method/function.
type nsStringOverloadModel struct {
	// Signature is the full function signature up to the opening brace.
	Signature string
	// HasReturn is true when the wrapped call's result is returned.
	HasReturn bool
	// CallExpr is the forwarding call into the underlying method/function.
	CallExpr string
}

// cfWrapperModel is template data for a CoreFoundation opaque pointer wrapper type.
type cfWrapperModel struct {
	GoName string // Primary type name, e.g. CFStringRef
}

// cfAliasModel is template data for a CF typedef alias pointing to a primary wrapper.
type cfAliasModel struct {
	GoAlias   string // e.g. CFMutableStringRef
	GoPrimary string // e.g. CFStringRef
}

// structTypedefModel is template data for a non-CF struct typedef alias.
type structTypedefModel struct {
	GoAlias  string
	GoTarget string
}

// ─── Protocols ────────────────────────────────────────────────────────────────

// protocolMethodModel is template data for one method in a protocol interface.
type protocolMethodModel struct {
	GoName string // Go method name
	Params string // Full parameter list string of mapped ObjC args
	Ret    string // Return type string, empty for void
}

// protocolModel is template data for one ObjC @protocol → Go interface.
type protocolModel struct {
	GoName        string
	ObjCName      string
	AvailComment  string   // e.g. "Introduced: macOS 10.9" — empty when not set
	Embeds        []string // Embedded interface identifiers (same-fw and cross-fw)
	EmbedComments []string // Comment lines for blocked cross-fw embeds
	Methods       []protocolMethodModel
}

// protocolsFileModel is template data for a complete _protocols.go file.
type protocolsFileModel struct {
	PkgName   string
	Imports   []string
	Protocols []protocolModel
}

// ─── Functions ────────────────────────────────────────────────────────────────

// functionModel is template data for one C function → Go wrapper.
type functionModel struct {
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

// functionsFileModel is template data for a complete _functions.go file.
type functionsFileModel struct {
	PkgName   string
	FwLower   string
	Imports   []string
	Functions []functionModel
}

// ─── Classes ──────────────────────────────────────────────────────────────────

// classFileHeaderModel is template data for the generated header section of a class file:
// the generated-code comment, build tag, package declaration, CGo include, and import block.
type classFileHeaderModel struct {
	Framework    string   // e.g. "Foundation"
	PkgName      string   // e.g. "foundation"
	BridgeHeader string   // e.g. "foundation_bridge.h"
	Imports      []string // sorted, deduplicated import paths
}

// ─── Bridge ───────────────────────────────────────────────────────────────────

// bridgeDeclModel is template data for one C function declaration in a bridge header.
type bridgeDeclModel struct {
	Entitlements []string // // Requires entitlement: ... lines
	BridgeID     string   // e.g. "Foundation.NSString/stringWithCString:encoding:"
	Decl         string   // complete C declaration, without trailing semicolon
}

// bridgeAllocModel is template data for one alloc helper (declaration and implementation).
type bridgeAllocModel struct {
	ClassName string // ObjC class name, e.g. "NSWindow"
	FuncName  string // C function name, e.g. "foundation_NSWindow_alloc"
}

// bridgeProtoDeclModel is template data for a protocol proxy bridge declaration.
type bridgeProtoDeclModel struct {
	BridgeID string
	RetType  string
	FuncName string
	Params   string
}

// bridgeHeaderModel is template data for a complete _bridge.h file.
type bridgeHeaderModel struct {
	MethodDecls   []bridgeDeclModel
	AllocHelpers  []bridgeAllocModel
	CodingDecls   string // pre-rendered NSCoding declarations block
	FunctionDecls []bridgeDeclModel
	ForeignDecls  []bridgeDeclModel
	ProtoDecls    []bridgeProtoDeclModel
}

// bridgeImplMethodModel is template data for one ObjC bridge function implementation.
type bridgeImplMethodModel struct {
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

// bridgeAllocImplModel is template data for one alloc helper implementation.
type bridgeAllocImplModel struct {
	ClassName string
	FuncName  string
}

// bridgeImplModel is template data for a complete _bridge.m file.
type bridgeImplModel struct {
	Framework       string
	ParentFramework string
	// UmbrellaHeader is the C library umbrella include path relative to the
	// SDK's usr/include (e.g. "compression.h", "os/log.h"). When non-empty it
	// replaces the framework-style <Name/Name.h> import.
	UmbrellaHeader string
	HeaderName     string
	Methods        []bridgeImplMethodModel
	AllocImpls     []bridgeAllocImplModel
	CodingImpls    string // pre-rendered NSCoding implementation block
	Functions      []bridgeImplMethodModel
	ForeignMethods []bridgeImplMethodModel
	ProtoMethods   []bridgeImplMethodModel
}

// bridgeShimModel is template data for the package-root _bridge_impl.m shim file.
// This file includes the real bridge implementation so CGo can find the symbols.
type bridgeShimModel struct {
	Stem string // e.g. "foundation_bridge" (produces bridge/foundation_bridge.m include)
}

// ─── Blocks ───────────────────────────────────────────────────────────────────

// blockTypeModel is template data for one ObjC block typedef → Go named func type alias.
type blockTypeModel struct {
	GoName    string // Go type alias name, e.g. CompletionBlock
	Framework string // Framework name for the doc comment, e.g. Foundation
	Sig       string // Complete Go func signature, e.g. func(result uint64, ok bool)
}

// ─── Interfaces ───────────────────────────────────────────────────────────────

// interfaceMethodModel is template data for one method in a [ClassName]able interface.
type interfaceMethodModel struct {
	GoName string // Go method name
	Params string // Full parameter list of mapped ObjC args
	Ret    string // Return type string, empty for void
}

// classInterfaceModel is template data for one [ClassName]able Go interface.
type classInterfaceModel struct {
	GoName      string // ObjC class name; the interface is GoName+"able"
	EmbedLine   string // Embedded type: "cgo.Object", "pkg.Superabled", or "Superable"
	Methods     []interfaceMethodModel
	HasNSCoding bool
}

// classInterfacesFileModel is template data for a complete _interfaces.go file.
type classInterfacesFileModel struct {
	PkgName    string
	Imports    []string // sorted, deduplicated import paths
	UsesUnsafe bool     // true when any method references unsafe.Pointer
	UsesObjc   bool     // true when any method or embed references cgo.*
	Interfaces []classInterfaceModel
}

// ─── Variadic Wrappers ────────────────────────────────────────────────────────

// variadicWrappersFileModel is template data for the Foundation variadic wrappers file.
type variadicWrappersFileModel struct {
	PkgName string // Go package name, e.g. "foundation"
}

// ─── Block Trampolines ────────────────────────────────────────────────────────

// blockTrampolineSigModel holds all pre-rendered strings for one block signature.
// It is used by the block_trampolines_go_file, block_trampolines_h_file, and
// block_trampolines_m_file templates.
type blockTrampolineSigModel struct {
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

// blockTrampolinesGoFileModel is template data for blocks_generated.go.
type blockTrampolinesGoFileModel struct {
	PkgName string
	Sigs    []blockTrampolineSigModel
}

// blockTrampolinesHFileModel is template data for block_trampolines_generated.h.
type blockTrampolinesHFileModel struct {
	Sigs []blockTrampolineSigModel
}

// blockTrampolinesMFileModel is template data for block_trampolines_generated.m.
type blockTrampolinesMFileModel struct {
	Sigs []blockTrampolineSigModel
}

// ─── Method Trampolines ───────────────────────────────────────────────────────

// methodTrampolineSigModel holds all pre-rendered strings for one method IMP signature.
// It is used by the method_trampolines_go_file, method_trampolines_h_file, and
// method_trampolines_m_file templates.
type methodTrampolineSigModel struct {
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

// methodTrampolinesGoFileModel is template data for callbacks_generated.go.
type methodTrampolinesGoFileModel struct {
	PkgName string
	Sigs    []methodTrampolineSigModel
}

// methodTrampolinesHFileModel is template data for method_trampolines_generated.h.
type methodTrampolinesHFileModel struct {
	Sigs []methodTrampolineSigModel
}

// methodTrampolinesMFileModel is template data for method_trampolines_generated.m.
type methodTrampolinesMFileModel struct {
	Sigs []methodTrampolineSigModel
}

// ─── Subclass Factories ───────────────────────────────────────────────────────

// subclassGoFileModel is template data for a *_subclass_generated.go file.
type subclassGoFileModel struct {
	ClassName       string   // e.g. "NSCharacterSet"
	GoClassName     string   // e.g. "GoOrinSubNSCharacterSet"
	PackageName     string   // e.g. "foundation"
	IMPSigs         []string // sorted unique IMP sig names
	OverridesFields string   // pre-rendered struct field lines (tab-indented, ends with \n)
	AddMethodBody   string   // pre-rendered per-method add_method if-blocks (inside "if overrides != nil")
	BindMethodBody  string   // pre-rendered per-method BindMethod if-blocks (inside "if overrides != nil")
}

// subclassHFileModel is template data for a *_subclass_generated.h file.
type subclassHFileModel struct {
	GoClassName string
	IMPSigs     []string
}

// subclassMFileModel is template data for a *_subclass_generated.m file.
type subclassMFileModel struct {
	Framework   string // e.g. "Foundation"
	ClassName   string // e.g. "NSCharacterSet"
	GoClassName string // e.g. "GoOrinSubNSCharacterSet"
	HeaderFile  string // e.g. "NSCharacterSet_subclass_generated.h"
	IMPSigs     []string
}

// ─── Protocol Impls ───────────────────────────────────────────────────────────

// protocolImplGoFileModel is template data for a *_protocol_callback.go file.
type protocolImplGoFileModel struct {
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

// protocolImplHFileModel is template data for a *_impl_generated.h file.
type protocolImplHFileModel struct {
	GoClassName string
	IMPSigs     []string
}

// protocolImplMFileModel is template data for a *_impl_generated.m file.
type protocolImplMFileModel struct {
	Framework   string
	ProtoName   string
	GoClassName string
	HeaderFile  string // e.g. "NSCacheDelegate_impl_generated.h"
	IMPSigs     []string
}

// ─── Protocol Proxies ─────────────────────────────────────────────────────────

// protocolProxiesFileModel is template data for a complete _protocol_proxies.go file.
type protocolProxiesFileModel struct {
	PkgName      string
	BridgeHeader string   // e.g. "foundation_bridge.h"
	Imports      []string // sorted, deduplicated import paths
	ProxyTypes   []protocolProxyTypeModel
}

// protocolProxyMethodModel is template data for one method on a protocol proxy type.
type protocolProxyMethodModel struct {
	GoName       string // Go method name
	Params       string // full parameter list of mapped ObjC args
	Ret          string // return type string, empty for void
	AvailComment string // availability comment, empty if none
	BodyLines    string // pre-rendered method body lines (tab-indented)
}

// protocolProxyTypeModel is template data for one id<Protocol> wrapper type.
type protocolProxyTypeModel struct {
	IDTypeName      string // e.g. "NSCacheDelegateIDProtocol"
	ProtoName       string // e.g. "NSCacheDelegate"
	NSObjectEmbed   string // "NSObject" or "foundation.NSObject"
	NSObjectWithPtr string // "NSObjectWithPtr" or "foundation.NSObjectWithPtr"
	AvailComment    string // availability comment, empty if none
	Methods         []protocolProxyMethodModel
}
