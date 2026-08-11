//go:build darwin

package purego

// This file re-exports the ObjC dynamic-dispatch surface from
// github.com/ebitengine/purego/objc so that consumers of the generated bindings
// can perform raw message-sends, register classes, and build blocks without
// importing the third-party package directly. Combined with the conversion
// helpers in this package (NSString, GoString, NSErrorToError, Retain, …), the
// rule for external code becomes: import only bindings/runtime/... .
//
// Types are aliases (not new defined types) so they stay identical to the
// objc.* types that appear in generated binding signatures — a *foundation.NSString
// argument typed as objc.ID and a purego.ID are the same type.

import "github.com/ebitengine/purego/objc"

// Dispatch type aliases.
type (
	ID                = objc.ID
	SEL               = objc.SEL
	Class             = objc.Class
	Protocol          = objc.Protocol
	Block             = objc.Block
	IMP               = objc.IMP
	MethodDef         = objc.MethodDef
	FieldDef          = objc.FieldDef
	Ivar              = objc.Ivar
	MethodDescription = objc.MethodDescription
	Property          = objc.Property
)

// Send sends sel to id with args and returns the result typed as T.
func Send[T any](id ID, sel SEL, args ...any) T { return objc.Send[T](id, sel, args...) }

// SendSuper sends sel to id's superclass implementation and returns the result typed as T.
func SendSuper[T any](id ID, sel SEL, args ...any) T { return objc.SendSuper[T](id, sel, args...) }

// InvokeBlock invokes block with args and returns the result typed as T.
func InvokeBlock[T any](block Block, args ...any) (T, error) {
	return objc.InvokeBlock[T](block, args...)
}

// RegisterName returns the SEL for the given selector name, registering it if needed.
func RegisterName(name string) SEL { return objc.RegisterName(name) }

// GetClass returns the registered class with the given name.
func GetClass(name string) Class { return objc.GetClass(name) }

// GetProtocol returns the registered protocol with the given name.
func GetProtocol(name string) *Protocol { return objc.GetProtocol(name) }

// AllocateProtocol creates a new protocol with the given name.
func AllocateProtocol(name string) *Protocol { return objc.AllocateProtocol(name) }

// NewBlock wraps a Go func as an ObjC block.
func NewBlock(fn any) Block { return objc.NewBlock(fn) }

// NewIMP wraps a Go func as an ObjC method implementation.
func NewIMP(fn any) IMP { return objc.NewIMP(fn) }

// RegisterClass registers a new ObjC class with the given superclass, protocols,
// ivars, and methods.
func RegisterClass(name string, superClass Class, protocols []*Protocol, ivars []FieldDef, methods []MethodDef) (Class, error) {
	return objc.RegisterClass(name, superClass, protocols, ivars, methods)
}
