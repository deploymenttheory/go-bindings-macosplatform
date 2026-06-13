package typemap

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Context carries the per-call state for a single GoType resolution.
// Stable, run-wide data (OwnerIndex, EnumIndex, …) lives on the Mapper;
// only fields that vary between individual method/argument resolutions belong here.
type Context struct {
	// ClassName is the ObjC class being processed (resolves instancetype).
	ClassName string
	// Framework is the framework being generated (e.g. "Foundation").
	Framework string
	// ClassNameIndex is the set of ObjC class names visible in the current framework.
	ClassNameIndex map[string]bool
	// GenericParams are the generic type parameter names of the current class
	// (e.g. ["ObjectType", "KeyType"] for NSDictionary). Set per-class by emitters.
	GenericParams []string
	// IsReturn indicates this type is in a return-value position.
	IsReturn bool
	// IsClassMethod indicates the type is being resolved for a class method (+method).
	// Class methods have no receiver [T] in scope, so generic type parameters must
	// be replaced with cgo.Object rather than T.
	IsClassMethod bool
}

// ImportSet is populated as a side effect of type resolution. It maps
// Go package alias → full import path for cross-framework references.
// Pass a non-nil ImportSet to GoType/GoReturnType/CType to collect imports.
type ImportSet map[string]string

// Mapper converts ObjC qualType strings to Go type strings.
type Mapper struct {
	// GenericClasses is the set of class names that use Go generics.
	GenericClasses map[string]bool
	// GenericParamIndex maps a generic class name to its ordered list of ObjC
	// generic type parameter names. See Context.GenericParamIndex.
	GenericParamIndex map[string][]string
	// OwnerIndex maps class name → owning framework name.
	// Used to qualify cross-framework type references (e.g. *foundation.NSString).
	OwnerIndex map[string]string
	// EnumIndex maps enum type name → owning framework.
	// Used to resolve enum return types to their Go equivalents.
	EnumIndex map[string]string
	// EnumGoTypeIndex maps enum type name → underlying Go integer type (e.g. "int64").
	// Used by CType() and bridge emitters to produce correct C casts for enum arguments.
	EnumGoTypeIndex map[string]string
	// TypedefIndex maps typedef name → target ObjC qualType.
	// Used as a last-resort fallback before degrading to unsafe.Pointer.
	TypedefIndex map[string]string
	// StructIndex maps struct name → owning framework. See Context.StructIndex.
	StructIndex map[string]string
	// CFTypeIndex maps framework-specific CF opaque typedef names to their
	// owning framework (e.g. "CFHTTPMessageRef" → "CFNetwork"). These are CF-style
	// reference types generated outside CoreFoundation and not in cfTypedefSet.
	CFTypeIndex map[string]string
	// ProtocolIndex maps protocol name → owning framework. See Context.ProtocolIndex.
	ProtocolIndex map[string]string
	// ProtocolProxyIndex maps protocol name → owning framework.
	// When non-empty, return-position id<Protocol> for a single known protocol
	// resolves to *<GoProtoName>IDProtocol instead of unsafe.Pointer.
	ProtocolProxyIndex map[string]string
	// ModulePrefix is the Go module import path prefix for framework packages.
	ModulePrefix string
	// BlockedImports maps sourceFramework → set of targetFrameworks whose concrete
	// types must not be referenced. When a type resolution would produce a cross-
	// framework reference into a blocked target, unsafe.Pointer is emitted instead.
	// This breaks import cycles detected before code generation begins.
	BlockedImports map[string]map[string]bool

	// Diagnostics accumulates a record every time a type degrades to unsafe.Pointer
	// due to a blocked import cycle or an unresolvable cross-framework reference.
	// The generator can print these under --verbose to help diagnose missing types.
	Diagnostics []string

	// IsNSStringOverloads enables generation of additional Go-string overloads for
	// methods that accept NSString * parameters. Each such method gets a companion
	// with a "Go" suffix where NSString * args become plain Go string args.
	IsNSStringOverloads bool

	// cache stores resolved type strings keyed by (normalised type, context fields
	// that affect resolution). It is populated lazily on first use.
	// Values also carry the set of cross-framework imports that resolution produced
	// so they can be replayed into the caller's UsedImports map on a cache hit.
	cache map[typeCacheKey]typeCacheVal
	// mu guards cache for concurrent GoType calls.
	mu sync.RWMutex
}

