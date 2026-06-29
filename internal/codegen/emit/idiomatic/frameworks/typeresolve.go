//go:build darwin

package idiofw

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// idiomaticBlockParam converts an Objective-C block parameter into a Go function
// parameter plus an adapter that wraps the Go function as a block the runtime can
// call. It handles blocks whose arguments are objects, numbers, or a BOOL
// out-parameter (the "stop" flag of enumeration blocks), and whose result is
// void, a bool, or a number/ordering. ok is false for shapes it does not yet
// handle (struct arguments, nested blocks, object results).
func idiomaticBlockParam(
	pName, blockType string,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	fc *frameworkContext,
	rawPkgAlias string,
) (sig, adapter string, imports map[string]string, ok bool) {
	imports = map[string]string{"objc": objcImportPath}
	args := parseBlockObjCParams(blockType)

	var goParams, abiParams, callArgs []string
	for i, a := range args {
		v := fmt.Sprintf("_b%d", i)
		an := normaliseObjC(a)
		switch {
		case strings.HasPrefix(an, "BOOL") && strings.Contains(an, "*"):
			// The "stop" out-parameter: setting *stop = true halts enumeration.
			goParams = append(goParams, "*bool")
			abiParams = append(abiParams, v+" unsafe.Pointer")
			callArgs = append(callArgs, "(*bool)("+v+")")
			imports["unsafe"] = "unsafe"
			continue
		}
		gt := qualifyRaw(
			mapper.GoType(a, ctx, make(typemap.ImportSet)),
			fc,
			rawPkgAlias,
			ctx.GenericParams,
		)
		switch {
		case gt == "objc.ID" || isObjectGoType(gt, mapper):
			goParams = append(goParams, "obj.Object")
			abiParams = append(abiParams, v+" objc.ID")
			callArgs = append(callArgs, "obj.Wrap("+v+")")
			imports["obj"], imports["objref"] = objImportPath, objrefImportPath
		case gt == "bool":
			goParams = append(goParams, "bool")
			abiParams = append(abiParams, v+" bool")
			callArgs = append(callArgs, v)
		case isHermeticScalar(gt):
			t := goSizeType(gt)
			goParams = append(goParams, t)
			abiParams = append(abiParams, v+" "+t)
			callArgs = append(callArgs, v)
		default:
			return "", "", imports, false
		}
	}

	// Block result: the text before "(^".
	retObjC := ""
	if i := strings.Index(blockType, "(^"); i > 0 {
		retObjC = normaliseObjC(blockType[:i])
	}
	goRetSig, abiRetSig, retPrefix := "", "", ""
	switch {
	case retObjC == "" || retObjC == "void":
		// no result
	case retObjC == "BOOL" || retObjC == "bool":
		goRetSig, abiRetSig, retPrefix = " bool", " bool", "return "
	case !strings.Contains(retObjC, "*"):
		// An ordering or other integer result (e.g. NSComparisonResult).
		goRetSig, abiRetSig, retPrefix = " int", " int", "return "
	default:
		return "", "", imports, false
	}

	sig = "func(" + strings.Join(goParams, ", ") + ")" + goRetSig
	body := retPrefix + pName + "(" + strings.Join(callArgs, ", ") + ")"
	adapter = "objc.NewBlock(func(_ objc.Block"
	if len(abiParams) > 0 {
		adapter += ", " + strings.Join(abiParams, ", ")
	}
	adapter += ")" + abiRetSig + " { " + body + " })"
	return sig, adapter, imports, true
}

