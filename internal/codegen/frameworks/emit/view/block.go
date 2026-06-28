package view

// BlockAdapter is the resolved objc.NewBlock construction for one block-typed
// parameter of a wrapper function or method. A degraded adapter emits nothing
// (the public objc.Block parameter is passed through verbatim); otherwise the
// template builds a block whose callback adapts the ABI values to the public Go
// closure and invokes it.
type BlockAdapter struct {
	// Degraded marks an unbridgeable block: no construction is emitted.
	Degraded bool
	// ParamName is the public closure parameter name (the Go func the caller
	// supplies).
	ParamName string
	// BlockVar is the local objc.Block variable ("__block_<ParamName>").
	BlockVar string
	// CallbackSig is the fully-rendered objc.NewBlock callback signature,
	// e.g. "func(_ objc.Block, blockParam0 objc.ID) bool".
	CallbackSig string
	// Params are the callback's adapted parameters, in order, used to emit the
	// retain guards for ObjC-object arguments.
	Params []BlockCallbackParam
	// HasReturn is true when the block returns a value (the call is returned).
	HasReturn bool
	// CallExpr is the call into the public closure with each ABI value converted
	// to its public form, e.g. "onDone(purego.GoCString(blockParam0))".
	CallExpr string
}

// BlockCallbackParam is one parameter of a block callback, used to emit the
// pre-call retain guard for ObjC-object arguments.
type BlockCallbackParam struct {
	// ArgName is the callback parameter name ("blockParam<i>").
	ArgName string
	// NeedsRetain marks an ObjC-object argument that must be retained before it
	// is wrapped, so the Go finalizer owns the +1.
	NeedsRetain bool
}
