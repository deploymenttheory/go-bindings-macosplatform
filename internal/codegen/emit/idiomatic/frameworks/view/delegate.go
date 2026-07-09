package view

// Delegate is one Objective-C delegate protocol surfaced as a Go interface
// plus the shim builder that lets a plain Go value act as that delegate. The
// gather phase (protocols.go) resolves every name, signature, and conversion
// expression; the delegate template renders the interface declarations, the
// once-registered shim class, and the wrap function with no string-built Go.
type Delegate struct {
	// DocComment is the rendered doc comment for the interface type.
	DocComment string
	// IfaceName is the Go interface name (VirtualMachineDelegate).
	IfaceName string
	// ProtocolName is the Objective-C protocol (VZVirtualMachineDelegate).
	ProtocolName string
	// ShimClassName is the runtime-registered Objective-C class name; globally
	// unique (GoShimVirtualizationVirtualMachineDelegate).
	ShimClassName string
	// ShimFuncName is the unexported builder that wraps a Go value
	// (newVirtualMachineDelegateShim).
	ShimFuncName string
	// ClassVar is the package-level variable base holding the registered class
	// and its sync.Once (_virtualMachineDelegateShimClass).
	ClassVar string
	// Required are the protocol's required methods — the interface's method
	// set. Optional are the @optional methods, each carried by its own
	// one-method upgrade interface.
	Required []DelegateMethod
	Optional []DelegateMethod
}

// DelegateMethod is one bridgeable protocol method: its Go interface
// signature, and the pieces of the Objective-C callback that routes the
// invocation to the Go value.
type DelegateMethod struct {
	// DocComment is the rendered doc line(s) for the interface method (already
	// "// "-prefixed, trailing newline), or empty.
	DocComment string
	// OptionalDoc is the rendered doc comment for an optional method's upgrade
	// interface type (empty for required methods).
	OptionalDoc string
	// OptIfaceName is the one-method upgrade interface for an optional method
	// (VirtualMachineDelegateGuestDidStopHandler); empty for required methods.
	OptIfaceName string
	// GoName is the Go method name.
	GoName string
	// Selector is the Objective-C selector this method answers.
	Selector string
	// SigParams is the Go interface signature parameter list.
	SigParams string
	// RetSig is the Go return clause (" bool", "" for void).
	RetSig string
	// AssertIface is the interface the callback type-asserts shim.Value to —
	// the delegate interface for required methods, OptIfaceName for optional.
	AssertIface string
	// ABIParams are the callback's parameter declarations after (self, _cmd),
	// in order ("_p0 objc.ID", "_p1 int").
	ABIParams []string
	// ABIRet is the callback's return type ("" for void).
	ABIRet string
	// CallArgs are the converted argument expressions passed to the Go method,
	// parallel to the interface signature.
	CallArgs []string
	// RetExpr converts the Go method call's result into the callback's return
	// value (set only when RetSig is non-empty); it embeds the full
	// _h.<GoName>(…) call.
	RetExpr string
	// RetZero is the callback's zero return when the Go value does not
	// implement the method (set only when RetSig is non-empty).
	RetZero string
}