// idiomaticArg decides how one Objective-C method parameter appears in the
// generated Go signature (sig) and how its Go value is passed to the Objective-C
// call (arg). It also reports any extra imports the conversion needs. ok is
// false when the parameter type cannot yet be expressed without exposing an
// Objective-C runtime type, in which case the caller skips the whole method.
func idiomaticArg(
	pName, objcType string,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	fc *frameworkContext,
	rawPkgAlias string,
	trialNames trialNameMap,
) (sig, arg string, imports map[string]string, ok bool) {
	imports = map[string]string{}
	norm := normaliseObjC(objcType)
	switch {
	case isNSStringType(norm):
		imports["purego"] = pureobjcImportPath
		return "string", "purego.NSString(" + pName + ")", imports, true
	case isNSURLType(norm):
		imports["rt"] = rtImportPath
		return "string", "rt.FileURL(" + pName + ")", imports, true
	}
	// A block parameter becomes a Go function.
	if strings.Contains(objcType, "(^") {
		sig, adapter, bimps, bok := idiomaticBlockParam(pName, objcType, ctx, mapper, fc, rawPkgAlias)
		if !bok {
			return "", "", imports, false
		}
		maps.Copy(imports, bimps)
		return sig, adapter, imports, true
	}
	// An array parameter is accepted as a Go slice and converted to an
	// Objective-C array.
	if looksLikeNSArray(objcType) {
		if elemObjC := extractNSArrayElem(objcType); elemObjC != "" {
			elemGo, _, toID, eimps := arrayElemConv(elemObjC, ctx, mapper, fc, rawPkgAlias, trialNames)
			maps.Copy(imports, eimps)
			conv := "func(_v " + elemGo + ") objc.ID { return " + fmt.Sprintf(toID, "_v") + " }"
			return "[]" + elemGo, "purego.SliceToNSArray(" + pName + ", " + conv + ")", imports, true
		}
	}
	impSet := make(typemap.ImportSet)
	goType := qualifyRaw(
		rawParamGoType(objcType, ctx, mapper, impSet),
		fc,
		rawPkgAlias,
		ctx.GenericParams,
	)
	if goType == "" {
		return "", "", imports, false
	}
	// A CoreFoundation reference that is a pointer is a toll-free-bridged object.
	// (Some "…Ref" types are really integer handles, e.g. MIDIObjectRef; those
	// resolve to a number and are handled as scalars below.)
	if goType == "unsafe.Pointer" && isCFObjectType(objcType, mapper) {
		imports["obj"] = objImportPath
		imports["objref"] = objrefImportPath
		return "obj.Object", "objref.IDOf(" + pName + ")", imports, true
	}
	// A bare host pointer input (e.g. void *addr in hv_vm_map) is passed through
	// unchanged. cfuncABIType keeps it as unsafe.Pointer, matching the raw binding.
	// Pointer out-parameters are lifted to return values before idiomaticArg runs,
	// so an unsafe.Pointer reaching here is a genuine input pointer.
	if goType == "unsafe.Pointer" {
		imports["unsafe"] = "unsafe"
		return "unsafe.Pointer", pName, imports, true
	}
	if sigEnum, _, isEnum := localizeEnumType(goType, fc, rawPkgAlias); isEnum {
		// A Go enum type is an integer; pass it to the call unchanged.
		return sigEnum, pName, imports, true
	}
	if base, has := trialWrapClass(goType, rawPkgAlias); has {
		if tt, named := trialNames[base]; named {
			imports["objref"] = objrefImportPath
			return "*" + tt, "objref.IDOf(" + pName + ")", imports, true
		}
	}
	if isObjectGoType(goType, mapper) {
		imports["obj"] = objImportPath
		imports["objref"] = objrefImportPath
		return "obj.Object", "objref.IDOf(" + pName + ")", imports, true
	}
	if sname, ok := localValueStructName(goType, fc, rawPkgAlias); ok {
		// A value struct is passed to Objective-C by value, unchanged.
		return sname, pName, imports, true
	}
	if sname, simps, ok := crossFrameworkValueStruct(goType, mapper, rawPkgAlias); ok {
		// A value struct owned by another framework, passed by value.
		maps.Copy(imports, simps)
		return sname, pName, imports, true
	}
	if isHermeticScalar(goType) {
		return goSizeType(goType), pName, imports, true
	}
	return "", "", imports, false
}

