package view

// Dispatch is the resolved body of a plain method or package-level function: the
// marshaled Objective-C call expression plus how its result, trailing NSError,
// and value out-parameters become Go return values. The gather phase makes every
// decision; the method_body / dispatch_tail template renders the statements with
// iteration and conditionals only — it never decides a type.
type Dispatch struct {
	// Guards are pre-call statements rendered first as early returns or panics
	// (for example a bounds check before an index accessor); empty for most
	// methods.
	Guards []string
	// Call is the Objective-C call expression, already marshaled
	// (objc.Send[T](recv, sel, args…)); it is an expression, not a declaration.
	Call string
	// Error is true when the selector took a trailing NSError**: the body then
	// emits the `if _nsErr != 0` early return and a trailing nil on success.
	Error bool
	// RetKind selects how Call's result is converted back to Go.
	RetKind RetKind
	// RetWrap is the conversion expression for an object/array result, with one
	// %s placeholder for the raw result pointer (empty for other kinds).
	RetWrap string
	// RetZero is the zero literal returned on the error path before the error
	// (empty for void).
	RetZero string
	// Outs are value out-parameters lifted from the signature into extra return
	// values, in declaration order.
	Outs []DispatchOut
}

// RetKind says how a Dispatch result pointer is turned into the Go return value.
type RetKind int

const (
	// RetVoid is a method with no value return (error may still be present).
	RetVoid RetKind = iota
	// RetScalar returns the raw result unchanged (scalar, bool, or enum).
	RetScalar
	// RetString converts the result to a Go string with a nil-pointer guard.
	RetString
	// RetObject wraps the result pointer via RetWrap (an object or a slice).
	RetObject
)

// DispatchOut is one value out-parameter, declared as a local whose address is
// passed to the call and whose value is returned alongside the method's result.
type DispatchOut struct {
	// GoName is the local variable / return name.
	GoName string
	// GoType is the out-parameter's Go type.
	GoType string
	// Zero is the type's zero literal for the error-path return.
	Zero string
}
