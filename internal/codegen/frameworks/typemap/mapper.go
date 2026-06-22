// Package typemap resolves ObjC qualType strings to Go type strings for the
// purego code generator. Unlike the CGo generator there is no CType() mapping —
// all types are expressed purely in Go using purego/objc primitives.
package typemap

import (
	"fmt"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// ImportSet accumulates the cross-framework imports needed by a single file.
// Key is the local alias (package name), value is the full import path.
type ImportSet map[string]string

// Context carries per-call state needed for context-sensitive resolution.
type Context struct {
	// ClassName is the ObjC class the current method belongs to ("NSString").
	ClassName string
	// Framework is the current framework being emitted ("Foundation").
	Framework string
	// GenericParams lists the ObjC generic type parameter names for ClassName
	// (e.g. ["ObjectType"] for NSArray). Empty for non-generic classes.
	GenericParams []string
	// ClassNameIndex maps known ObjC class names to their owning framework.
	// Used for protocol name disambiguation (protocol vs class with same name).
	ClassNameIndex map[string]string
	// IsReturn is true when resolving a method return type.
	IsReturn bool
	// IsClassMethod is true when resolving types on a class method.
	IsClassMethod bool
}

// Mapper resolves ObjC qualType strings to Go types.
type Mapper struct {
	// OwnerIndex maps ObjC class name → owning framework name.
	OwnerIndex map[string]string
	// GenericClasses is the set of ObjC class names that have generic params.
	GenericClasses map[string]bool
	// GenericParamIndex maps class name → ordered generic param names.
	GenericParamIndex map[string][]string
	// EnumIndex maps enum name → owning framework.
	EnumIndex map[string]string
	// EnumGoTypeIndex maps enum name → underlying Go integer type.
	EnumGoTypeIndex map[string]string
	// TypedefIndex maps typedef name → target ObjC qualType.
	TypedefIndex map[string]string
	// StructIndex maps struct name → owning framework.
	StructIndex map[string]string
	// EmittableStructs is the set of value-struct Go names (across all
	// frameworks) the idiomatic layer actually emits a definition for, so a
	// cross-framework reference targets only a struct that exists. Keyed by the
	// struct's exported Go name (e.g. "CGRect").
	EmittableStructs map[string]bool
	// ProtocolIndex maps protocol name → owning framework.
	ProtocolIndex map[string]string
	// CFTypeIndex maps framework-specific CF opaque typedef names → owning framework.
	CFTypeIndex map[string]string
	// ModulePrefix is the import path prefix for framework packages,
	// e.g. "github.com/deploymenttheory/go-bindings-macosplatform/purego-frameworks".
	ModulePrefix string
	// LibraryModulePrefix is the import path prefix for library packages.
	LibraryModulePrefix string
	// BlockedImports is the set of import edges to suppress for cycle breaking.
	// BlockedImports[srcFramework][dstFramework] = true means the src package
	// must not import the dst package; types degrade to unsafe.Pointer.
	BlockedImports map[string]map[string]bool
	// UnavailableClasses is the set of ObjC class names that are marked as
	// unavailable in the metadata and therefore not emitted as Go types.
	// References to these degrade to objc.ID to avoid referencing undefined types.
	UnavailableClasses map[string]bool
	// UnavailableEnumBaseTypes maps unavailable enum names to their underlying
	// Go integer type (e.g. "PHAssetCollectionSubtype" → "int64"). References
	// to these enums degrade to the underlying type since the enum is not emitted.
	UnavailableEnumBaseTypes map[string]string

	// Diagnostics accumulates a record each time a type resolution degrades:
	// to unsafe.Pointer for an unresolvable named/pointer type, or to objc.ID
	// for a cycle-blocked cross-framework reference. Intentional degradations
	// (unavailable classes, multi-protocol ids, function pointers) are not
	// recorded. The generator collects these per framework so the CLI can
	// enforce a committed baseline (new entries fail CI).
	Diagnostics []string
}

// appendDiag records a degradation diagnostic.
func (m *Mapper) appendDiag(format string, args ...any) {
	m.Diagnostics = append(m.Diagnostics, fmt.Sprintf(format, args...))
}

// AppendDiagnostic records a degradation diagnostic raised by an emitter
// (e.g. an exported-name collision or an unbridgeable block signature) so it
// participates in the committed diagnostics baseline like mapper-internal
// degradations.
func (m *Mapper) AppendDiagnostic(format string, args ...any) {
	m.appendDiag(format, args...)
}

// GoType resolves an ObjC qualType string to a Go type string.
// Cross-framework types are added to imports as a side effect.
// normalise strips ObjC nullability and other qualifiers that don't affect the
// Go type.
func normalise(qt string) string {
	qt = strings.TrimSpace(qt)
	for _, qual := range []string{
		"_Nonnull", "_Nullable", "_Null_unspecified",
		"__kindof", "__unsafe_unretained", "__autoreleasing",
		"__strong", "__weak", "_Nullable_result",
		"WK_SWIFT_UI_ACTOR", "__covariant",
	} {
		qt = strings.ReplaceAll(qt, qual, "")
	}
	// Collapse multiple spaces.
	for strings.Contains(qt, "  ") {
		qt = strings.ReplaceAll(qt, "  ", " ")
	}
	return strings.TrimSpace(qt)
}

// GoType resolves an ObjC qualType string to a Go type string.
// Cross-framework types are added to imports as a side effect.
func (m *Mapper) GoType(qt string, ctx Context, imports ImportSet) string {
	return m.resolveDepth(normalise(qt), ctx, imports, 0)
}

// GoReturnType resolves an ObjC qualType string in return position.
func (m *Mapper) GoReturnType(qt string, ctx Context, imports ImportSet) string {
	ctx.IsReturn = true
	return m.resolveDepth(normalise(qt), ctx, imports, 0)
}

// primitiveGoTypes maps scalar C/ObjC type spellings (booleans, integers,
// floats, ObjC handles, raw/char pointers) to their Go types.
var primitiveGoTypes = map[string]string{
	"BOOL": "bool", "bool": "bool", "_Bool": "bool",
	"int": "int", "unsigned int": "uint", "unsigned": "uint",
	"long": "int", "unsigned long": "uint",
	"long long": "int64", "unsigned long long": "uint64",
	"short": "int16", "unsigned short": "uint16",
	"char": "int8", "unsigned char": "uint8", "signed char": "int8",
	"float": "float32", "double": "float64", "long double": "float64",
	"int8_t": "int8", "int16_t": "int16", "int32_t": "int32", "int64_t": "int64",
	"uint8_t": "uint8", "uint16_t": "uint16", "uint32_t": "uint32", "uint64_t": "uint64",
	"size_t": "uint", "ssize_t": "int",
	"NSInteger": "int", "CFIndex": "int", "NSUInteger": "uint", "CGFloat": "float64",
	"id": "objc.ID", "SEL": "objc.SEL", "Class": "objc.Class", "IMP": "unsafe.Pointer",
	"void *": "unsafe.Pointer", "void*": "unsafe.Pointer",
	"const void *": "unsafe.Pointer", "const void*": "unsafe.Pointer",
	"char *": "string", "const char *": "string",
	"char*": "string", "const char*": "string",
}

// primitivePointerGoTypes maps pointer base types to their Go pointer types
// (uint8_t/unsigned char are handled separately for return-position rules).
var primitivePointerGoTypes = map[string]string{
	"int8_t": "*int8", "signed char": "*int8",
	"uint16_t": "*uint16", "unsigned short": "*uint16",
	"int16_t": "*int16", "short": "*int16",
	"uint32_t": "*uint32", "unsigned int": "*uint32",
	"int32_t": "*int32", "int": "*int32",
	"uint64_t": "*uint64", "unsigned long long": "*uint64",
	"int64_t": "*int64", "long long": "*int64", "NSInteger": "*int64", "long": "*int64",
	"CGFloat": "*float64", "double": "*float64",
	"float": "*float32",
}

// resolveDepth is the core type resolution engine with a depth guard.
func (m *Mapper) resolveDepth(n string, ctx Context, imports ImportSet, depth int) string {
	if depth > 12 {
		return "unsafe.Pointer"
	}
	n = strings.TrimSpace(n)

	// void
	if n == "void" || n == "" {
		return ""
	}

	// instancetype → *ClassName (or *ClassName[T1, T2] for generic classes)
	if n == "instancetype" {
		if ctx.ClassName != "" {
			if len(ctx.GenericParams) > 0 {
				return "*" + ctx.ClassName + "[" + strings.Join(ctx.GenericParams, ", ") + "]"
			}
			return "*" + ctx.ClassName
		}
		return "unsafe.Pointer"
	}

	// Booleans, primitive integers/floats, ObjC special types, char*.
	if goType, ok := primitiveGoTypes[n]; ok {
		return goType
	}

	// Block type: "RetType (^ ...)(params)" or "RetType (^)(params)"
	if strings.Contains(n, "(^") {
		return m.resolveBlock(n, ctx, imports, depth)
	}

	// Function pointer: "RetType (*)(params)" or "RetType (*name)(params)"
	if strings.Contains(n, "(*") {
		return "unsafe.Pointer"
	}

	// id<Protocol> handling
	if strings.HasPrefix(n, "id<") {
		return m.resolveIDProtocol(n, ctx, imports)
	}

	// C array: "type [N]"
	if idx := strings.LastIndex(n, "["); idx >= 0 && strings.HasSuffix(n, "]") {
		elem := strings.TrimSpace(n[:idx])
		size := n[idx+1 : len(n)-1]
		goElem := m.resolveDepth(elem, ctx, imports, depth+1)
		return "[" + size + "]" + goElem
	}

	// Pointer types
	if strings.HasSuffix(n, " *") || strings.HasSuffix(n, "*") {
		return m.resolvePointer(n, ctx, imports, depth)
	}

	// Named types: enums, structs, classes, typedefs
	return m.resolveNamed(n, ctx, imports, depth)
}

// resolvePointer handles pointer types like "NSString *", "uint8_t *", etc.
func (m *Mapper) resolvePointer(n string, ctx Context, imports ImportSet, depth int) string {
	// Strip trailing * to get the base type.
	base := strings.TrimRight(n, " *")
	base = strings.TrimSpace(base)
	base = strings.TrimPrefix(base, "const ")
	base = strings.TrimSpace(base)
	// Strip "struct " or "enum " tag prefix (e.g. "struct ColorSyncCMM" → "ColorSyncCMM").
	base = strings.TrimPrefix(base, "struct ")
	base = strings.TrimPrefix(base, "enum ")
	base = strings.TrimSpace(base)

	// void* → unsafe.Pointer
	if base == "void" || base == "" {
		return "unsafe.Pointer"
	}

	// char* → string
	if base == "char" || base == "signed char" {
		return "string"
	}

	// BOOL* → *bool
	if base == "BOOL" || base == "_Bool" {
		return "*bool"
	}

	// NSError** is handled by the IsNSError flag on Method; skip here.
	if base == "NSError *" || base == "NSError" {
		return "unsafe.Pointer"
	}

	// Known ObjC class pointer → *ClassName (or *ClassName[objc.ID, ...] for generics)
	if m.OwnerIndex != nil {
		if owner, ok := m.OwnerIndex[base]; ok {
			// Classes marked unavailable are not emitted; degrade to objc.ID.
			if m.UnavailableClasses != nil && m.UnavailableClasses[base] {
				return "objc.ID"
			}
			goType := "*" + base
			// If the class is generic and no type params were given in the ObjC
			// type string, instantiate with objc.ID to produce valid Go code.
			if m.GenericClasses != nil && m.GenericClasses[base] {
				if params, ok2 := m.GenericParamIndex[base]; ok2 && len(params) > 0 {
					ids := make([]string, len(params))
					for i := range ids {
						ids[i] = "objc.ID"
					}
					goType = "*" + base + "[" + strings.Join(ids, ", ") + "]"
				}
			}
			return m.qualifyType(goType, base, owner, ctx.Framework, imports)
		}
	}

	// Generic collection pointer: NSArray<T>*
	if strings.Contains(base, "<") {
		return m.resolveGenericPointer(base, ctx, imports, depth)
	}

	// Struct pointer
	if m.StructIndex != nil {
		if owner, ok := m.StructIndex[base]; ok {
			if goName := m.exportedStructName(base); goName != "" {
				return m.qualifyType("*"+goName, base, owner, ctx.Framework, imports)
			}
		}
	}

	// Primitive pointer
	if base == "uint8_t" || base == "unsigned char" {
		if ctx.IsReturn {
			return "unsafe.Pointer"
		}
		return "*uint8"
	}
	if goType, ok := primitivePointerGoTypes[base]; ok {
		return goType
	}

	// Typedef chain — iterative to avoid mutual recursion.
	if resolved, ok := m.resolvePointerTypedefChain(base, ctx, imports, depth); ok {
		return resolved
	}

	m.appendDiag("%s: %q → unsafe.Pointer (unresolved pointer type)", ctx.Framework, n)
	return "unsafe.Pointer"
}

// resolvePointerTypedefChain walks the typedef chain for a pointer base type
// (e.g. ColorSyncCMMRef) and returns "*<resolved>" when the chain ends in a
// usable Go type.
func (m *Mapper) resolvePointerTypedefChain(
	base string,
	ctx Context,
	imports ImportSet,
	depth int,
) (string, bool) {
	if m.TypedefIndex == nil {
		return "", false
	}
	visited := map[string]bool{}
	cur := base
	for i := 0; i < 8 && !visited[cur]; i++ {
		visited[cur] = true
		target, ok := m.TypedefIndex[cur]
		if !ok {
			return "", false
		}
		resolved := m.resolveDepth(normalise(target), ctx, imports, depth+1)
		if resolved == "" || resolved == "unsafe.Pointer" {
			continue
		}
		// Don't build a pointer to an unexported cross-package identifier
		// (e.g. id* → **foundation.objc_object); fall back to unsafe.Pointer.
		if dotIdx := strings.LastIndex(resolved, "."); dotIdx >= 0 {
			after := strings.TrimPrefix(resolved[dotIdx+1:], "*")
			if len(after) > 0 && after[0] >= 'a' && after[0] <= 'z' {
				return "", false
			}
		}
		return "*" + resolved, true
	}
	return "", false
}

// resolveGenericPointer handles NSArray<NSString *>* etc.
func (m *Mapper) resolveGenericPointer(
	base string,
	ctx Context,
	imports ImportSet,
	depth int,
) string {
	open := strings.Index(base, "<")
	if open < 0 {
		return "unsafe.Pointer"
	}
	className := base[:open]
	inner := strings.TrimSuffix(base[open+1:], ">")

	owner, ok := m.OwnerIndex[className]
	if !ok {
		m.appendDiag(
			"%s: %q → unsafe.Pointer (unknown generic class %s)",
			ctx.Framework,
			base,
			className,
		)
		return "unsafe.Pointer"
	}

	// Check if blocked import
	if m.isBlocked(ctx.Framework, owner) {
		m.appendDiag("%s: %q → unsafe.Pointer (import cycle %s→%s)",
			ctx.Framework, base, strings.ToLower(ctx.Framework), strings.ToLower(owner))
		return "unsafe.Pointer"
	}

	// If the class is not generic in Go, the <T> part is an ObjC protocol
	// conformance annotation (e.g. NSObject<OS_dispatch_queue> *) rather than
	// a real type parameter. Treat as a plain class pointer.
	if m.GenericClasses == nil || !m.GenericClasses[className] {
		return m.qualifyType("*"+className, className, owner, ctx.Framework, imports)
	}

	// Split multiple ObjC type args respecting nested angle brackets
	// (e.g. "NSString *, id" for NSDictionary<K,V>) and resolve each.
	// Pad with objc.ID when the ObjC type supplies fewer args than the
	// Go generic class has parameters (e.g. NSDictionary<NSString *> → 2 params).
	superParams := m.GenericParamIndex[className]
	parts := splitCSV(inner)
	typeArgs := make([]string, len(superParams))
	for i := range typeArgs {
		if i < len(parts) {
			arg := m.resolveDepth(strings.TrimSpace(parts[i]), ctx, imports, depth+1)
			if arg == "" || arg == "unsafe.Pointer" {
				arg = "objc.ID"
			}
			// If the resolved type arg is itself a generic instantiation (contains "["),
			// degrade to objc.ID to prevent Go generic instantiation cycles
			// (e.g. NSArray[*NSOrderedCollectionChange[T]] → NSArray[objc.ID]).
			if strings.Contains(arg, "[") {
				arg = "objc.ID"
			}
			typeArgs[i] = arg
		} else {
			typeArgs[i] = "objc.ID"
		}
	}

	result := className + "[" + strings.Join(typeArgs, ", ") + "]"
	return m.qualifyType("*"+result, className, owner, ctx.Framework, imports)
}

// resolveIDProtocol handles id<Protocol> types.
func (m *Mapper) resolveIDProtocol(n string, ctx Context, imports ImportSet) string {
	open := strings.Index(n, "<")
	close := strings.LastIndex(n, ">")
	if open < 0 || close <= open {
		return "objc.ID"
	}
	inner := strings.TrimSpace(n[open+1 : close])
	if strings.Contains(inner, ",") {
		// Multi-protocol — can't represent as a single interface
		return "objc.ID"
	}
	if m.ProtocolIndex == nil {
		return "objc.ID"
	}
	owner, ok := m.ProtocolIndex[inner]
	if !ok {
		return "objc.ID"
	}
	protoGoName := inner
	if _, isClass := m.OwnerIndex[inner]; isClass {
		protoGoName = inner + "Protocol"
	}
	return m.qualifyType(protoGoName, protoGoName, owner, ctx.Framework, imports)
}

// BlockSignature is the resolved component view of an ObjC block type:
// the normalised ObjC type strings of its parameters and return value.
type BlockSignature struct {
	ParamObjCTypes []string
	ReturnObjCType string
}

// parseBlockComponents splits an ObjC block literal such as
// "ReturnType (^ _Nonnull)(ParamType1, ParamType2)" into its return type and
// parameter ObjC type strings (param names stripped, qualifiers normalised).
func parseBlockComponents(n string) (BlockSignature, bool) {
	caretIdx := strings.Index(n, "(^")
	if caretIdx < 0 {
		return BlockSignature{}, false
	}
	signature := BlockSignature{ReturnObjCType: strings.TrimSpace(n[:caretIdx])}

	// Skip the "(^ ...)" caret expression, then expect "(params)".
	rest := n[caretIdx:]
	closeParen := strings.Index(rest, ")")
	if closeParen < 0 {
		return BlockSignature{}, false
	}
	rest = strings.TrimSpace(rest[closeParen+1:])
	if !strings.HasPrefix(rest, "(") {
		// No param list — block takes no params.
		return signature, true
	}
	if !strings.HasSuffix(rest, ")") {
		return BlockSignature{}, false
	}
	params := strings.TrimSpace(rest[1 : len(rest)-1])
	if params == "" || params == "void" {
		return signature, true
	}
	for _, param := range splitCSV(params) {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}
		signature.ParamObjCTypes = append(
			signature.ParamObjCTypes,
			normalise(stripParamName(param)),
		)
	}
	return signature, true
}

