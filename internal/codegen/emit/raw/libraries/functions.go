package rawlib

import (
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// buildFunctionsModel filters eligible functions, maps their types and arguments,
// and collects all imports. The complex return-path dispatch is resolved here
// (in buildFunctionCallBody) so the template stays a structural description.
// EmittableFunctions returns the plain C functions that EmitFunctions emits as Go
// wrappers for framework, in declaration order, after applying the same skip and
// de-duplication rules (inline/variadic/unavailable/builtin/UPP/va_list/by-value
// unknown filters, plus collision with package-level type names). The idiomatic
// library layer uses this so it only ever wraps raw functions that actually exist.
func EmittableFunctions(framework *macosplatformmetadata.FrameworkMeta) []macosplatformmetadata.Function {
	pkgTypeNames := packageTypeNames(framework)

	seen := make(map[string]bool)
	out := make([]macosplatformmetadata.Function, 0, len(framework.Functions))
	for _, fn := range framework.Functions {
		if fn.IsInline || fn.IsVariadic || fn.Availability.IsUnavailable ||
			strings.HasPrefix(fn.Name, "__builtin") || isUPPFunction(fn.Name) {
			continue
		}
		if hasVAListArgFn(fn) || hasByValueUnknownTypeFor(framework, fn) {
			continue
		}
		goName := functionGoName(pkgTypeNames, fn)
		if seen[goName] {
			continue
		}
		seen[goName] = true
		out = append(out, fn)
	}
	return out
}

// packageTypeNames returns the package-level Go type names (structs,
// struct-typedefs, enums) that function wrapper names must not collide with.
func packageTypeNames(framework *macosplatformmetadata.FrameworkMeta) map[string]bool {
	pkgTypeNames := make(map[string]bool)
	for n := range framework.Structs {
		pkgTypeNames[naming.ExportedTypeName(n)] = true
	}
	for n, target := range framework.Typedefs {
		if strings.HasPrefix(target, "struct ") {
			pkgTypeNames[naming.ExportedTypeName(n)] = true
		}
	}
	for n := range framework.Enums {
		pkgTypeNames[naming.ExportedTypeName(n)] = true
	}
	return pkgTypeNames
}

// FunctionGoName returns the Go wrapper name EmitFunctions gives fn. C
// permits a struct and a function to share a name (e.g. mach_time.h declares
// both a mach_timebase_info struct and function); the natural Go name then
// collides with the emitted type, so the wrapper gains an "Fn" suffix
// (Mach_timebase_infoFn) instead of being silently dropped. The idiomatic
// layer resolves its raw call targets through this same rule. The spelling is
// backend-independent: the purego backend swaps only function bodies, never
// the Go surface.
func FunctionGoName(framework *macosplatformmetadata.FrameworkMeta, fn macosplatformmetadata.Function) string {
	return functionGoName(packageTypeNames(framework), fn)
}

func functionGoName(pkgTypeNames map[string]bool, fn macosplatformmetadata.Function) string {
	goName := naming.GoTypeName(fn.Name)
	if pkgTypeNames[goName] {
		goName += "Fn"
	}
	return goName
}

// buildFunctionModel builds the model for a single C function → Go wrapper.
// goName is the collision-resolved wrapper name from functionGoName.
// buildFunctionCallBody pre-renders the CGo call + exception check + optional return
// for a function with the given return type. The multi-path dispatch lives here in Go
// so the template can remain a structural description of the function wrapper.
// buildFunctionsImports assembles the import list for a _functions.go file.
// It scans the pre-rendered function bodies AND signatures for tell-tale strings
// rather than re-running the type mapper a second time.
// writeFunction is a thin adapter used by tests to exercise individual function
// generation without going through the full file model.
func hasVAListArgFn(fn macosplatformmetadata.Function) bool {
	for _, arg := range fn.Params {
		n := strings.ToLower(typemap.Normalise(arg.ObjCType))
		if strings.Contains(n, "va_list") {
			return true
		}
	}
	return false
}
