package view

// FunctionFile is the whole rendered functions output for a package: the
// `var (...)` block of purego-bound C function pointers followed by the exported
// wrapper functions.
type FunctionFile struct {
	Vars     []FunctionVar
	Wrappers []Function
}

// FunctionVar is one entry in the function-pointer var block: the binding
// variable and its C-ABI func type. Its CommentBlock matches the original
// emitter's indentation (the doc line is tab-indented inside the var block; a
// deprecation line is at column 0 and gofmt re-indents it).
type FunctionVar struct {
	CommentBlock string
	VarName      string
	FuncType     string
}

// Function is an exported wrapper around a C function pointer. Block-typed
// parameters are adapted by Adapters before the call; the result is returned
// directly, returned after a retain (object returns), or discarded (void).
type Function struct {
	// CommentBlock is the doc, "// C function: <name>", and deprecation comment
	// (all column 0, trailing newline).
	CommentBlock string
	GoName       string
	ParamStr     string
	RetSig       string
	// Adapters build the objc.Block values for block-typed parameters.
	Adapters []BlockAdapter
	// FuncVarName is the bound C function-pointer variable.
	FuncVarName string
	// CallStr is the comma-joined call argument list.
	CallStr string
	// ReturnKind selects the body shape: 0 void, 1 plain return, 2 object
	// return (retain then wrap).
	ReturnKind int
	// WrapExpr is the wrapper expression for an object return (ReturnKind 2).
	WrapExpr string
}