// New returns a Mapper.
func New() *Mapper { return &Mapper{} }

// BaseContext returns a Context pre-populated with the per-call fields.
// Stable, run-wide data (OwnerIndex, EnumIndex, etc.) is read directly
// from the Mapper by resolution methods — callers need not set it on Context.
func (m *Mapper) BaseContext(framework string, knownClasses map[string]bool) Context {
	return Context{
		Framework:      framework,
		ClassNameIndex: knownClasses,
	}
}

// GoType converts an ObjC qualType string to its Go equivalent.
// It returns the Go type string (e.g. "uint64", "*NSString", "unsafe.Pointer").
// An empty string means "no type" (void return).
//
// Results are cached keyed by (normalised type, framework, generic params, class name).
// Cross-framework import side effects are also cached and replayed on hits so callers
// always receive correct imports regardless of whether the result came from the cache.
func (m *Mapper) GoType(qt string, ctx Context, imports ImportSet) string {
	n := strings.TrimSpace(Normalise(qt))
	if n == "" || n == "void" {
		return ""
	}
	return m.resolveWithCache(cacheKeyFor(n, ctx), ctx, imports, func(c Context, imp ImportSet) string {
		return m.resolveGoType(n, c, imp)
	})
}

// GoReturnType is like GoType but handles the void case explicitly — it returns
// the empty string for void (caller checks len).
func (m *Mapper) GoReturnType(qt string, ctx Context, imports ImportSet) string {
	ctx.IsReturn = true
	return m.GoType(qt, ctx, imports)
}

// EnumGoIntType returns the underlying Go integer type (e.g. "int64") for a Go
// enum type expression (e.g. "CFNotificationSuspensionBehavior" or
// "corefoundation.CFNotificationSuspensionBehavior"). Returns "" when goType is
// not a known enum. Used by bridge emitters to generate correct integer casts.
func (m *Mapper) EnumGoIntType(goType string) string {
	name := goType
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		name = name[dot+1:]
	}
	return m.EnumGoTypeIndex[name]
}

// CType returns the C type for a bridge function parameter/return.
// This is what appears in the generated .h file.
func (m *Mapper) CType(qt string, ctx Context, imports ImportSet) string {
	g := m.GoType(qt, ctx, imports)
	switch g {
	case "":
		return "void"
	case "bool":
		return "bool"
	case "int8":
		return "int8_t"
	case "int16":
		return "int16_t"
	case "int32":
		return "int32_t"
	case "int64":
		return "int64_t"
	case "uint8":
		return "uint8_t"
	case "uint16":
		return "uint16_t"
	case "uint32":
		return "uint32_t"
	case "uint64":
		return "uint64_t"
	case "float32":
		return "float"
	case "float64":
		return "double"
	case "string":
		return "const char *"
	}
	// Named enum types: strip any package qualifier and look up the underlying
	// integer type so the bridge declares the correct C integer type instead of void*.
	if cInt := m.resolveEnumCType(g); cInt != "" {
		return cInt
	}
	// ObjC object pointer wrappers → id (the semantically correct ObjC bridge type).
	// id<Protocol> resolutions and typedef-resolved ObjC objects (e.g. dispatch_queue_t →
	// NSObject<OS_dispatch_queue>) both surface as "cgo.Object" from GoType.
	// Named ObjC class pointer wrappers surface as "*ClassName" or "*pkg.ClassName[...]".
	// C struct wrappers, CF opaque types, and other non-ObjC pointers stay as void *.
	// The OwnerIndex check is the key discriminator: only classes registered as ObjC
	// classes (scanned from the SDK) qualify; CF opaque types (e.g. MTAudioProcessingTapRef),
	// C struct aliases, and framework-specific CF types are absent from OwnerIndex.
	if g == "cgo.Object" {
		return "id"
	}
	if strings.HasPrefix(g, "*") {
		goBase := g[1:]
		if dot := strings.LastIndex(goBase, "."); dot >= 0 {
			goBase = goBase[dot+1:]
		}
		if bracket := strings.Index(goBase, "["); bracket >= 0 {
			goBase = goBase[:bracket]
		}
		// Only emit id for types registered as ObjC classes in OwnerIndex.
		// This correctly excludes CF opaque types, C struct aliases, and other
		// non-ObjC pointer types that GoType also represents as *T.
		if goBase != "" {
			if _, isObjCClass := m.OwnerIndex[goBase]; isObjCClass {
				return "id"
			}
		}
	}
	// Genuinely opaque: blocks, raw byte pointers, C/CF structs, NSZone, Class.
	return "void *"
}

