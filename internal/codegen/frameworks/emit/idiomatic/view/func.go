package view

// Func is a C-function wrapper: a lazily-bound Go function over a framework C
// symbol. Every wrapper shares a prologue (a package-level bound-function
// variable and a one-time RegisterLibFunc) and differs only in how the result
// and any error are produced, which Kind selects. The gather phase resolves all
// expressions; the cfunc template renders the body with no string-built Go.
type Func struct {
	// GoName is the exported Go function name.
	GoName string
	// CName is the underlying C symbol bound via RegisterLibFunc.
	CName string
	// VarName is the package-level bound-function variable (for example _fnFoo).
	VarName string
	// CommentLine is the rendered doc comment line (already "// …\n").
	CommentLine string
	// CommentFirst places CommentLine before the bound-function variable rather
	// than before the func declaration. The CFErrorRef wrappers document the var;
	// the generic and OSStatus wrappers document the func.
	CommentFirst bool
	// ABIParams are the C ABI parameter types for the bound-function variable.
	ABIParams []string
	// ABIRet is the C ABI return type ("" for void).
	ABIRet string
	// SigParams are the Go signature parameters ("name Type").
	SigParams []string
	// RetSig is the Go return clause, including a leading space when non-empty
	// (" error", " (obj.Object, error)", " int", "").
	RetSig string
	// Kind selects the body tail.
	Kind FuncKind
	// PreLines are statements emitted after the bind prologue and before the call
	// (out-parameter declarations, or a CFErrorRef holder).
	PreLines []string
	// Call is the bound-function call expression (for example _fnFoo(a, b)).
	Call string
	// Wrap is the object-result conversion template (one %s); used by FuncObject.
	Wrap string
	// Outs are pointer out-parameters lifted from the signature into extra return
	// values, in declaration order. When non-empty the body renders var decls for
	// each, passes &local to the call, and returns the function's own result (per
	// Kind) followed by each out value; RetSig is then the parenthesised tuple.
	Outs []DispatchOut
	// FailRet is the error-path return list for FuncOSStatus.
	FailRet string
	// OkRet is the success return list for FuncOSStatus.
	OkRet string
	// Fail is the failure condition for FuncCFErrorBool (for example "!_ok").
	Fail string
}

// FuncKind selects how a Func wrapper turns its bound call into a Go result.
type FuncKind int

const (
	// FuncVoid discards the result.
	FuncVoid FuncKind = iota
	// FuncString converts a non-nil result pointer to a Go string.
	FuncString
	// FuncObject wraps the result pointer via Wrap.
	FuncObject
	// FuncScalar returns the raw result.
	FuncScalar
	// FuncOSStatus turns an OSStatus return into a Go error.
	FuncOSStatus
	// FuncCFErrorBool turns a BOOL + CFErrorRef out-parameter into a Go error.
	FuncCFErrorBool
	// FuncCFErrorPtr turns a pointer return + CFErrorRef out-parameter into
	// (obj.Object, error).
	FuncCFErrorPtr
)