// outParamGoType reports whether objcType is a value out-parameter — a pointer
// to a scalar or enum that the Objective-C call writes through (for example
// "NSPropertyListFormat *"). It returns the idiomatic Go type of the pointed-to
// value and true. Object pointers, double pointers, blocks, and NSError (handled
// as a returned error) are not value out-parameters and return ("", false).
// allowStructOut lets the generic C-function pass lift a pointer-to-value-struct
// out-parameter (e.g. hv_vcpu_create's hv_vcpu_exit_t **) into a returned struct
// or *struct. Methods pass false: their out-param machinery has an error/zero
// path that only handles scalar, enum, and bool zeros, so a struct out there
// would emit an invalid "0" zero literal.
func outParamGoType(
	objcType string,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	fc *frameworkContext,
	rawPkgAlias string,
	allowStructOut bool,
) (string, bool) {
	norm := normaliseObjC(objcType)
	if strings.Contains(objcType, "(^") || strings.Contains(norm, "**") ||
		strings.HasPrefix(norm, "NSError") {
		return "", false
	}
	resolved := qualifyRaw(
		rawParamGoType(objcType, ctx, mapper, make(typemap.ImportSet)),
		fc,
		rawPkgAlias,
		ctx.GenericParams,
	)
	// Only a single-level pointer-to-value is a value out-parameter. The raw
	// mapper collapses "<value> **" to "*<value>" (e.g. hv_vcpu_exit_t ** →
	// *HvVcpuExitT), so a genuine double pointer to a value still reads as "**…"
	// here and is rejected.
	if !strings.HasPrefix(resolved, "*") || strings.HasPrefix(resolved, "**") {
		return "", false
	}
	base := resolved[1:]
	if base == "" || strings.HasPrefix(base, "*") { // not a pointer-to-value
		return "", false
	}
	if sigEnum, _, isEnum := localizeEnumType(base, fc, rawPkgAlias); isEnum {
		return sigEnum, true
	}
	if base == "bool" {
		return "bool", true
	}
	if isHermeticScalar(base) {
		return goSizeType(base), true
	}
	// A pointer to a value struct this framework re-declares locally (e.g.
	// hv_vcpu_exit_t). The local name keeps the idiomatic layer hermetic. The
	// emittable set (built from resolved Go types) is authoritative — fc.localStruct
	// relies on a scanner-recorded field GoType that some metadata omits. When the
	// original ObjC type is a double pointer (hv_vcpu_exit_t **) the callee writes a
	// pointer it owns, so the out value is the typed pointer (*Struct); a single
	// pointer fills a caller buffer, so the out value is the struct itself. The raw
	// mapper collapses the ObjC "**" to one "*", so count asterisks on objcType.
	if allowStructOut && strings.HasPrefix(base, rawPkgAlias+".") {
		sname := base[len(rawPkgAlias)+1:]
		if sname != "" && !strings.ContainsAny(sname, ".*[]") &&
			fc.ownTypes[sname] && mapper.EmittableStructs[sname] {
			if strings.Count(objcType, "*") >= 2 {
				return "*" + sname, true
			}
			return sname, true
		}
	}
	return "", false
}

// zeroLiteral returns the zero value literal for an out-parameter's Go type
// (used on the error-return path of a method that has out-parameters).
func zeroLiteral(goType string) string {
	switch goType {
	case "bool":
		return "false"
	case "string":
		return `""`
	default:
		if strings.HasPrefix(goType, "*") {
			return "nil" // a lifted pointer out-value (e.g. *hvraw.HvVcpuExitT)
		}
		return "0" // scalars and enums (named integer types)
	}
}

// goSizeType maps Objective-C's NSUInteger (resolved to Go uint) to int, the Go
// convention for sizes and indices. Other scalar types are returned unchanged.
func goSizeType(goType string) string {
	if goType == "uint" {
		return "int"
	}
	return goType
}

