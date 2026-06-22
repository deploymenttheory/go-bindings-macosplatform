package view

// Method is a resolved Go method (or package-level function) that calls a single
// Objective-C method. The signature pieces are pre-rendered by the gather phase;
// the body is described by Dispatch and rendered by the method template alone.
type Method struct {
	// DocComment is the fully-rendered doc comment block (already "// "-prefixed,
	// trailing newline), or empty when undocumented.
	DocComment string
	// Recv is the receiver clause "(x *Type) " for an instance method, or "" for
	// a package-level class function.
	Recv string
	// GoName is the exported Go method name.
	GoName string
	// ParamStr is the rendered signature parameter list (out-parameters omitted).
	ParamStr string
	// RetSig is the rendered return clause: "", " T", or " (T, error)".
	RetSig string
	// Dispatch is the method body (call + result/error/out-parameter handling).
	Dispatch Dispatch
}

// AsyncMethod is an Objective-C completion-handler method surfaced as a blocking,
// ctx-aware Go call. The completion block feeds a buffered channel of size 1 so
// it can always send and never leaks, and the outer function selects between the
// channel and ctx.Done. All marshaling expressions are resolved by gather; the
// template assembles the block and select with no string-built Go statements.
type AsyncMethod struct {
	// DocComment is the rendered doc comment block, or empty.
	DocComment string
	// Recv is the receiver clause "(x *Type) " or "" for a class function.
	Recv string
	// GoName is the exported Go method name.
	GoName string
	// ParamStr is the signature parameter list (always begins with
	// "ctx context.Context").
	ParamStr string
	// HasResult is true when the block carries one non-error value, so the
	// wrapper returns (ResultGoType, error) instead of plain error.
	HasResult bool
	// ResultGoType is the wrapped Go result type (set only when HasResult).
	ResultGoType string
	// ClosureParams are the completion block's Go parameter declarations, in
	// order (for example "_p0 objc.ID").
	ClosureParams []string
	// SendCall is the objc.Send expression that hands the block to Objective-C
	// (it references the local _block); an expression, not a declaration.
	SendCall string
	// ErrConvExpr is the right-hand side that converts the block's NSError
	// parameter to a Go error, or "" when the block has no error parameter.
	ErrConvExpr string
	// ResultConvExpr is the right-hand side that converts the block's result
	// parameter to ResultGoType (set only when HasResult).
	ResultConvExpr string
}

// SliceMethod is an Objective-C array-returning getter surfaced as a Go method
// that returns a slice, converting each element with ConvClosure.
type SliceMethod struct {
	// DocComment is the rendered doc comment block, or empty.
	DocComment string
	// Recv is the receiver clause "(x *Type) " or "".
	Recv string
	// GoName is the exported Go method name.
	GoName string
	// RecvExpr is the object the Objective-C call is sent to.
	RecvExpr string
	// Selector is the Objective-C selector being sent.
	Selector string
	// ElemGoType is the slice element's Go type.
	ElemGoType string
	// HasError is true when the underlying getter reports an NSError, so the
	// wrapper returns ([]T, error).
	HasError bool
	// ConvClosure is the per-element conversion closure expression
	// (func(_id objc.ID) T { return … }).
	ConvClosure string
}

// BoolNSErrorMethod is an Objective-C method returning a success flag plus an
// NSError out-parameter, surfaced as a Go method that returns only error (nil on
// success).
type BoolNSErrorMethod struct {
	// DocComment is the rendered doc comment block, or empty.
	DocComment string
	// Recv is the receiver clause "(x *Type) " or "".
	Recv string
	// GoName is the exported Go method name.
	GoName string
	// RecvExpr is the object the Objective-C call is sent to.
	RecvExpr string
	// Selector is the Objective-C selector being sent.
	Selector string
}
