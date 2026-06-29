package emit

import (
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// MethodWillBeEmitted reports whether an ObjC method will appear in the
// generated purego bindings. It mirrors isMethodBridgeable in classes.go —
// any changes to the emitter's skip rules must be reflected here.
//
// It does not account for name-collision avoidance (classMethodName), which
// silently drops class methods whose derived Go name would collide with a
// package-level type. Those cases are rare and impossible to detect without a
// full registry load; callers of this function must tolerate rare false
// positives (a function name that was ultimately not emitted).
func MethodWillBeEmitted(method meta.Method) bool {
	return isMethodBridgeable(method)
}

// ClassMethodGoName returns the expected exported Go function name for an ObjC
// class method. The result is className + MethodName(selector), which matches
// the emitter's output for the common case. Name-collision avoidance may
// produce a different name in rare cases; tests generated from this name
// should be treated as best-effort.
//
//	("NSBundle", "mainBundle")        → "NSBundleMainBundle"
//	("NSProcessInfo", "processInfo")  → "NSProcessInfoProcessInfo"
func ClassMethodGoName(className, selector string) string {
	return className + naming.MethodName(selector)
}

// ClassMethodGoNameFromMeta is like ClassMethodGoName but also applies the
// same name-collision avoidance the emitter uses: if the derived candidate
// collides with an enum, struct, or class name in the same framework, a
// "Class" suffix is appended (matching classMethodName in classes.go).
//
//	("SFSpeechRecognizer", "authorizationStatus", m) → "SFSpeechRecognizerAuthorizationStatusClass"
//	("CBManager",          "authorization",        m) → "CBManagerAuthorizationClass"
func ClassMethodGoNameFromMeta(className, selector string, m *meta.FrameworkMeta) string {
	candidate := className + naming.MethodName(selector)
	if _, ok := m.Enums[candidate]; ok {
		return candidate + "Class"
	}
	if _, ok := m.Structs[candidate]; ok {
		return candidate + "Class"
	}
	if _, ok := m.Classes[candidate]; ok {
		return candidate + "Class"
	}
	return candidate
}

// ReturnIsVoid reports whether a method's return type is void.
func ReturnIsVoid(retType meta.ReturnType) bool {
	t := strings.TrimSpace(retType.ObjCType)
	return t == "" || t == "void"
}

// FunctionWillBeEmitted reports whether a free C function will appear in the
// generated purego bindings as an exported Go symbol.
//
// It does not account for exported-name collision skips (seedReservedGoNames
// in EmitFunctions); callers must tolerate rare false positives.
func FunctionWillBeEmitted(fn meta.Function) bool {
	if fn.Availability.IsUnavailable {
		return false
	}
	if fn.IsInline {
		return false
	}
	if fn.IsVariadic {
		return false
	}
	return naming.ExportedFunctionName(fn.Name) != ""
}

// FunctionGoName returns the exported Go name the emitter uses for a free C
// function (snake_case names become PascalCase; already-exported C names are
// unchanged).
func FunctionGoName(fn meta.Function) string {
	return naming.ExportedFunctionName(fn.Name)
}