// idiomaticRet decides the generated method's return type and how the call's
// result is converted back to Go. kind selects the conversion; wrap is the
// expression that builds an object result (with one %s for the result pointer);
// sendType is the type the call itself returns. ok is false for a result type
// that cannot yet be expressed without an Objective-C runtime type.
func idiomaticRet(
	objcType string,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	fc *frameworkContext,
	rawPkgAlias string,
	trialNames trialNameMap,
) (retType string, kind objKind, wrap, sendType string, imports map[string]string, ok bool) {
	imports = map[string]string{}
	ret := strings.TrimSpace(objcType)
	if ret == "" || ret == "void" {
		// A void method still needs a type for the call; the result is discarded.
		return "", kindVoid, "", "objc.ID", imports, true
	}
	if isNSStringType(normaliseObjC(objcType)) {
		imports["purego"] = pureobjcImportPath
		return "string", kindString, "", "objc.ID", imports, true
	}
	// An array return is surfaced as a Go slice.
	if looksLikeNSArray(objcType) {
		if elemObjC := extractNSArrayElem(objcType); elemObjC != "" {
			elemGo, fromID, _, eimps := arrayElemConv(elemObjC, ctx, mapper, fc, rawPkgAlias, trialNames)
			// Only the element fromID conversion (obj.Wrap / <T>FromID /
			// purego.GoString) runs on a return; objref.IDOf belongs to the
			// parameter (toID) direction, so it is not imported here.
			delete(eimps, "objref")
			maps.Copy(imports, eimps)
			conv := "func(_id objc.ID) " + elemGo + " { return " + fmt.Sprintf(fromID, "_id") + " }"
			wrap := "purego.NSArrayToSlice(%s, " + conv + ")"
			return "[]" + elemGo, kindArray, wrap, "objc.ID", imports, true
		}
	}
	if _, isBlock := mapper.ResolveBlockSignature(ret); isBlock {
		return "", kindVoid, "", "", imports, false
	}
	impSet := make(typemap.ImportSet)
	goRet := qualifyRaw(mapper.GoReturnType(objcType, ctx, impSet), fc, rawPkgAlias, ctx.GenericParams)
	if goRet == "" {
		return "", kindVoid, "", "", imports, false
	}
	// A CoreFoundation reference that is a pointer is a toll-free-bridged object;
	// integer-handle "…Ref" types fall through to the scalar case.
	if goRet == "unsafe.Pointer" && isCFObjectType(objcType, mapper) {
		imports["obj"] = objImportPath
		return "obj.Object", kindObject, "obj.Wrap(%s)", "objc.ID", imports, true
	}
	// An os_object handle returned already retained (OS_OBJECT_RETURNS_RETAINED,
	// e.g. hv_vm_config_create) is surfaced as obj.Object and ADOPTED — obj.Adopt
	// takes ownership of the +1 reference without retaining a second time.
	if goRet == "unsafe.Pointer" && isOSObjectReturn(objcType) {
		imports["obj"] = objImportPath
		return "obj.Object", kindObject, "obj.Adopt(%s)", "objc.ID", imports, true
	}
	if goRet == "bool" {
		return "bool", kindBool, "", "bool", imports, true
	}
	if sigEnum, _, isEnum := localizeEnumType(goRet, fc, rawPkgAlias); isEnum {
		return sigEnum, kindEnum, "", sigEnum, imports, true
	}
	if base, has := trialWrapClass(goRet, rawPkgAlias); has {
		if tt, named := trialNames[base]; named {
			imports["objref"] = objrefImportPath
			return "*" + tt, kindObject, tt + "FromID(%s)", "objc.ID", imports, true
		}
	}
	if isObjectGoType(goRet, mapper) {
		imports["obj"] = objImportPath
		return "obj.Object", kindObject, "obj.Wrap(%s)", "objc.ID", imports, true
	}
	if sname, ok := localValueStructName(goRet, fc, rawPkgAlias); ok {
		// A value struct is returned by value, unchanged.
		return sname, kindScalar, "", sname, imports, true
	}
	if sname, simps, ok := crossFrameworkValueStruct(goRet, mapper, rawPkgAlias); ok {
		// A value struct owned by another framework, returned by value.
		maps.Copy(imports, simps)
		return sname, kindScalar, "", sname, imports, true
	}
	if isHermeticScalar(goRet) {
		t := goSizeType(goRet)
		return t, kindScalar, "", t, imports, true
	}
	return "", kindVoid, "", "", imports, false
}

