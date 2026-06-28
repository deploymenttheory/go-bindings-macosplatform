package view

// ClassTypeDecl is the resolved Go struct declaration for an ObjC class: the doc
// comment block, the (possibly generic) type header, the single embedded field
// (the raw ptr for a root/cycle-broken class, otherwise the superclass struct),
// and — for those root/cycle-broken classes — the promoted Ptr/InitPtr methods.
type ClassTypeDecl struct {
	// CommentBlock is the class doc, the Apple documentation URL, and any
	// deprecation line (all column 0, trailing newline).
	CommentBlock string
	// TypeHeader is the type name with any generic constraint list,
	// e.g. "NSView" or "NSArray[ObjectType objc.AnyObject]".
	TypeHeader string
	// EmbedLine is the struct's single line: "ptr objc.ID" for a root or
	// cycle-broken class, otherwise the embedded superclass type.
	EmbedLine string
	// EmitPtrMethods emits the promoted Ptr()/InitPtr() accessors (only for
	// root/cycle-broken classes that hold the ptr field directly).
	EmitPtrMethods bool
	// PtrReceiver is the receiver type for the Ptr/InitPtr methods, with any
	// generic parameter list, e.g. "NSView" or "NSArray[ObjectType]".
	PtrReceiver string
}

// ClassVars is the resolved package-level var block for a class: the class
// reference variable and one registered-selector variable per selector used by
// the class's methods.
type ClassVars struct {
	// ClassVarName is the unexported class-reference variable (e.g. "_clsNSView").
	ClassVarName string
	// ClassName is the ObjC class name passed to _objcClass.
	ClassName string
	// Selectors are the registered-selector variables, deduplicated by name; an
	// empty list collapses to a single `var` declaration.
	Selectors []ClassSelectorVar
}

// ClassSelectorVar is one registered-selector variable in a class's var block.
type ClassSelectorVar struct {
	// VarName is the unexported selector variable name.
	VarName string
	// Selector is the ObjC selector string passed to objc.RegisterName.
	Selector string
}

// FromIDConstructor is the resolved XFromID factory: it wraps a raw objc.ID in
// the class's Go type and registers it with the runtime for finalization.
type FromIDConstructor struct {
	// Signature is the full function signature up to the opening brace, e.g.
	// "func NSViewFromID(id objc.ID) *NSView" or the generic form.
	Signature string
	// AllocType is the type literal allocated for the wrapper, e.g. "NSView" or
	// "NSArray[ObjectType]".
	AllocType string
}

// RawMethod is a resolved Go method (or class function) wrapping an ObjC method
// via objc.Send. Block parameters are adapted first; the call dispatches on the
// resolved return kind, optionally unpacking a trailing NSError.
type RawMethod struct {
	// CommentBlock is the doc + deprecation comment (column 0, trailing newline).
	CommentBlock string
	// Receiver is the method receiver clause "(o *Type) " (empty for a class
	// function).
	Receiver string
	// GoName is the exported method or function name.
	GoName string
	// ParamStr is the rendered signature parameter list.
	ParamStr string
	// ReturnSig is the return clause: "", " T", " error", or " (T, error)".
	ReturnSig string
	// Adapters build objc.Block values for block-typed parameters.
	Adapters []BlockAdapter
	// HasNSError adds the trailing NSError out-parameter and error handling.
	HasNSError bool
	// Target is the receiver expression of the objc.Send ("o.Ptr()" or the class
	// reference for class methods).
	Target string
	// SelVar is the registered-selector variable passed to objc.Send.
	SelVar string
	// SendArgStr is the marshaled argument list, already comma-prefixed (or empty).
	SendArgStr string
	// ReturnKind selects the dispatch shape: 0 void, 1 object, 2 string, 3 bool,
	// 4 scalar/struct.
	ReturnKind int
	// RetGoType is the Go return type (the objc.Send type parameter for the
	// scalar/struct kind).
	RetGoType string
	// AlreadyRetained skips the post-call retain for object returns the callee
	// already retained (NARC).
	AlreadyRetained bool
	// WrapExpr wraps the raw objc.ID result for an object return (kind 1).
	WrapExpr string
	// ZeroVal is the zero value returned on the error path for a scalar/struct
	// return (kind 4 with NSError).
	ZeroVal string
}
