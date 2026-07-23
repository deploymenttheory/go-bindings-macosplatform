//go:build darwin

package idiofw

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	rawfw "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks"
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
	pName string,
	block typemap.BlockSignature,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	fc *frameworkContext,
	rawPkgAlias string,
) (sig, adapter string, imports map[string]string, ok bool) {
	imports = map[string]string{"objc": objcImportPath}
	args := block.ParamObjCTypes

	var goParams, abiParams, callArgs []string
	for i, a := range args {
		v := fmt.Sprintf("_b%d", i)
		an := normaliseObjC(a)
		if strings.HasPrefix(an, "BOOL") && strings.Contains(an, "*") {
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
			// An enum block argument (vmnet's void(^)(vmnet_return_t, …)) is an
			// integer; surface it with its localized Go enum type and pass it
			// through unchanged, mirroring idiomaticArg's enum handling.
			if sigEnum, _, isEnum := localizeEnumType(gt, fc, rawPkgAlias); isEnum {
				goParams = append(goParams, sigEnum)
				abiParams = append(abiParams, v+" "+sigEnum)
				callArgs = append(callArgs, v)
				continue
			}
			// A bare host pointer block argument (a void* context/refcon, e.g. an
			// IOUSBHostInterestHandler's messageArgument) is passed through unchanged,
			// mirroring the unsafe.Pointer input passthrough in idiomaticArg.
			if gt == "unsafe.Pointer" {
				goParams = append(goParams, "unsafe.Pointer")
				abiParams = append(abiParams, v+" unsafe.Pointer")
				callArgs = append(callArgs, v)
				imports["unsafe"] = "unsafe"
				continue
			}
			return "", "", imports, false
		}
	}

	// Block result.
	retObjC := normaliseObjC(block.ReturnObjCType)
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

// osObjectRegistry maps an os_object handle typedef to the concrete idiomatic
// library type the libraries emitter produces for it. These are the dispatch/xpc
// handles a framework function or method takes as a parameter; surfacing them as
// their real library type (rather than a generic obj.Object) lets a caller pass
// the dispatch.Queue / xpc.Object a library constructor returned without a manual
// obj.Adopt bridge. Only handles with an unambiguous library counterpart are
// listed; anything else keeps the obj.Object shape.
var osObjectRegistry = map[string]struct{ pkg, typ string }{
	"dispatch_queue_t":     {"dispatch", "Queue"},
	"dispatch_data_t":      {"dispatch", "Data"},
	"dispatch_group_t":     {"dispatch", "Group"},
	"dispatch_semaphore_t": {"dispatch", "Semaphore"},
	"dispatch_io_t":        {"dispatch", "Io"},
	"dispatch_object_t":    {"dispatch", "Object"},
	"os_workgroup_t":       {"dispatch", "OsWorkgroup"},
	"xpc_object_t":         {"xpc", "Object"},
	"xpc_connection_t":     {"xpc", "Connection"},
	"xpc_endpoint_t":       {"xpc", "Endpoint"},
	"xpc_activity_t":       {"xpc", "Activity"},
}

// osObjectLibraryType reports whether objcType is an os_object handle typedef with
// a concrete idiomatic library type, returning that Go type ("dispatch.Queue"),
// its package name, and the package import path. The lookup is on the bare typedef
// name, stripped of const/pointer/nullability qualifiers.
func osObjectLibraryType(objcType string) (goType, pkg, importPath string, ok bool) {
	bare := objcType
	for _, q := range []string{"const", "volatile", "_Nonnull", "_Nullable", "_Null_unspecified", "*"} {
		bare = strings.ReplaceAll(bare, q, " ")
	}
	fields := strings.Fields(bare)
	if len(fields) != 1 {
		return "", "", "", false
	}
	entry, found := osObjectRegistry[fields[0]]
	if !found {
		return "", "", "", false
	}
	return entry.pkg + "." + entry.typ, entry.pkg, idiomaticLibraryPrefix + entry.pkg, true
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
	case isNSDateType(norm):
		imports["rt"] = rtImportPath
		imports["time"] = "time"
		return "time.Time", "rt.TimeToNSDate(" + pName + ")", imports, true
	case isNSDataType(norm):
		imports["rt"] = rtImportPath
		return "[]byte", "rt.BytesToNSData(" + pName + ")", imports, true
	}
	// A block parameter — a literal "(^" type or a typedef chain ending in one
	// (vmnet's vmnet_interface_event_callback_t) — becomes a Go function.
	if bsig, isBlock := mapper.ResolveBlockSignature(objcType); isBlock {
		s, adapter, bimps, bok := idiomaticBlockParam(
			pName,
			bsig,
			ctx,
			mapper,
			fc,
			rawPkgAlias,
		)
		if !bok {
			return "", "", imports, false
		}
		maps.Copy(imports, bimps)
		return s, adapter, imports, true
	}
	// An array parameter is accepted as a Go slice and converted to an
	// Objective-C array.
	if looksLikeNSArray(objcType) {
		if elemObjC := extractNSArrayElem(objcType); elemObjC != "" {
			elemGo, _, toID, eimps := arrayElemConv(
				elemObjC,
				ctx,
				mapper,
				fc,
				rawPkgAlias,
				trialNames,
			)
			maps.Copy(imports, eimps)
			imports["purego"] = pureobjcImportPath // purego.SliceToNSArray
			conv := "func(_v " + elemGo + ") objc.ID { return " + fmt.Sprintf(toID, "_v") + " }"
			return "[]" + elemGo, "purego.SliceToNSArray(" + pName + ", " + conv + ")", imports, true
		}
	}
	// A string-keyed generic dictionary parameter is accepted as a Go map and
	// converted to an Objective-C dictionary. Non-string keys and ungenericized
	// dictionaries keep today's obj.Object shape (a Go map key conversion for an
	// arbitrary object key would silently misdecode).
	if looksLikeNSDictionary(objcType) {
		if keyObjC, valueObjC, kvOK := extractDictKV(
			objcType,
		); kvOK &&
			isNSStringType(normaliseObjC(keyObjC)) {
			valueGo, _, valueToID, vimps := arrayElemConv(
				valueObjC,
				ctx,
				mapper,
				fc,
				rawPkgAlias,
				trialNames,
			)
			maps.Copy(imports, vimps)
			imports["rt"] = rtImportPath
			imports["purego"] = pureobjcImportPath // purego.NSString in the key conversion
			keyConv := "func(_k string) objc.ID { return purego.NSString(_k) }"
			valueConv := "func(_v " + valueGo + ") objc.ID { return " + fmt.Sprintf(
				valueToID,
				"_v",
			) + " }"
			return "map[string]" + valueGo,
				"rt.MapToDict(" + pName + ", " + keyConv + ", " + valueConv + ")",
				imports, true
		}
	}
	// A set parameter is accepted as a Go slice and converted to an
	// Objective-C set.
	if looksLikeNSSet(objcType) {
		if elemObjC := extractNSArrayElem(objcType); elemObjC != "" {
			elemGo, _, toID, eimps := arrayElemConv(
				elemObjC,
				ctx,
				mapper,
				fc,
				rawPkgAlias,
				trialNames,
			)
			maps.Copy(imports, eimps)
			imports["rt"] = rtImportPath
			conv := "func(_v " + elemGo + ") objc.ID { return " + fmt.Sprintf(toID, "_v") + " }"
			return "[]" + elemGo, "rt.SliceToNSSet(" + pName + ", " + conv + ")", imports, true
		}
	}
	// An os_object handle typedef (dispatch_queue_t, xpc_object_t) is surfaced as
	// the concrete idiomatic library type (dispatch.Queue, xpc.Object) so callers
	// pass the value the library constructor already handed them, instead of an
	// opaque obj.Object they would have to hand-bridge. The C ABI wants the
	// underlying pointer as an objc.ID.
	if goT, pkg, importPath, ok := osObjectLibraryType(objcType); ok {
		imports[pkg] = importPath
		imports["objc"] = objcImportPath
		return goT, "objc.ID(uintptr(" + pName + ".Ptr()))", imports, true
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
		if strings.HasPrefix(goType, "*") || goType == "unsafe.Pointer" {
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
	// A URL return is surfaced as a Go string: the filesystem path for a file
	// URL (round-tripping with the rt.FileURL parameter conversion), the
	// absolute URL string otherwise.
	if isNSURLType(normaliseObjC(objcType)) {
		imports["rt"] = rtImportPath
		return "string", kindObject, "rt.URLString(%s)", "objc.ID", imports, true
	}
	// A date return is surfaced as a Go time.Time (the zero time.Time for nil).
	if isNSDateType(normaliseObjC(objcType)) {
		imports["rt"] = rtImportPath
		imports["time"] = "time"
		return "time.Time", kindObject, "rt.NSDateToTime(%s)", "objc.ID", imports, true
	}
	// A data return is surfaced as a Go byte slice (copied; nil for nil data).
	if isNSDataType(normaliseObjC(objcType)) {
		imports["rt"] = rtImportPath
		return "[]byte", kindObject, "rt.NSDataToBytes(%s)", "objc.ID", imports, true
	}
	// An array return is surfaced as a Go slice.
	if looksLikeNSArray(objcType) {
		if elemObjC := extractNSArrayElem(objcType); elemObjC != "" {
			elemGo, fromID, _, eimps := arrayElemConv(
				elemObjC,
				ctx,
				mapper,
				fc,
				rawPkgAlias,
				trialNames,
			)
			// Only the element fromID conversion (obj.Wrap / <T>FromID /
			// purego.GoString) runs on a return; objref.IDOf belongs to the
			// parameter (toID) direction, so it is not imported here.
			delete(eimps, "objref")
			maps.Copy(imports, eimps)
			imports["purego"] = pureobjcImportPath // purego.NSArrayToSlice
			conv := "func(_id objc.ID) " + elemGo + " { return " + fmt.Sprintf(fromID, "_id") + " }"
			wrap := "purego.NSArrayToSlice(%s, " + conv + ")"
			return "[]" + elemGo, kindArray, wrap, "objc.ID", imports, true
		}
	}
	// A string-keyed generic dictionary return is surfaced as a Go map.
	if looksLikeNSDictionary(objcType) {
		if keyObjC, valueObjC, kvOK := extractDictKV(
			objcType,
		); kvOK &&
			isNSStringType(normaliseObjC(keyObjC)) {
			valueGo, valueFromID, _, vimps := arrayElemConv(
				valueObjC,
				ctx,
				mapper,
				fc,
				rawPkgAlias,
				trialNames,
			)
			delete(vimps, "objref")
			maps.Copy(imports, vimps)
			imports["rt"] = rtImportPath
			imports["purego"] = pureobjcImportPath // purego.GoString in the key conversion
			keyConv := "func(_id objc.ID) string { return purego.GoString(_id) }"
			valueConv := "func(_id objc.ID) " + valueGo + " { return " + fmt.Sprintf(
				valueFromID,
				"_id",
			) + " }"
			wrap := "rt.DictToMap(%s, " + keyConv + ", " + valueConv + ")"
			return "map[string]" + valueGo, kindObject, wrap, "objc.ID", imports, true
		}
	}
	// A set return is surfaced as a Go slice (order unspecified).
	if looksLikeNSSet(objcType) {
		if elemObjC := extractNSArrayElem(objcType); elemObjC != "" {
			elemGo, fromID, _, eimps := arrayElemConv(
				elemObjC,
				ctx,
				mapper,
				fc,
				rawPkgAlias,
				trialNames,
			)
			delete(eimps, "objref")
			maps.Copy(imports, eimps)
			imports["rt"] = rtImportPath
			conv := "func(_id objc.ID) " + elemGo + " { return " + fmt.Sprintf(fromID, "_id") + " }"
			wrap := "rt.NSSetToSlice(%s, " + conv + ")"
			return "[]" + elemGo, kindArray, wrap, "objc.ID", imports, true
		}
	}
	if _, isBlock := mapper.ResolveBlockSignature(ret); isBlock {
		return "", kindVoid, "", "", imports, false
	}
	impSet := make(typemap.ImportSet)
	goRet := qualifyRaw(
		mapper.GoReturnType(objcType, ctx, impSet),
		fc,
		rawPkgAlias,
		ctx.GenericParams,
	)
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
	// A bare host pointer return that is neither a toll-free-bridged CF object nor
	// an os_object — a raw C struct pointer (IOKit's deviceDescriptor) or an inner
	// pointer (NS_RETURNS_INNER_POINTER, e.g. -[NSMutableData mutableBytes]) — is
	// surfaced unchanged as an opaque pointer for the caller to cast, mirroring the
	// unsafe.Pointer input passthrough in idiomaticArg.
	if goRet == "unsafe.Pointer" {
		imports["unsafe"] = "unsafe"
		return "unsafe.Pointer", kindScalar, "", "unsafe.Pointer", imports, true
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
	if ref, refImports, isCross := crossFrameworkWrapClass(goRet, fc, mapper); isCross {
		maps.Copy(imports, refImports)
		return "*" + ref.Package + "." + ref.TypeName, kindObject,
			ref.Package + "." + ref.TypeName + "FromID(%s)", "objc.ID", imports, true
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
	// A single pointer to a value struct — this framework's own (IOKit's
	// const IOUSBDeviceDescriptor *) or another's — is returned as *Struct so the
	// caller reads the fields directly instead of an opaque unsafe.Pointer. Gated
	// on the struct being an emittable, layout-safe value struct, so the pointer
	// never dangles or misreads a packed layout.
	if base, simps, ok := pointerValueStructType(goRet, fc, mapper, rawPkgAlias); ok {
		maps.Copy(imports, simps)
		return "*" + base, kindScalar, "", "*" + base, imports, true
	}
	if isHermeticScalar(goRet) {
		t := goSizeType(goRet)
		return t, kindScalar, "", t, imports, true
	}
	return "", kindVoid, "", "", imports, false
}

// pointerValueStructType recognises a single pointer to an emittable value struct
// — this framework's own ("*<rawAlias>.Foo") or another framework's ("*pkg.Foo")
// — and returns the Go type to name the pointee ("Foo" or "pkg.Foo") plus any
// import needed. ok is false for double pointers, slices, non-struct pointees, and
// structs that are not emittable value structs. The local case is gated on the
// authoritative mapper.EmittableStructs (which already excludes layout-unsafe
// packed structs), mirroring outParamGoType — fc.localStruct relies on a scanner
// field GoType the metadata omits and so under-reports newly captured structs.
func pointerValueStructType(
	goType string,
	fc *frameworkContext,
	mapper *typemap.Mapper,
	rawPkgAlias string,
) (string, map[string]string, bool) {
	if !strings.HasPrefix(goType, "*") || strings.HasPrefix(goType, "**") {
		return "", nil, false
	}
	pointee := goType[1:]
	if strings.HasPrefix(pointee, rawPkgAlias+".") {
		sname := pointee[len(rawPkgAlias)+1:]
		if sname != "" && !strings.ContainsAny(sname, ".*[]") &&
			fc.ownTypes[sname] && mapper.EmittableStructs[sname] {
			return sname, map[string]string{}, true
		}
		return "", nil, false
	}
	if name, simps, ok := crossFrameworkValueStruct(pointee, mapper, rawPkgAlias); ok {
		return name, simps, true
	}
	return "", nil, false
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

// crossFrameworkEmittedStruct is crossFrameworkValueStruct's counterpart for
// typedef aliases: it accepts any cross-framework struct the idiomatic layer
// physically emits (mapper.AllEmittedStructs), not just the all-clean-fields
// EmittableStructs subset. An alias only names the target type — it never reads
// its fields — so an opaque or degraded target is a perfectly valid alias RHS
// (e.g. AudioComponentInstance = *carboncore.ComponentInstanceRecord), whereas a
// struct FIELD referencing the same type would risk a non-hermetic field and so
// stays gated on the stricter EmittableStructs via crossFrameworkValueStruct.
func crossFrameworkEmittedStruct(
	goType string,
	mapper *typemap.Mapper,
	rawPkgAlias string,
) (string, map[string]string, bool) {
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
	if !mapper.AllEmittedStructs[name] {
		return "", nil, false
	}
	// The owner may be a C library (bindings/libraries/<pkg>) rather than a
	// framework (bindings/frameworks/<pkg>). The restored framework.Structs gather
	// gate means no library target reaches here today (matching raw), but selecting
	// the prefix by package kind guards against a future SDK bump silently emitting
	// a non-existent bindings/frameworks/<lib> import.
	prefix := idiomaticFrameworkPrefix
	if mapper.LibraryPkgs[pkg] {
		prefix = mapper.LibraryModulePrefix
	}
	return goType, map[string]string{pkg: prefix + pkg}, true
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
	// The conversion closures are typed func(...) objc.ID, so objc is always
	// referenced; every other import is contributed only by the branch whose
	// conversion actually names it (a blanket purego here would leave an unused
	// import in a file whose collections hold only rt-converted elements).
	imports = map[string]string{"objc": objcImportPath}
	normElem := normaliseObjC(elemObjC)
	switch {
	case isNSStringType(normElem):
		imports["purego"] = pureobjcImportPath
		return "string", "purego.GoString(%s)", "purego.NSString(%s)", imports
	case isNSURLType(normElem):
		imports["rt"] = rtImportPath
		return "string", "rt.URLString(%s)", "rt.FileURL(%s)", imports
	case isNSDateType(normElem):
		imports["rt"] = rtImportPath
		imports["time"] = "time"
		return "time.Time", "rt.NSDateToTime(%s)", "rt.TimeToNSDate(%s)", imports
	case isNSDataType(normElem):
		imports["rt"] = rtImportPath
		return "[]byte", "rt.NSDataToBytes(%s)", "rt.BytesToNSData(%s)", imports
	}
	impSet := make(typemap.ImportSet)
	goElem := qualifyRaw(mapper.GoType(elemObjC, ctx, impSet), fc, rawPkgAlias, ctx.GenericParams)
	if base, has := trialWrapClass(goElem, rawPkgAlias); has {
		if tt, named := trialNames[base]; named {
			imports["objref"] = objrefImportPath
			return "*" + tt, tt + "FromID(%s)", "objref.IDOf(%s)", imports
		}
	}
	if ref, refImports, isCross := crossFrameworkWrapClass(goElem, fc, mapper); isCross {
		maps.Copy(imports, refImports)
		imports["objref"] = objrefImportPath
		qualified := ref.Package + "." + ref.TypeName
		return "*" + qualified, qualified + "FromID(%s)", "objref.IDOf(%s)", imports
	}
	imports["obj"] = objImportPath
	imports["objref"] = objrefImportPath
	return "obj.Object", "obj.Wrap(%s)", "objref.IDOf(%s)", imports
}

// idiomaticCrossTargets are the idiomatic packages another package's returns
// may reference by typed wrapper. Restricting the targets to the foundational
// tier — packages that themselves never gain cross-package wrapper references
// — keeps the generated import graph trivially acyclic.
var idiomaticCrossTargets = map[string]bool{
	"foundation":     true,
	"corefoundation": true,
	"coregraphics":   true,
}

// crossFrameworkWrapClass recognises a resolved return (or collection element)
// type that is a single pointer to a class another idiomatic package wraps
// ("*foundation.NSProgress") and resolves it to that package's wrapper
// ({foundation, Progress}), so the caller emits *foundation.Progress via
// foundation.ProgressFromID instead of a generic obj.Object. Only the
// foundational allowlisted packages may be referenced (see
// idiomaticCrossTargets) and never from within one of them, which keeps the
// import graph acyclic by construction.
func crossFrameworkWrapClass(
	goType string,
	fc *frameworkContext,
	mapper *typemap.Mapper,
) (typemap.IdiomaticClassRef, map[string]string, bool) {
	currentPackage := strings.ToLower(fc.framework.Framework)
	if idiomaticCrossTargets[currentPackage] {
		return typemap.IdiomaticClassRef{}, nil, false
	}
	if !strings.HasPrefix(goType, "*") {
		return typemap.IdiomaticClassRef{}, nil, false
	}
	// base must be exactly pkg.Class — one dot, no generics or extra pointers.
	base := goType[1:]
	dot := strings.IndexByte(base, '.')
	if dot <= 0 || strings.ContainsAny(base, "[]* ") || strings.IndexByte(base[dot+1:], '.') >= 0 {
		return typemap.IdiomaticClassRef{}, nil, false
	}
	packageName, className := base[:dot], base[dot+1:]
	if !idiomaticCrossTargets[packageName] || packageName == currentPackage {
		return typemap.IdiomaticClassRef{}, nil, false
	}
	ref, known := mapper.IdiomaticClassIndex[className]
	if !known || ref.Package != packageName {
		return typemap.IdiomaticClassRef{}, nil, false
	}
	return ref, map[string]string{ref.Package: idiomaticFrameworkPrefix + ref.Package}, true
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

// isExactClassPointer reports whether objcType is a single pointer to exactly
// the named class: the identifier must end at a type boundary (so NSData does
// not match NSDataDetector) and a double pointer (an out-parameter, with or
// without nullability annotations between the asterisks) or a
// block/function-pointer type never matches.
func isExactClassPointer(objcType, className string) bool {
	if strings.Contains(objcType, "(^") || strings.Contains(objcType, "(*") {
		return false
	}
	t := normaliseObjC(objcType)
	rest, ok := strings.CutPrefix(t, className)
	if !ok {
		return false
	}
	if rest != "" && isIdentByte(rest[0], false) {
		return false
	}
	return pointerDepthOutsideGenerics(t) == 1
}

// pointerDepthOutsideGenerics counts the asterisks outside any <…> generic
// argument section, which is the type's own pointer depth: one for a value
// ("NSDictionary<NSString *, NSNumber *> *"), two for an out-parameter
// ("NSData * _Nullable *").
func pointerDepthOutsideGenerics(objcType string) int {
	depth, stars := 0, 0
	for i := 0; i < len(objcType); i++ {
		switch objcType[i] {
		case '<':
			depth++
		case '>':
			depth--
		case '*':
			if depth == 0 {
				stars++
			}
		}
	}
	return stars
}

// isNSDateType reports whether objcType is an NSDate pointer. Exact-boundary
// matching keeps NSDateComponents, NSDateFormatter, and NSDateInterval — whose
// values are not points in time — out of the time.Time conversion.
func isNSDateType(objcType string) bool {
	return isExactClassPointer(objcType, "NSDate")
}

// isNSDataType reports whether objcType is an NSData pointer. NSMutableData is
// deliberately not matched: converting it to []byte would hide its in-place
// mutation semantics, so it keeps its wrapper type.
func isNSDataType(objcType string) bool {
	return isExactClassPointer(objcType, "NSData")
}

// looksLikeNSDictionary reports whether objcType is an NSDictionary pointer
// (generic or not). NSMutableDictionary is deliberately not matched: a Go map
// copy would hide its in-place mutation semantics, so it keeps its wrapper
// type and the Set/Get augment.
func looksLikeNSDictionary(objcType string) bool {
	if strings.Contains(objcType, "(^") || strings.Contains(objcType, "(*") {
		return false
	}
	t := normaliseObjC(objcType)
	rest, ok := strings.CutPrefix(t, "NSDictionary")
	if !ok {
		return false
	}
	if rest != "" && isIdentByte(rest[0], false) {
		return false
	}
	return pointerDepthOutsideGenerics(t) == 1
}

// looksLikeNSSet reports whether objcType is an NSSet pointer (generic or
// not). NSMutableSet, NSCountedSet, and NSOrderedSet are deliberately not
// matched — the first two for mutation semantics, the last because its order
// is significant and a Go slice conversion via allObjects is already what
// NSArray handling provides.
func looksLikeNSSet(objcType string) bool {
	return isExactClassPointer(objcType, "NSSet")
}

// extractDictKV splits a generic dictionary's type arguments into the key and
// value Objective-C types ("NSDictionary<NSString *, NSNumber *> *"). ok is
// false for an ungenericized dictionary.
func extractDictKV(objcType string) (keyObjC, valueObjC string, ok bool) {
	_, after, found := strings.Cut(objcType, "<")
	if !found {
		return "", "", false
	}
	end := strings.LastIndex(after, ">")
	if end < 0 {
		return "", "", false
	}
	inner := after[:end]
	depth := 0
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				key := strings.TrimSpace(inner[:i])
				value := strings.TrimSpace(inner[i+1:])
				if key == "" || value == "" {
					return "", "", false
				}
				return key, value, true
			}
		}
	}
	return "", "", false
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
	return fc.localEnumTypeName(name), qualified, true
}