// ResolveBlockSignature reports whether objcType denotes an ObjC block —
// either a literal "(^" type or a typedef chain ending in one — and returns
// its component ObjC types. The typedef walk mirrors resolveNamed (≤8 hops).
func (m *Mapper) ResolveBlockSignature(objcType string) (BlockSignature, bool) {
	n := normalise(objcType)
	if strings.Contains(n, "(^") {
		return parseBlockComponents(n)
	}
	bare := strings.TrimPrefix(n, "const ")
	bare = strings.TrimSpace(bare)
	if m.TypedefIndex == nil || strings.ContainsAny(bare, " *<>^()") {
		return BlockSignature{}, false
	}
	visited := map[string]bool{}
	cur := bare
	for i := 0; i < 8 && !visited[cur]; i++ {
		visited[cur] = true
		target, ok := m.TypedefIndex[cur]
		if !ok {
			return BlockSignature{}, false
		}
		next := normalise(target)
		if strings.Contains(next, "(^") {
			return parseBlockComponents(next)
		}
		if strings.ContainsAny(next, " *<>^()") {
			return BlockSignature{}, false
		}
		cur = next
	}
	return BlockSignature{}, false
}

// resolveBlock parses an ObjC block type string to a Go func type.
func (m *Mapper) resolveBlock(n string, ctx Context, imports ImportSet, depth int) string {
	signature, ok := parseBlockComponents(n)
	if !ok {
		return "unsafe.Pointer"
	}

	goParams := make([]string, 0, len(signature.ParamObjCTypes))
	for _, paramObjC := range signature.ParamObjCTypes {
		resolved := m.resolveDepth(paramObjC, ctx, imports, depth+1)
		if resolved == "" {
			resolved = "unsafe.Pointer"
		}
		goParams = append(goParams, resolved)
	}
	goRet := m.resolveDepth(signature.ReturnObjCType, ctx, imports, depth+1)

	sig := "func(" + strings.Join(goParams, ", ") + ")"
	if goRet != "" {
		sig += " " + goRet
	}
	return sig
}

