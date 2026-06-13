package raw

import (
	"fmt"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// isFormatStringVariadic reports whether method is a variadic ObjC method whose
// trailing args are printf-style format arguments (e.g. stringWithFormat:, predicateWithFormat:).
// These can be bridged with only the fixed named parameters; callers pre-format the string
// on the Go side using fmt.Sprintf before passing it.
func isFormatStringVariadic(method meta.Method) bool {
	return method.IsVariadic && strings.Contains(strings.ToLower(method.Selector), "format")
}

// shouldSkipBridgeMethod reports whether a method must be excluded from bridge
// generation. Two categories are excluded:
//
//   - Non-format variadic methods: CGo cannot express C variadic calls, so only
//     format-style variadics (which can be bridged with fixed named params) are kept.
//   - ObjC runtime lifecycle methods (+initialize, +load): the ObjC runtime calls
//     these automatically. Bridging an explicit call causes duplicate initialization
//     and undefined behaviour per [NSObject +initialize].
func shouldSkipBridgeMethod(method meta.Method) bool {
	if method.IsVariadic && !isFormatStringVariadic(method) {
		return true
	}
	if method.IsClassMethod && (method.Selector == "initialize" || method.Selector == "load") {
		return true
	}
	return false
}

// buildParamNames returns a deduplicated list of Go parameter names for args.
// Unnamed args get positional names (arg0, arg1…). When two args share the
// same name (valid in ObjC, e.g. recordId:formattedRecord:recordId:), the
// second and subsequent occurrences are suffixed with _2, _3, …
func buildParamNames(args []meta.Param) []string {
	seen := make(map[string]int)
	names := make([]string, len(args))
	for i, arg := range args {
		var base string
		if arg.Name == "" {
			base = fmt.Sprintf("arg%d", i)
		} else {
			base = naming.ParamName(arg.Name)
		}
		seen[base]++
		if seen[base] == 1 {
			names[i] = base
		} else {
			names[i] = fmt.Sprintf("%s_%d", base, seen[base])
		}
	}
	// Second pass: if any name appeared more than once, also rename the FIRST
	// occurrence so the original base name is not left ambiguous.
	firstSeen := make(map[string]int)
	for i, arg := range args {
		var base string
		if arg.Name == "" {
			base = fmt.Sprintf("arg%d", i)
		} else {
			base = naming.ParamName(arg.Name)
		}
		firstSeen[base]++
	}
	counter := make(map[string]int)
	for i, arg := range args {
		var base string
		if arg.Name == "" {
			base = fmt.Sprintf("arg%d", i)
		} else {
			base = naming.ParamName(arg.Name)
		}
		if firstSeen[base] > 1 {
			counter[base]++
			names[i] = fmt.Sprintf("%s_%d", base, counter[base])
		}
	}
	return names
}


// buildGoArgs converts method arguments to a Go parameter list.
// If hasNSError is true the trailing NSError** arg has already been removed
// by the scanner, so we just need to build the remaining args.
// Block args appear as Go func types; NSError * block args are mapped to the
// Go error interface so callers receive idiomatic errors (see GoBlockUserFuncType).
func buildGoArgs(args []meta.Param, hasNSError bool, ctx typemap.Context, m *typemap.Mapper, imports typemap.ImportSet) []string {
	resolved := buildParamNames(args)
	var result []string
	for i, arg := range args {
		// Detect block types and use user-friendly types (NSError * → error).
		blockObjCType := ""
		if arg.IsBlock {
			blockObjCType = arg.ObjCType
		} else if target, ok := m.TypedefIndex[typemap.Normalise(arg.ObjCType)]; ok && typemap.IsBlock(target) {
			blockObjCType = target
		}
		var goType string
		if blockObjCType != "" {
			goType = m.GoBlockUserFuncType(blockObjCType, ctx, imports)
		}
		if goType == "" {
			goType = m.GoType(arg.ObjCType, ctx, imports)
		}
		if goType == "" {
			goType = "unsafe.Pointer"
		}
		result = append(result, fmt.Sprintf("%s %s", resolved[i], goType))
	}
	return result
}

// buildGoReturn constructs the Go return type expression for a method.
// className is used to resolve instancetype.
func buildGoReturn(method meta.Method, ctx typemap.Context, m *typemap.Mapper, className string, imports typemap.ImportSet) string {
	ret := method.Return

	var parts []string

	// Compute the primary return type from the method's own ObjC return.
	var retType string
	if ret.IsGeneric && len(ctx.GenericParams) > 0 {
		// Return type is one of the class's generic type parameters (e.g. firstObject → T).
		// We cannot construct an arbitrary T from an unsafe.Pointer in Go generics without
		// a per-type factory, so we return objc.Object (the common constraint interface).
		// Callers can type-assert to the concrete element type when needed.
		retType = "objc.Object"
	} else if ret.IsInstancetype || typemap.IsInstancetype(ret.ObjCType) {
		if className != "" {
			retType = "*" + className
			if len(ctx.GenericParams) > 0 && !ctx.IsClassMethod {
				// Instance method on generic class: T is in scope.
				retType = "*" + className + "[T]"
			} else if m.GenericClasses[className] {
				// Class method or non-generic context: instantiate with objc.Object.
				retType = "*" + className + "[objc.Object]"
			}
		} else {
			retType = "unsafe.Pointer"
		}
	} else {
		retType = m.GoReturnType(ret.ObjCType, ctx, imports)
		// Nullable C string (const char *) cannot signal nil through the Go string
		// zero value. Wrap as *string so nil-returns are distinguishable.
		if ret.IsNullable && retType == "string" {
			retType = "*string"
		}
	}

	if retType != "" {
		parts = append(parts, retType)
	}

	// NSError** out-param → add error return.
	if method.IsNSError {
		parts = append(parts, "error")
	}

	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		return "(" + strings.Join(parts, ", ") + ")"
	}
}

