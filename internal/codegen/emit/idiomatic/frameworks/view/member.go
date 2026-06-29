package view

// Enum is a named Objective-C enumeration re-emitted as a concrete Go type (a
// `type X <underlying>` declaration with a typed const block and a String
// method), not an alias to the raw bindings. Every naming, dedup, and comment
// decision is already made by the gather phase and recorded here; the render
// phase turns it into Go source through the enum template alone.
type Enum struct {
	// GoName is the de-prefixed exported Go type name (VZVirtualMachineState ->
	// VirtualMachineState).
	GoName string
	// GoType is the underlying integer type (for example int64 or uint).
	GoType string
	// IsBitmask is true when the values are option flags meant to be combined
	// with |, which selects the bitwise String rendering.
	IsBitmask bool
	// CommentBlock is the fully-rendered doc + deprecation comment for the type,
	// already prefixed with "// " on every line (empty when undocumented).
	CommentBlock string
	// Members are the enum constants in declaration order, deduplicated by
	// (name, value); they populate the const block.
	Members []EnumMember
	// UniqueMembers are the members deduplicated by value alone; they populate
	// the non-bitmask String switch so two names for one value do not produce a
	// duplicate case.
	UniqueMembers []EnumMember
	// DefaultFmt is the fmt verb string for the String default branch (for
	// example "VirtualMachineState(%d)").
	DefaultFmt string
}

// Constant is a global constant value re-emitted as an idiomatic accessor
// function. The gather phase resolves which Kind the symbol's value is produced
// as; the constant template renders the matching accessor.
type Constant struct {
	// GoName is the exported Go accessor name.
	GoName string
	// ExternName is the underlying framework symbol whose address is read.
	ExternName string
	// CommentBlock is the rendered doc comment (already "// …\n"), or empty.
	CommentBlock string
	// Kind selects how the symbol's value is produced and typed.
	Kind ConstKind
	// GoTypeName is the idiomatic wrapper type name; set only for ConstObjcClass.
	GoTypeName string
}

// ConstKind says how a Constant accessor produces and types its value.
// It is a string so templates can compare against readable labels rather than
// opaque integers.
type ConstKind string

const (
	// ConstCFRef is a CoreFoundation reference value (const CF<T>Ref) returned as
	// an obj.Object.
	ConstCFRef ConstKind = "cfref"
	// ConstNSString is an NSString* global returned as the package-local *String
	// wrapper (Foundation only).
	ConstNSString ConstKind = "nsstring"
	// ConstObjcID is an NSString* global in a non-Foundation package, returned as
	// an obj.Object usable as a dictionary key or argument.
	ConstObjcID ConstKind = "objcid"
	// ConstObjcClass is a same-framework ObjC class pointer global returned as
	// the idiomatic wrapper type for that class.
	ConstObjcClass ConstKind = "objcclass"
)

// ErrorSentinel is a named error value for one member of a framework's
// error-code enum, so callers can match a returned error with errors.Is.
type ErrorSentinel struct {
	// GoName is the exported sentinel variable name (for example ErrInternalError).
	GoName string
	// CommentBlock is the rendered doc comment (already "// …\n").
	CommentBlock string
	// Domain is the NSError domain string the framework reports.
	Domain string
	// Code is the integer error code as a Go literal string.
	Code string
}

// EnumMember is one constant of an Enum.
type EnumMember struct {
	// ConstName is the de-prefixed exported constant name.
	ConstName string
	// Value is the integer value as a Go literal string.
	Value string
	// CommentBlock is the rendered doc + deprecation comment for the member,
	// indented inside the const block (empty when undocumented).
	CommentBlock string
	// IsZeroVal is true when Value is "0"; the bitmask String rendering skips
	// zero-valued members because they contribute no flag bit.
	IsZeroVal bool
}