// resolveNamed resolves a non-pointer named type (enum, struct, typedef, class).
func (m *Mapper) resolveNamed(n string, ctx Context, imports ImportSet, depth int) string {
	// Strip "struct " or "enum " prefix
	bare := n
	bare = strings.TrimPrefix(bare, "struct ")
	bare = strings.TrimPrefix(bare, "enum ")
	bare = strings.TrimPrefix(bare, "const ")
	bare = strings.TrimSpace(bare)

	// Check generic params — if it's a generic param name like "ObjectType"
	for _, gp := range ctx.GenericParams {
		if bare == gp {
			return bare // use as-is (the type parameter in the generic signature)
		}
	}

	// Enum
	if m.EnumIndex != nil {
		if owner, ok := m.EnumIndex[bare]; ok {
			// Unavailable enums are not emitted; return their underlying integer type.
			if m.UnavailableEnumBaseTypes != nil {
				if baseType, unavail := m.UnavailableEnumBaseTypes[bare]; unavail &&
					baseType != "" {
					return baseType
				}
			}
			exported := exportedName(bare)
			return m.qualifyType(exported, bare, owner, ctx.Framework, imports)
		}
		// Try capitalised version
		cap := capitalize(bare)
		if owner, ok := m.EnumIndex[cap]; ok {
			if m.UnavailableEnumBaseTypes != nil {
				if baseType, unavail := m.UnavailableEnumBaseTypes[cap]; unavail && baseType != "" {
					return baseType
				}
			}
			exported := exportedName(cap)
			return m.qualifyType(exported, cap, owner, ctx.Framework, imports)
		}
	}

	// Struct
	if m.StructIndex != nil {
		if owner, ok := m.StructIndex[bare]; ok {
			if goName := m.exportedStructName(bare); goName != "" {
				return m.qualifyType(goName, bare, owner, ctx.Framework, imports)
			}
		}
	}

	// ObjC class (non-pointer occurrence — shouldn't happen often)
	if m.OwnerIndex != nil {
		if owner, ok := m.OwnerIndex[bare]; ok {
			if m.UnavailableClasses != nil && m.UnavailableClasses[bare] {
				return "objc.ID"
			}
			goType := bare
			if m.GenericClasses != nil && m.GenericClasses[bare] {
				if params, ok2 := m.GenericParamIndex[bare]; ok2 && len(params) > 0 {
					ids := make([]string, len(params))
					for i := range ids {
						ids[i] = "objc.ID"
					}
					goType = bare + "[" + strings.Join(ids, ", ") + "]"
				}
			}
			return m.qualifyType(goType, bare, owner, ctx.Framework, imports)
		}
	}

	// Typedef chain first — resolves concrete types (e.g. ColorSyncCMMRef →
	// "struct ColorSyncCMM *" → *ColorSyncCMM). Placed before CFTypeIndex so
	// that typedef-defined opaque pointers resolve to their base struct rather
	// than returning a bare typedef name that may be undefined as a Go type.
	if resolved, ok := m.resolveNamedTypedefChain(bare, ctx, imports, depth); ok {
		return resolved
	}

	// CF opaque (fallback for types not in TypedefIndex but detected by loader).
	if m.CFTypeIndex != nil {
		if owner, ok := m.CFTypeIndex[bare]; ok {
			if goName := m.exportedStructName(bare); goName != "" {
				return m.qualifyType("*"+goName, bare, owner, ctx.Framework, imports)
			}
		}
	}

	m.appendDiag("%s: %q → unsafe.Pointer (unresolved named type)", ctx.Framework, n)
	return "unsafe.Pointer"
}

