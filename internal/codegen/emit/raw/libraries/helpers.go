package rawlib

import (
	"fmt"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// isFormatStringVariadic reports whether method is a variadic ObjC method whose
// trailing args are printf-style format arguments (e.g. stringWithFormat:, predicateWithFormat:).
// These can be bridged with only the fixed named parameters; callers pre-format the string
// on the Go side using fmt.Sprintf before passing it.
func isFormatStringVariadic(method macosplatformmetadata.Method) bool {
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
func shouldSkipBridgeMethod(method macosplatformmetadata.Method) bool {
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
func buildParamNames(args []macosplatformmetadata.Param) []string {
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
func buildGoArgs(args []macosplatformmetadata.Param, hasNSError bool, ctx typemap.Context, m *typemap.Mapper, imports typemap.ImportSet) []string {
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
func buildGoReturn(method macosplatformmetadata.Method, ctx typemap.Context, m *typemap.Mapper, className string, imports typemap.ImportSet) string {
	ret := method.Return

	var parts []string

	// Compute the primary return type from the method's own ObjC return.
	var retType string
	if ret.IsGeneric && len(ctx.GenericParams) > 0 {
		// Return type is one of the class's generic type parameters (e.g. firstObject → T).
		// We cannot construct an arbitrary T from an unsafe.Pointer in Go generics without
		// a per-type factory, so we return cgo.Object (the common constraint interface).
		// Callers can type-assert to the concrete element type when needed.
		retType = "cgo.Object"
	} else if ret.IsInstancetype || typemap.IsInstancetype(ret.ObjCType) {
		if className != "" {
			retType = "*" + className
			if len(ctx.GenericParams) > 0 && !ctx.IsClassMethod {
				// Instance method on generic class: T is in scope.
				retType = "*" + className + "[T]"
			} else if m.GenericClasses[className] {
				// Class method or non-generic context: instantiate with cgo.Object.
				retType = "*" + className + "[cgo.Object]"
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

// goCallArgs builds the C function call arguments for the Go-side CGo call.
// It inserts o.ptr first for instance methods.
// methodRefsUnavailableClass reports whether any argument or return type in
// method references a class that is marked unavailable in framework. Methods that
// reference unavailable (iOS-only) classes cannot be generated because the
// type would be undefined in the compiled package.
func methodRefsUnavailableClass(method macosplatformmetadata.Method, framework *macosplatformmetadata.FrameworkMeta, allClasses map[string]macosplatformmetadata.Class) bool {
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

// ── Helpers relocated from the retired cgo class/bridge emitters ──────────────
// These are the only members of the former classes.go / bridge.go still reached
// by the live purego surface path: isObjectReturn/isKnownStruct/objectConstructExpr
// (+ its cross-framework helpers) are used by the purego function emitter, and
// isUPPFunction/hasByValueUnknownTypeFor (+ helpers) by EmittableFunctions. The
// rest of those files was dead once every library moved to the purego backend.

// isObjectReturn returns true if the Go return type is an ObjC wrapper pointer.
func isObjectReturn(goType string) bool {
	if !strings.HasPrefix(goType, "*") {
		return false
	}
	inner := goType[1:]
	switch inner {
	case "bool", "int8", "int16", "int32", "int64",
		"uint8", "uint16", "uint32", "uint64", "float32", "float64", "string":
		return false
	}
	return true
}

// isKnownStruct reports whether bare (the Go type name with pkg. prefix stripped)
// is a registered C struct. It checks both the Go-capitalized form (e.g. "Decform")
// and the original ObjC-lowercase form (e.g. "decform"), because the StructIndex
// registry stores ObjC names as-is while qualifiedStructType capitalises the first
// letter via naming.GoTypeName.
func isKnownStruct(bare string, m *typemap.Mapper) bool {
	// Preferred: forward-mapped exported Go names (bare is the resolved Go type
	// name, which uses the non-invertible ExportedTypeName mapping).
	if m.IsStructGoName(bare) {
		return true
	}
	// Fallback for single-word names (where ExportedTypeName == the capitalised C
	// name) and callers that did not build the Go-name index: the StructIndex is
	// keyed by the ObjC/C name, so also try the lowercase-first form.
	if m.StructIndex[bare] != "" {
		return true
	}
	if len(bare) > 0 {
		lower := strings.ToLower(string(bare[0])) + bare[1:]
		if m.StructIndex[lower] != "" {
			return true
		}
	}
	return false
}

// objectConstructExpr returns the Go expression that wraps _ptr in a typed Go object.
// structType has no leading "*" (e.g. "NSString", "NSArray[T]", "foundation.NSString").
func objectConstructExpr(
	structType, ptrVar string,
	fmClasses map[string]macosplatformmetadata.Class,
	m *typemap.Mapper,
) string {
	// Cross-framework: "foundation.NSString" → foundation.NewNSString(_ptr)
	if isCrossFrameworkType(structType) {
		return crossFrameworkCtor(structType, ptrVar)
	}

	// T-generic same-framework: "NSArray[T]" → NewNSArrayT[T](_ptr)
	if strings.Contains(structType, "[T]") {
		baseName := structType[:strings.Index(structType, "[")]
		return "New" + baseName + "T[T](" + ptrVar + ")"
	}

	// Non-T same-framework: "NSString" or "NSArray[runtime.Object]"
	baseName := structType
	if br := strings.Index(baseName, "["); br > 0 {
		baseName = baseName[:br]
	}
	return "New" + baseName + "(" + ptrVar + ")"
}

// isCrossFrameworkType returns true when structType belongs to a foreign package:
// it has a "." before any "[" (e.g. "foundation.NSString").
func isCrossFrameworkType(structType string) bool {
	bracket := strings.Index(structType, "[")
	dot := strings.Index(structType, ".")
	if dot < 0 {
		return false
	}
	return bracket < 0 || dot < bracket
}

// crossFrameworkCtor builds a foreign constructor call: pkg.NewTypeName(ptr).
func crossFrameworkCtor(structType, ptr string) string {
	dot := strings.Index(structType, ".")
	pkg := structType[:dot]
	typeName := structType[dot+1:]
	isGenericT := strings.Contains(typeName, "[T]")
	if br := strings.Index(typeName, "["); br > 0 {
		typeName = typeName[:br]
	}
	if isGenericT {
		// Use the exported generic constructor to preserve the T type parameter.
		return pkg + ".New" + typeName + "T[T](" + ptr + ")"
	}
	return pkg + ".New" + typeName + "(" + ptr + ")"
}

// isUPPFunction returns true for Carbon Universal Procedure Pointer functions
// (Invoke*UPP, New*UPP, Dispose*UPP). These use macros that dereference the
// UPP void* argument as a function pointer — which fails in a -fno-objc-arc
// bridge context where all pointers are void*. Skip them entirely.
func isUPPFunction(name string) bool {
	return (strings.HasPrefix(name, "Invoke") || strings.HasPrefix(name, "New") || strings.HasPrefix(name, "Dispose")) &&
		strings.HasSuffix(name, "UPP")
}

// hasByValueUnknownType returns true when any argument or return type is a
// C value type (no '*') that the typemap cannot represent as void* — specifically
// SIMD vector types (vFloat, vDouble) and structs passed by value (DenseMatrix_Float).
// The compiler cannot cast a pointer to these types, producing build errors.
//
// This check is intentionally narrow: it allows through pointer typedefs (recognised
// by _Nonnull / _Nullable nullability annotations — only valid on pointer types),
// enums, and any named type that the mapper would handle via its enum or CF tables.
// Entitlement-gated APIs (vmnet, NetworkExtension, etc.) use pointer typedefs and
// integer enums — they must NOT be filtered here.
// hasByValueUnknownTypeFor is hasByValueUnknownType with the declaring
// framework's own enum table consulted first: a bare enum name that matches
// none of the heuristics (compression_algorithm has no '_t' suffix) is still
// a plain C integer and must not disqualify the function.
func hasByValueUnknownTypeFor(
	framework *macosplatformmetadata.FrameworkMeta,
	fn macosplatformmetadata.Function,
) bool {
	if !hasByValueUnknownType(fn) {
		return false
	}
	isEnum := func(objcType string) bool {
		n := typemap.Normalise(objcType)
		_, ok := framework.Enums[n]
		return ok
	}
	if fn.Return.ObjCType != "" && hasByValueUnknownType(macosplatformmetadata.Function{
		Return: fn.Return,
	}) && !isEnum(fn.Return.ObjCType) {
		return true
	}
	for _, arg := range fn.Params {
		if hasByValueUnknownType(macosplatformmetadata.Function{
			Params: []macosplatformmetadata.Param{arg},
		}) && !isEnum(arg.ObjCType) {
			return true
		}
	}
	return false
}

func hasByValueUnknownType(fn macosplatformmetadata.Function) bool {
	check := func(objcType string) bool {
		if objcType == "" || objcType == "void" {
			return false
		}
		// Pointer types (explicit * or nullability-annotated pointer typedefs).
		if strings.Contains(objcType, "*") {
			return false
		}
		// Nullability annotations (_Nonnull, _Nullable, _Null_unspecified) are only
		// valid on pointer types — a bare typedef with one of these is a pointer.
		if strings.Contains(objcType, "_Nonnull") ||
			strings.Contains(objcType, "_Nullable") ||
			strings.Contains(objcType, "_Null_unspecified") {
			return false
		}
		n := typemap.Normalise(objcType)
		// Known scalar / ObjC meta types.
		if typemap.IsBOOL(n) || typemap.IsID(n) || typemap.IsSEL(n) || typemap.IsClass(n) {
			return false
		}
		if isVAList(n) || typemap.IsBlock(n) {
			return false
		}
		// Standard C scalar types.
		switch n {
		case "void", "bool", "_Bool",
			"char", "signed char", "unsigned char",
			"short", "unsigned short",
			"int", "unsigned int",
			"long", "unsigned long",
			"long long", "unsigned long long",
			"float", "double", "long double",
			"int8_t", "int16_t", "int32_t", "int64_t",
			"uint8_t", "uint16_t", "uint32_t", "uint64_t",
			"size_t", "ssize_t", "ptrdiff_t", "intptr_t", "uintptr_t",
			"NSInteger", "NSUInteger", "CGFloat":
			return false
		}
		// Named types without a nullability annotation and without * fall into two
		// categories:
		//   (a) Enum / integer typedefs — safe, CType() maps them to the right int type.
		//   (b) SIMD vector / struct-by-value types — unsafe.
		// We conservatively allow any type whose normalised name contains "Ref" or
		// ends in "_t" (common C typedef conventions for pointer/integer types), and
		// any type that looks like an enum (e.g. vmnet_return_t, FFTDirection).
		// Types known to be SIMD/vector (e.g. vFloat, vDouble, DenseMatrix_Float,
		// DenseVector_Double) have neither characteristic.
		if strings.HasSuffix(n, "_t") || strings.Contains(n, "Ref") {
			return false
		}
		// Enum-like names: contain no spaces and are not a known struct prefix.
		// Struct-by-value types from vecLib/BNNS look like: DenseMatrix_Float,
		// DenseVector_Double, BNNSNearestNeighbors, vFloat, vDouble, etc.
		// These all lack "_t"/"Ref" and represent value types.
		return true
	}
	if check(fn.Return.ObjCType) {
		return true
	}
	for _, arg := range fn.Params {
		if check(arg.ObjCType) {
			return true
		}
	}
	return false
}

// isVAList returns true for va_list ObjC argument types.
func isVAList(objcType string) bool {
	n := typemap.Normalise(objcType)
	return strings.Contains(n, "va_list") || strings.Contains(n, "__va_list")
}