// localValueStructName reports the local value-struct name for a resolved Go
// type of the form "<rawAlias>.<Struct>", when <Struct> is one the idiomatic
// package re-declares locally. The struct is passed to and from Objective-C by
// value, so the call uses the local type directly.
// crossFrameworkValueStruct recognises a value struct owned by another framework
// (e.g. "corefoundation.CGRect" referenced from vision). Such a struct is emitted
// in its owning idiomatic package, so it is named there and passed by value. It
// returns the qualified Go type unchanged plus the import needed to name it. ok
// is false for same-package types, pointers/slices, and names that are not known
// value structs.
func crossFrameworkValueStruct(
	goType string,
	mapper *typemap.Mapper,
	rawPkgAlias string,
) (string, map[string]string, bool) {
	// Only a plain "pkg.Name" reference is a by-value struct; a pointer or slice
	// (e.g. "*corefoundation.CGPoint") is something else and is left alone.
	if strings.ContainsAny(goType, "*[] ") {
		return "", nil, false
	}
	dot := strings.IndexByte(goType, '.')
	if dot <= 0 {
		return "", nil, false
	}
	pkg, name := goType[:dot], goType[dot+1:]
	if pkg == rawPkgAlias || strings.Contains(name, ".") {
		return "", nil, false
	}
	// Only reference a struct that the owning package actually emits, so the
	// reference never dangles.
	if !mapper.EmittableStructs[name] {
		return "", nil, false
	}
	return goType, map[string]string{pkg: idiomaticFrameworkPrefix + pkg}, true
}

func localValueStructName(goType string, fc *frameworkContext, rawPkgAlias string) (string, bool) {
	prefix := rawPkgAlias + "."
	if !strings.HasPrefix(goType, prefix) {
		return "", false
	}
	name := goType[len(prefix):]
	if strings.ContainsAny(name, ".*[]") {
		return "", false
	}
	if fc.localStruct[name] {
		return name, true
	}
	return "", false
}

// arrayElemConv decides how the elements of an Objective-C array convert to and
// from Go. elemGoType is the Go element type; fromID converts an element pointer
// to Go (format with one %s); toID converts a Go element to a pointer (format
// with one %s). It always succeeds: an unrecognised element is treated as a
// generic object.
func arrayElemConv(
	elemObjC string,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	fc *frameworkContext,
	rawPkgAlias string,
	trialNames trialNameMap,
) (elemGoType, fromID, toID string, imports map[string]string) {
	imports = map[string]string{"purego": pureobjcImportPath, "objc": objcImportPath}
	if isNSStringType(normaliseObjC(elemObjC)) {
		return "string", "purego.GoString(%s)", "purego.NSString(%s)", imports
	}
	impSet := make(typemap.ImportSet)
	goElem := qualifyRaw(mapper.GoType(elemObjC, ctx, impSet), fc, rawPkgAlias, ctx.GenericParams)
	if base, has := trialWrapClass(goElem, rawPkgAlias); has {
		if tt, named := trialNames[base]; named {
			imports["objref"] = objrefImportPath
			return "*" + tt, tt + "FromID(%s)", "objref.IDOf(%s)", imports
		}
	}
	imports["obj"] = objImportPath
	imports["objref"] = objrefImportPath
	return "obj.Object", "obj.Wrap(%s)", "objref.IDOf(%s)", imports
}

// isCFObjectType reports whether an Objective-C type is a CoreFoundation opaque
// reference (e.g. CFStringRef, CGColorRef, SecKeyRef). These are toll-free
// bridged to Objective-C objects, so the idiomatic layer surfaces them as
// obj.Object.
func isCFObjectType(objcType string, mapper *typemap.Mapper) bool {
	t := strings.TrimPrefix(normaliseObjC(objcType), "const ")
	t = strings.TrimSpace(t)
	if strings.Contains(t, "*") {
		return false // a pointer to a ref is an out-parameter, not a ref value
	}
	fields := strings.Fields(t)
	if len(fields) == 0 {
		return false
	}
	name := fields[len(fields)-1]
	if typemap.IsCoreFoundationOpaqueRef(name) {
		return true
	}
	if mapper.CFTypeIndex != nil {
		if _, ok := mapper.CFTypeIndex[name]; ok {
			return true
		}
	}
	return false
}

// isOSObjectReturn reports whether an Objective-C return type is an os_object
// handle the function hands back already retained (its declaration carries the
// OS_OBJECT_RETURNS_RETAINED ownership attribute, e.g. hv_vm_config_t). These
// typedefs resolve to Objective-C objects, so the idiomatic layer surfaces them
// as obj.Object — adopting the +1 reference rather than retaining a second time.
func isOSObjectReturn(objcType string) bool {
	return strings.Contains(objcType, "OS_OBJECT_RETURNS_RETAINED")
}