// resolveNamedTypedefChain walks the typedef chain for a bare named type and
// returns the first concrete resolution (≤8 hops, cycle-guarded).
func (m *Mapper) resolveNamedTypedefChain(
	bare string,
	ctx Context,
	imports ImportSet,
	depth int,
) (string, bool) {
	if m.TypedefIndex == nil {
		return "", false
	}
	visited := map[string]bool{}
	cur := bare
	for i := 0; i < 8 && !visited[cur]; i++ {
		visited[cur] = true
		target, ok := m.TypedefIndex[cur]
		if !ok {
			return "", false
		}
		if resolved := m.resolveDepth(normalise(target), ctx, imports, depth+1); resolved != "" {
			return resolved, true
		}
		next := normalise(target)
		if strings.ContainsAny(next, " *<>^()") {
			return "", false
		}
		cur = next
	}
	return "", false
}

// exportedStructName returns the exported Go type name the struct emitter
// uses for a C struct, or "" when the exported name is claimed by an ObjC
// class (the struct emitter skips such structs, so references must degrade).
func (m *Mapper) exportedStructName(structName string) string {
	goName := naming.ExportedTypeName(structName)
	if goName == "" {
		return ""
	}
	if m.OwnerIndex != nil {
		if _, isClass := m.OwnerIndex[goName]; isClass {
			return ""
		}
	}
	return goName
}