// cArgList builds the C function argument list for the bridge header.
// It prepends a `void *self` argument for instance methods and adds
// `void **outError` when hasNSError is true.
func cArgList(isClassMethod bool, args []meta.Param, hasNSError bool, m *typemap.Mapper, ctx typemap.Context, imports typemap.ImportSet) string {
	resolved := buildParamNames(args)
	var parts []string
	if !isClassMethod {
		parts = append(parts, "void *self")
	}
	for i, arg := range args {
		cType := m.CType(arg.ObjCType, ctx, imports)
		parts = append(parts, fmt.Sprintf("%s %s", cType, resolved[i]))
	}
	if hasNSError {
		parts = append(parts, "void **outError")
	}
	if len(parts) == 0 {
		return "void"
	}
	return strings.Join(parts, ", ")
}

// goCallArgs builds the C function call arguments for the Go-side CGo call.
// It inserts o.ptr first for instance methods.
func goCallArgs(isClassMethod bool, args []meta.Param, hasNSError bool) string {
	resolved := buildParamNames(args)
	var parts []string
	if !isClassMethod {
		parts = append(parts, "o.ptr")
	}
	parts = append(parts, resolved...)
	if hasNSError {
		parts = append(parts, "&nsErr")
	}
	return strings.Join(parts, ", ")
}

// methodRefsUnavailableClass reports whether any argument or return type in
// method references a class that is marked unavailable in framework. Methods that
// reference unavailable (iOS-only) classes cannot be generated because the
// type would be undefined in the compiled package.
func methodRefsUnavailableClass(method meta.Method, framework *meta.FrameworkMeta, allClasses map[string]meta.Class) bool {
	check := func(objcType string) bool {
		s := objcType
		for len(s) > 0 {
			i := strings.IndexFunc(s, func(r rune) bool { return r >= 'A' && r <= 'Z' })
			if i < 0 {
				break
			}
			s = s[i:]
			j := 0
			for j < len(s) && (s[j] == '_' || (s[j] >= 'a' && s[j] <= 'z') || (s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= '0' && s[j] <= '9')) {
				j++
			}
			if j > 0 {
				name := s[:j]
				// Check current framework first, then cross-framework classes.
				if cls, ok := framework.Classes[name]; ok && cls.Availability.IsUnavailable {
					return true
				}
				if cls, ok := allClasses[name]; ok && cls.Availability.IsUnavailable {
					return true
				}
				s = s[j:]
			} else {
				s = s[1:]
			}
		}
		return false
	}
	for _, arg := range method.Params {
		if check(arg.ObjCType) {
			return true
		}
	}
	return check(method.Return.ObjCType)
}

// isPrimitiveGoType reports whether s is one of the Go built-in numeric/bool
// primitive type names produced by the type mapper. Used by goCGoArgExpr to
// distinguish *uint8 (pointer to scalar) from *SomeClass (ObjC wrapper).
func isPrimitiveGoType(s string) bool {
	switch s {
	case "bool",
		"int8", "int16", "int32", "int64",
		"uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return true
	}
	return false
}