// resolveEnumCType returns the C integer type (e.g. "int64_t") for a Go enum type
// expression (e.g. "CFNotificationSuspensionBehavior" or
// "corefoundation.CFNotificationSuspensionBehavior"). Returns "" when goType
// is not a known enum.
func (m *Mapper) resolveEnumCType(goType string) string {
	name := goType
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		name = name[dot+1:]
	}
	goIntType, ok := m.EnumGoTypeIndex[name]
	if !ok {
		return ""
	}
	switch goIntType {
	case "int8":
		return "int8_t"
	case "int16":
		return "int16_t"
	case "int32":
		return "int32_t"
	case "int64":
		return "int64_t"
	case "uint8":
		return "uint8_t"
	case "uint16":
		return "uint16_t"
	case "uint32":
		return "uint32_t"
	case "uint64":
		return "uint64_t"
	}
	return ""
}

// --- internal ---

// resolveGoType performs the actual ObjC→Go type resolution on an already-normalised
// string n. It must not call Normalise again. Cross-framework imports are written to
// imports (the caller-supplied ImportSet).
func (m *Mapper) resolveGoType(n string, ctx Context, imports ImportSet) string {
	// instancetype → pointer to the class being defined
	if IsInstancetype(n) {
		if ctx.ClassName != "" {
			return "*" + ctx.ClassName
		}
		return "unsafe.Pointer"
	}

	// Block type: void (^)(NSString *, NSError *)
	// In parameter position: resolve to a primitive Go func type so callers can
	// pass Go closures directly to MakeBlock_* factories.
	// In return position: keep as unsafe.Pointer — a returned block is an ObjC
	// object whose lifecycle must be managed by the caller.
	if IsBlock(n) {
		if !ctx.IsReturn {
			if ft := m.buildBlockGoType(n, ctx); ft != "" {
				return ft
			}
		}
		return "unsafe.Pointer"
	}

	// Primitive and well-known C/ObjC types
	if g := primitiveGoType(n); g != "" {
		return g
	}

	// id<Protocol> → typed Go interface or proxy type.
	// In parameter position, single-protocol id<P> resolves to the matching
	// Go interface; multi-protocol id<P1,P2> emits an inline composite interface.
	//
	// In return position, a proxy type is used when available: the bridge retains
	// the returned object and wraps it in *<GoProtoName>Proxy, which embeds
	// foundation.NSObject and exposes the protocol methods via CGo. If no proxy
	// is registered for the protocol (or the reference is cross-framework and
	// import-blocked), fall back to cgo.Object (the common ObjC object interface).
	if protos := IDProtocols(n); len(protos) > 0 {
		if ctx.IsReturn {
			if len(protos) == 1 && len(m.ProtocolProxyIndex) > 0 {
				if resolved := m.qualifiedProtocolIDType(protos[0], ctx, imports); resolved != "" {
					return resolved
				}
			}
			return "cgo.Object"
		}
		if resolved := m.qualifiedProtocolType(protos, ctx, imports); resolved != "" {
			return resolved
		}
		// Unknown protocols (none in ProtocolIndex): use cgo.Object as the
		// minimum guarantee that the caller passes a valid ObjC object wrapper.
		return "cgo.Object"
	}

	// id → cgo.Object. Bare `id` is an untyped ObjC object reference; cgo.Object
	// (interface { Ptr() unsafe.Pointer }) is the common constraint satisfied by
	// every generated wrapper type, so callers can pass any typed wrapper directly.
	if IsID(n) {
		return "cgo.Object"
	}

	// SEL → string (selectors are passed as Go strings and converted in bridge)
	if IsSEL(n) {
		return "string"
	}

	// Class → unsafe.Pointer (ObjC meta-class object)
	if IsClass(n) {
		return "unsafe.Pointer"
	}

	// BOOL → bool
	if IsBOOL(n) {
		return "bool"
	}

	// CF opaque pointer typedef (e.g. CFStringRef, CFArrayRef) — bare name with no *
	// because the pointer is implicit in the CF typedef definition.
	if cfTypedefSet[n] {
		return m.qualifiedCFType(n, ctx, imports)
	}
	// Framework-specific CF opaque types (e.g. CFHTTPMessageRef → CFNetwork).
	// These follow the same bare-name convention but live outside CoreFoundation.
	if owner, ok := m.CFTypeIndex[n]; ok {
		return m.qualifiedFrameworkCFType(n, owner, ctx, imports)
	}

	// Pointer types
	if IsPointer(n) {
		return m.resolvePointerType(n, ctx, imports)
	}

	// Bare ObjC generic type parameter (e.g. "ObjectType" in NSArray.objectAtIndex: return,
	// or "KeyType" in NSMutableDictionary.removeObjectForKey: arg). The scanner sets
	// IsGeneric=true only for nullable returns; non-nullable and arg uses fall through here.
	// Return cgo.Object — same as buildGoReturn does when IsGeneric=true — since T cannot
	// be constructed from unsafe.Pointer without a per-type factory in Go generics.
	if len(ctx.GenericParams) > 0 && !strings.ContainsAny(n, " *<>^()") {
		for _, gp := range ctx.GenericParams {
			if n == gp {
				return "cgo.Object"
			}
		}
	}

	// Non-pointer named type: struct, enum, or typedef.
	if name := ClassName(n); name != "" {
		// Struct value type registered by the loader (CGSize, CGRect,
		// NSOperatingSystemVersion, …). Same-framework references emit the
		// bare name; cross-framework references qualify with the owning
		// package alias. In return position the bridge malloc's the struct and
		// returns void*; the emitter handles dereference+free at the call site.
		if owner, ok := m.StructIndex[name]; ok {
			return m.qualifiedStructType(name, owner, false, ctx, imports)
		}
		// Enum type defined in this or another loaded framework.
		if owner, ok := m.EnumIndex[name]; ok {
			return m.qualifiedEnumType(name, owner, ctx, imports)
		}
	}
	// Typedef chain resolution — covers both ObjC-style uppercase names
	// (VZMemorySize, NSComparator) and C/POSIX lowercase names (dispatch_block_t,
	// ether_addr_t). Only attempt when n is a simple identifier (no spaces, *, <, ^).
	if !strings.ContainsAny(n, " *<>^()") {
		if target, ok := m.TypedefIndex[n]; ok && target != n {
			if resolved := m.resolveGoType(Normalise(target), ctx, imports); resolved != "unsafe.Pointer" {
				return resolved
			}
		}
		// C-style lowercase enum names (e.g. interface_event_t, vmnet_return_t) are
		// not detected by ClassName (which requires an uppercase first letter). Check
		// EnumIndex directly before falling through to unsafe.Pointer.
		if owner, ok := m.EnumIndex[n]; ok {
			return m.qualifiedEnumType(capitaliseFirst(n), owner, ctx, imports)
		}
		// Direct-name struct resolution for C-style lowercase struct names.
		if owner, ok := m.StructIndex[n]; ok {
			return m.qualifiedStructType(n, owner, false, ctx, imports)
		}
	}

	// "struct XYZ" bare form — strip the struct keyword and resolve the inner name.
	// This handles typedef targets like "struct ether_addr", "struct NSOperatingSystemVersion".
	if strings.HasPrefix(n, "struct ") {
		bare := strings.TrimSpace(n[7:])
		if bare != "" && !strings.ContainsAny(bare, " *<>^()") {
			if owner, ok := m.StructIndex[bare]; ok {
				return m.qualifiedStructType(bare, owner, false, ctx, imports)
			}
			// POSIX/BSD structs absent from SDK metadata: resolve to bsd.TypeName.
			if goName, ok := bsdStructTypes[bare]; ok {
				return m.qualifiedBSDType(goName, false, ctx, imports)
			}
		}
	}

	// C fixed-size array: "unsigned char [6]", "int32_t [4]" → [6]uint8, [4]int32.
	if bracketStart := strings.LastIndex(n, " ["); bracketStart > 0 {
		if bracketEnd := strings.Index(n[bracketStart:], "]"); bracketEnd > 0 {
			sizeStr := strings.TrimSpace(n[bracketStart+2 : bracketStart+bracketEnd])
			if size, err := strconv.Atoi(sizeStr); err == nil && size > 0 {
				baseType := strings.TrimSpace(n[:bracketStart])
				if goElem := m.resolveGoType(baseType, ctx, imports); goElem != "unsafe.Pointer" && goElem != "" {
					return fmt.Sprintf("[%d]%s", size, goElem)
				}
			}
		}
	}

	return "unsafe.Pointer"
}