// qualifyType returns the Go type string for a type owned by ownerFramework,
// adding a cross-framework import to imports when necessary.
func (m *Mapper) qualifyType(
	goType, typeName, ownerFramework, currentFramework string,
	imports ImportSet,
) string {
	if ownerFramework == "" || ownerFramework == currentFramework {
		return goType
	}
	if m.isBlocked(currentFramework, ownerFramework) {
		// Use objc.ID for blocked ObjC object references — cleaner than unsafe.Pointer
		// and still fully functional since ObjC objects are just opaque ID handles.
		m.appendDiag("%s: %s.%s → objc.ID (import cycle %s→%s)",
			currentFramework, strings.ToLower(ownerFramework), typeName,
			strings.ToLower(currentFramework), strings.ToLower(ownerFramework))
		return "objc.ID"
	}
	pkgName := strings.ToLower(ownerFramework)
	importPath := m.ModulePrefix + "/" + pkgName
	if imports != nil {
		imports[pkgName] = importPath
	}
	// Prefix the type name (strip leading * if present, add it back after prefix)
	if strings.HasPrefix(goType, "*") {
		return "*" + pkgName + "." + goType[1:]
	}
	if strings.HasPrefix(goType, "[") {
		// Array type — prefix the element
		idx := strings.LastIndex(goType, "]")
		elem := goType[idx+1:]
		return goType[:idx+1] + pkgName + "." + elem
	}
	return pkgName + "." + goType
}