// isObjectGoType reports whether a resolved Go type refers to an Objective-C
// object: a pointer to a known class, or a bare object handle.
func isObjectGoType(goType string, mapper *typemap.Mapper) bool {
	if goType == "objc.ID" {
		return true
	}
	return isObjectPointerType(goType, mapper)
}

// isHermeticScalar reports whether a Go type is a plain value (an integer,
// float, or bool) that needs no conversion and names no other package. It
// rejects pointers, generics, and anything package-qualified (which would carry
// a raw/runtime type into a public signature).
func isHermeticScalar(goType string) bool {
	if goType == "" {
		return false
	}
	return !strings.ContainsAny(goType, ".*[]")
}

// trialWrapClass reports the bare class name when rawRet is a non-generic single
// pointer into this package's raw alias (e.g. "*raw.NSWindow" → "NSWindow", ok).
// Generic instantiations, slices, double pointers and cross-package types return
// ("", false) so the wrapper keeps the raw return type.
func trialWrapClass(rawRet, rawPkgAlias string) (string, bool) {
	prefix := "*" + rawPkgAlias + "."
	if !strings.HasPrefix(rawRet, prefix) {
		return "", false
	}
	base := strings.TrimPrefix(rawRet, prefix)
	if strings.ContainsAny(base, "[].* ") {
		return "", false
	}
	return base, true
}

// rawParamGoType resolves a raw method/function parameter type exactly as the
// raw emitter does: block params go through the shared adapter model in
// package emit so the idiomatic layer's closures keep matching the raw
// signatures; everything else uses the type mapper directly.
func rawParamGoType(
	objcType string,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	impSet typemap.ImportSet,
) string {
	if _, isBlock := mapper.ResolveBlockSignature(objcType); isBlock {
		return rawfw.BlockGoFuncType(objcType, ctx, mapper, impSet, mapper.OwnerIndex)
	}
	resolved := mapper.GoType(objcType, ctx, impSet)
	if resolved == "" {
		resolved = "unsafe.Pointer"
	}
	return resolved
}

func isAsyncCompletion(method meta.Method) bool {
	if strings.TrimSpace(method.Return.ObjCType) != "void" || len(method.Params) == 0 {
		return false
	}
	last := method.Params[len(method.Params)-1]
	if !last.IsBlock {
		return false
	}
	t := strings.TrimSpace(last.ObjCType)
	return strings.HasPrefix(t, "void (^") && !strings.Contains(t, "BOOL *")
}

func parseBlockObjCParams(blockObjCType string) []string {
	idx := strings.Index(blockObjCType, ")(")
	if idx < 0 {
		return nil
	}
	after := blockObjCType[idx+2:]
	end := strings.LastIndex(after, ")")
	if end < 0 {
		return nil
	}
	inner := strings.TrimSpace(after[:end])
	if inner == "" || inner == "void" {
		return nil
	}
	var parts []string
	depth, start := 0, 0
	for i, r := range inner {
		switch r {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(inner[start:i]))
				start = i + 1
			}
		}
	}
	if tail := strings.TrimSpace(inner[start:]); tail != "" {
		parts = append(parts, tail)
	}
	return parts
}

func looksLikeNSArray(objcType string) bool {
	if strings.Contains(objcType, "(^") {
		return false
	}
	t := normaliseObjC(objcType)
	return strings.HasPrefix(t, "NSArray") || strings.HasPrefix(t, "NSMutableArray")
}

func extractNSArrayElem(objcType string) string {
	_, after, ok := strings.Cut(objcType, "<")
	if !ok {
		return ""
	}
	end := strings.LastIndex(after, ">")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(after[:end])
}

func isNSErrorType(objcType string) bool {
	t := normaliseObjC(objcType)
	return strings.HasPrefix(t, "NSError") && strings.Contains(t, "*")
}

func isNSURLType(objcType string) bool {
	if strings.Contains(objcType, "(^") || strings.Contains(objcType, "(*") {
		return false // block or C function pointer returning NSURL, not an NSURL value
	}
	t := normaliseObjC(objcType)
	return t == "NSURL *" || strings.HasPrefix(t, "NSURL *") || t == "NSURL"
}