// resolvePointerType handles types with at least one *.
func (m *Mapper) resolvePointerType(n string, ctx Context, imports ImportSet) string {
	// void * → unsafe.Pointer
	if strings.HasPrefix(n, "void *") || n == "void *" {
		return "unsafe.Pointer"
	}

	// char * → string
	if n == "char *" || n == "const char *" {
		return "string"
	}

	// BOOL * → *bool (nullable in-out / output BOOL parameter)
	if IsBOOLPointer(n) {
		return "*bool"
	}

	// NSError ** → handled upstream (method transforms to error return)
	// Still need a type for the bridge layer
	if strings.Contains(n, "NSError") && IsDoublePointer(n) {
		return "unsafe.Pointer"
	}

	// Generic collection: NSArray<ObjectType> *, NSDictionary<KeyType, ObjectType> *
	// Check if any of the type arguments is one of the current class's generic params.
	// Using GenericParams (not GenericParam) handles multi-param generics correctly.
	gps := GenericParams(n)
	if len(gps) > 0 {
		baseClass := ClassName(n)
		if baseClass != "" && ctx.ClassNameIndex[baseClass] {
			paramSet := make(map[string]bool, len(ctx.GenericParams))
			for _, p := range ctx.GenericParams {
				paramSet[p] = true
			}
			anyIsTypeVar := false
			for _, gp := range gps {
				if paramSet[gp] {
					anyIsTypeVar = true
					break
				}
			}
			if anyIsTypeVar {
				if ctx.IsClassMethod {
					// Class methods have no [T] in scope; use cgo.Object.
					return m.qualifiedType(baseClass, baseClass+"[cgo.Object]", ctx, imports)
				}
				return m.qualifiedType(baseClass, baseClass+"[T]", ctx, imports)
			}
			if m.GenericClasses[baseClass] {
				return m.qualifiedType(baseClass, baseClass+"[cgo.Object]", ctx, imports)
			}
			return m.qualifiedType(baseClass, baseClass, ctx, imports)
		}
	}

	// Named ObjC class pointer: NSString * → *NSString (or *foundation.NSString)
	cls := ClassName(n)
	if cls != "" {
		for _, gp := range ctx.GenericParams {
			if cls == gp {
				return "unsafe.Pointer"
			}
		}
		if ctx.ClassNameIndex[cls] {
			if m.GenericClasses[cls] {
				return m.qualifiedType(cls, cls+"[cgo.Object]", ctx, imports)
			}
			return m.qualifiedType(cls, cls, ctx, imports)
		}
		// Struct pointer (e.g. `CGSize *`, `NSOperatingSystemVersion *`):
		// In parameter position, resolve to `*<pkg>.<Name>` so callers can pass
		// typed Go struct pointers; goCGoArgExpr converts to unsafe.Pointer for CGo.
		// In return position, use unsafe.Pointer — the emitter cannot call a
		// New-constructor for a C struct (only ObjC classes have those), so the
		// caller must manage the raw pointer.
		if owner, ok := m.StructIndex[cls]; ok {
			if ctx.IsReturn {
				return "unsafe.Pointer"
			}
			return m.qualifiedStructType(cls, owner, true, ctx, imports)
		}
		// Typedef alias struct pointer: follow the typedef chain up to 8 hops to
		// find a known struct (e.g. NSRect*→CGRect*, AppleEvent*→AERecord*→
		// AEDescList*→AEDesc*→struct AEDesc). Single-hop cases (NSRect→CGRect)
		// still work on the first iteration; deep chains (Carbon/CoreServices)
		// resolve on later iterations.
		if target, ok := m.TypedefIndex[cls]; ok && target != cls {
			resolved := target
			seen := map[string]bool{cls: true, target: true}
			for range 8 {
				resolvedCls := ClassName(resolved)
				if resolvedCls == "" {
					// Handle "struct X" typedef targets.
					if strings.HasPrefix(resolved, "struct ") {
						resolvedCls = strings.TrimSpace(strings.TrimPrefix(resolved, "struct "))
					}
				}
				if resolvedCls != "" {
					if owner, ok := m.StructIndex[resolvedCls]; ok {
						if ctx.IsReturn {
							return "unsafe.Pointer"
						}
						return m.qualifiedStructType(resolvedCls, owner, true, ctx, imports)
					}
					if next, ok := m.TypedefIndex[resolvedCls]; ok && next != resolvedCls && !seen[next] {
						seen[next] = true
						resolved = next
						continue
					}
					// Typedef chain leads to a known ObjC class pointer
					// (e.g. MPSShape → NSArray<NSNumber *> → *foundation.NSArray[cgo.Object]).
					// Guard: skip if the target already ends with * to avoid double-pointer.
					if m.OwnerIndex[resolvedCls] != "" && !strings.HasSuffix(strings.TrimSpace(resolved), "*") {
						return m.resolvePointerType(strings.TrimSpace(resolved)+" *", ctx, imports)
					}
				}
				break
			}
			// Pointer to a typedef that resolves to a numeric primitive
			// (e.g. CMAttachmentMode * → uint32_t * → *uint32, CFIndex * → *int64).
			// GoBlockArgType returns the already-mapped Go type ("uint32", "int64", …),
			// so we exclude non-pointer-safe primitives ("string", "bool") explicitly.
			// In return position keep unsafe.Pointer — the bridge returns unsafe.Pointer
			// and the caller cannot assume ownership of the pointed-to memory.
			if g := m.GoBlockArgType(target); g != "" && g != "unsafe.Pointer" && g != "string" && g != "bool" {
				if ctx.IsReturn {
					return "unsafe.Pointer"
				}
				return "*" + g
			}
		}
		// Unknown class: fall back to unsafe.Pointer.
		return "unsafe.Pointer"
	}

	// C-style struct pointer with lowercase name: "struct vmnet_network *".
	// ClassName() requires an uppercase-starting name, so it returns "" above.
	// Strip the "struct " prefix and check StructIndex directly.
	bare := strings.TrimSpace(strings.TrimPrefix(n, "struct "))
	bare = strings.TrimSuffix(strings.TrimSpace(bare), "*")
	bare = strings.TrimSpace(bare)
	// Multi-word C primitive pointer (e.g. "unsigned char *" → *uint8).
	// bare contains a space so it can't enter the single-token block below;
	// handle it before the space-exclusion guard.
	if bare != "" && strings.Contains(bare, " ") && !strings.ContainsAny(bare, "*<>^()") {
		if goElem := primitiveGoType(bare); goElem != "" {
			if ctx.IsReturn {
				return "unsafe.Pointer"
			}
			return "*" + goElem
		}
	}
	if bare != "" && !strings.ContainsAny(bare, " *<>^()") {
		if owner, ok := m.StructIndex[bare]; ok {
			if ctx.IsReturn {
				return "unsafe.Pointer"
			}
			return m.qualifiedStructType(bare, owner, true, ctx, imports)
		}
		// POSIX/BSD value-type struct pointer (e.g. struct in_addr *):
		// resolve to *bsd.TypeName in param position; unsafe.Pointer in return
		// position (the bridge malloc's a copy, emitter can't dereference a
		// pointer-to-pointer here without a dedicated helper).
		if goName, ok := bsdStructTypes[bare]; ok {
			if ctx.IsReturn {
				return "unsafe.Pointer"
			}
			return m.qualifiedBSDType(goName, true, ctx, imports)
		}
		// Pointer to primitive (e.g. uint8_t *, int *): type as *T in parameter
		// position for output-parameter patterns. In return position keep
		// unsafe.Pointer — the caller cannot assume ownership of the pointed-to
		// memory (e.g. NSData -bytes returns const uint8_t *).
		if goElem := primitiveGoType(bare); goElem != "" {
			if ctx.IsReturn {
				return "unsafe.Pointer"
			}
			return "*" + goElem
		}
		// Pointer to enum (output-parameter pattern, e.g. vmnet_return_t *):
		// type as *EnumType in parameter position only.
		if owner, ok := m.EnumIndex[bare]; ok {
			if ctx.IsReturn {
				return "unsafe.Pointer"
			}
			return "*" + m.qualifiedEnumType(capitaliseFirst(bare), owner, ctx, imports)
		}
		// Pointer to a lowercase/underscore typedef that resolves to a primitive
		// (e.g. __CLPK_integer * → int * → *int32, __CLPK_real * → float * → *float32).
		if target, ok := m.TypedefIndex[bare]; ok && target != bare {
			if g := m.GoBlockArgType(target); g != "" && g != "unsafe.Pointer" && g != "string" && g != "bool" {
				if ctx.IsReturn {
					return "unsafe.Pointer"
				}
				return "*" + g
			}
		}
	}

	return "unsafe.Pointer"
}