// isBlocked reports whether importing dstFramework from srcFramework is blocked.
func (m *Mapper) isBlocked(srcFramework, dstFramework string) bool {
	if m.BlockedImports == nil {
		return false
	}
	if blocked, ok := m.BlockedImports[srcFramework]; ok {
		return blocked[dstFramework]
	}
	return false
}

// IsEnumType reports whether a Go type string (possibly cross-package qualified)
// represents a generated ObjC enum type. If true, the zero value is "0".
func (m *Mapper) IsEnumType(goType string) bool {
	if m.EnumGoTypeIndex == nil {
		return false
	}
	bare := goType
	if dotIdx := strings.LastIndex(bare, "."); dotIdx >= 0 {
		bare = bare[dotIdx+1:]
	}
	if brIdx := strings.Index(bare, "["); brIdx >= 0 {
		bare = bare[:brIdx]
	}
	_, ok := m.EnumGoTypeIndex[bare]
	return ok
}

// IsCoreFoundationOpaqueRef reports whether a typedef name looks like a
// CoreFoundation opaque reference (e.g. CFStringRef, CGColorRef).
func IsCoreFoundationOpaqueRef(name string) bool {
	// CoreFoundation opaque refs end in Ref and start with CF.
	if strings.HasPrefix(name, "CF") && strings.HasSuffix(name, "Ref") {
		return true
	}
	// CG types from CoreGraphics (also in cfTypedefSet conceptually)
	if strings.HasPrefix(name, "CG") && strings.HasSuffix(name, "Ref") {
		return true
	}
	return false
}

// splitCSV splits a comma-separated parameter list, respecting angle brackets
// for generic types.
func splitCSV(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, c := range s {
		switch c {
		case '<', '(':
			depth++
		case '>', ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// stripParamName removes a trailing parameter name from an ObjC type+name
// string like "NSString * _Nonnull name" → "NSString * _Nonnull".
func stripParamName(s string) string {
	// Split on spaces, the last token that doesn't start with _ or * and
	// doesn't look like a qualifier is the name.
	parts := strings.Fields(s)
	if len(parts) <= 1 {
		return s
	}
	last := parts[len(parts)-1]
	// If last part looks like a type qualifier or pointer symbol, keep it.
	if last == "*" || strings.HasPrefix(last, "_") || last == "const" {
		return s
	}
	// If last part starts with a letter (variable name), strip it.
	if len(last) > 0 && ((last[0] >= 'a' && last[0] <= 'z') || (last[0] >= 'A' && last[0] <= 'Z')) {
		return strings.TrimSpace(strings.Join(parts[:len(parts)-1], " "))
	}
	return s
}

// exportedName uppercases the first letter.
func exportedName(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// capitalize uppercases the first letter (alias for consistency).
func capitalize(s string) string { return exportedName(s) }