func isNSStringType(objcType string) bool {
	if strings.Contains(objcType, "(^") || strings.Contains(objcType, "(*") {
		return false // block or C function pointer returning NSString, not an NSString value
	}
	t := normaliseObjC(objcType)
	rest, ok := strings.CutPrefix(t, "NSString")
	if !ok {
		return false
	}
	// Require a type boundary after "NSString" so longer identifiers such as
	// NSStringEncoding (a uint out-param) or NSStringDrawingContext are not
	// mistaken for an NSString and converted to a Go string.
	if rest != "" && isIdentByte(rest[0], false) {
		return false
	}
	return strings.Contains(t, "*")
}

func normaliseObjC(t string) string {
	t = strings.TrimSpace(t)
	for _, q := range []string{
		"_Nullable ", "_Nonnull ", "_Null_unspecified ", "__kindof ",
		"__unsafe_unretained ", "__autoreleasing ", "__strong ", "__weak ",
	} {
		t = strings.TrimPrefix(t, q)
	}
	for _, q := range []string{" _Nullable", " _Nonnull", " _Null_unspecified"} {
		t = strings.TrimSuffix(t, q)
	}
	return strings.TrimSpace(t)
}

// qualifyRaw rewrites a mapper-resolved Go type for use inside a trial package:
//   - package references to the current framework become the raw alias
//   - bare same-framework identifiers (classes, enums, structs, protocols) get
//     the raw alias prefix, wherever they appear (generic brackets, func types)
//   - leaked generic param names (e.g. ObjectType) become objc.ID, matching the
//     instantiation trial uses for generic raw classes
func qualifyRaw(
	goType string,
	fc *frameworkContext,
	rawPkgAlias string,
	genericParams []string,
) string {
	fwPkg := strings.ToLower(fc.framework.Framework)

	var sb strings.Builder
	for i := 0; i < len(goType); {
		if !isIdentByte(goType[i], true) {
			sb.WriteByte(goType[i])
			i++
			continue
		}
		j := i + 1
		for j < len(goType) && isIdentByte(goType[j], false) {
			j++
		}
		word := goType[i:j]
		prevIsDot := i > 0 && goType[i-1] == '.'
		nextIsDot := j < len(goType) && goType[j] == '.'
		switch {
		case nextIsDot && !prevIsDot && word == fwPkg:
			// The mapper qualified a type with the framework's own package
			// name — rewrite to the raw alias. Token-scoped so substring
			// packages (corefoundation vs foundation) are untouched.
			sb.WriteString(rawPkgAlias)
		case prevIsDot || nextIsDot:
			sb.WriteString(word)
		case identInList(word, genericParams):
			sb.WriteString("objc.ID")
		case fc.ownTypes[word]:
			sb.WriteString(rawPkgAlias)
			sb.WriteByte('.')
			sb.WriteString(word)
		default:
			sb.WriteString(word)
		}
		i = j
	}
	return sb.String()
}

func isIdentByte(c byte, start bool) bool {
	if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' {
		return true
	}
	return !start && c >= '0' && c <= '9'
}

func identInList(word string, list []string) bool {
	return slices.Contains(list, word)
}

// enumHasAvailableMember reports whether e has at least one available member,
// matching the condition under which emitEnums emits a concrete local type.
func enumHasAvailableMember(enum meta.Enum) bool {
	for _, member := range enum.Members {
		if !member.Availability.IsUnavailable {
			return true
		}
	}
	return false
}

// localizeEnumType rewrites a raw-qualified scalar enum type (rawPkgAlias.<E>,
// where <E> is one of framework's own enums) to the local idiomatic spelling <E>,
// recording it as referenced so emitEnums emits its concrete definition. It
// returns (localType, rawType, true); for anything else it returns the input
// unchanged with isEnum=false. Only the exact alias.<E> form matches, so
// pointer/slice/compound types keep their raw spelling.
func localizeEnumType(
	qualified string,
	fc *frameworkContext,
	rawPkgAlias string,
) (sigType, rawType string, isEnum bool) {
	prefix := rawPkgAlias + "."
	if !strings.HasPrefix(qualified, prefix) {
		return qualified, "", false
	}
	name := qualified[len(prefix):]
	if !fc.ownEnums[name] {
		return qualified, "", false
	}
	fc.referenced[name] = true
	return deprefixEnumName(name, fc.prefix), qualified, true
}
