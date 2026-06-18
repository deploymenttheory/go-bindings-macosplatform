package raw

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// ProtocolProxies writes a Go source file containing concrete id<Protocol> wrapper
// types for each ObjC protocol in the framework that appears in a return position.
//
// The name reflects the ObjC id<Protocol> concept: each type is named
// <GoProtoName>IDProtocol (e.g. VZVirtualMachineDelegateIDProtocol for
// id<VZVirtualMachineDelegate>). The type:
//   - Embeds foundation.NSObject, satisfying cgo.Object and NSObjectProtocol
//   - Exports a New<Name> constructor that registers a GC finalizer via cgo.Track
//   - Implements all non-variadic protocol methods via CGo bridge calls that
//     cast the receiver to id<Protocol> and dispatch dynamically
func EmitProtocolProxies(w io.Writer, pkgName, packageName string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, allClasses map[string]macosplatformmetadata.Class) error {
	// Only emit types for protocols that are in ProtocolProxyIndex (i.e. they
	// actually appear as id<P> return types somewhere in the SDK).
	names := make([]string, 0, len(framework.Protocols))
	for k := range framework.Protocols {
		if _, ok := m.ProtocolProxyIndex[k]; ok {
			names = append(names, k)
		}
	}
	if len(names) == 0 {
		return nil
	}
	sort.Strings(names)

	usedImports := make(typemap.ImportSet)
	ctx := m.BaseContext(framework.Framework, knownClasses)

	// Phase 1: build proxy type models, collecting imports as side effect.
	var proxyTypes []protocolProxyTypeModel
	needsBlocks := false
	for _, name := range names {
		p := framework.Protocols[name]
		model, hasBlks := buildProtocolIDTypeModel(name, p, framework, m, allClasses, ctx, usedImports)
		proxyTypes = append(proxyTypes, model)
		if hasBlks {
			needsBlocks = true
		}
	}

	// Phase 2: assemble the file model and render via template.
	allImports := []string{
		"unsafe",
		"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/cgo",
	}
	if pkgName != "foundation" {
		allImports = append(allImports, "github.com/deploymenttheory/go-bindings-macosplatform/frameworks/foundation")
	}
	if needsBlocks {
		allImports = append(allImports, "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/blocks")
	}
	for _, path := range usedImports {
		allImports = append(allImports, path)
	}

	sort.Strings(allImports)
	seen := map[string]bool{}
	var deduped []string
	for _, imp := range allImports {
		if !seen[imp] {
			seen[imp] = true
			deduped = append(deduped, imp)
		}
	}

	return executeTemplate(w, "protocol_proxies_file", protocolProxiesFileModel{
		PkgName:      pkgName,
		BridgeHeader: packageName + "_bridge.h",
		Imports:      deduped,
		ProxyTypes:   proxyTypes,
	})
}

// buildProtocolIDTypeModel constructs a protocolProxyTypeModel for one protocol.
// Returns the model plus flags indicating whether methods use context or blocks.
// Cross-framework imports discovered during method body building are recorded in imports.
func buildProtocolIDTypeModel(name string, p macosplatformmetadata.Protocol, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, allClasses map[string]macosplatformmetadata.Class, ctx typemap.Context, imports typemap.ImportSet) (model protocolProxyTypeModel, needsBlocks bool) {
	goName := naming.ProtocolGoTypeName(name, m.OwnerIndex)
	idTypeName := goName + "IDProtocol"

	isSelfFoundation := strings.EqualFold(framework.Framework, "Foundation")
	nsObjectEmbed := "foundation.NSObject"
	nsObjectWithPtr := "foundation.NSObjectWithPtr"
	if isSelfFoundation {
		nsObjectEmbed = "NSObject"
		nsObjectWithPtr = "NSObjectWithPtr"
	}

	bridgeNames := buildClassBridgeNames(framework.Framework, idTypeName, p.Methods)

	var methods []protocolProxyMethodModel
	seenMethods := map[string]bool{}
	for _, method := range p.Methods {
		if shouldSkipBridgeMethod(method) {
			continue
		}
		if method.Availability.IsUnavailable {
			continue
		}
		if method.IsClassMethod {
			continue
		}
		if methodRefsUnavailableClass(method, framework, allClasses) {
			continue
		}
		goMethodName := naming.MethodName(method.Selector)
		if seenMethods[goMethodName] {
			continue
		}
		seenMethods[goMethodName] = true

		cFunc := bridgeNames[methodKey(method.Selector, false)]
		if cFunc == "" {
			continue
		}

		idCtx := ctx
		idCtx.ClassName = idTypeName

		args := buildGoArgs(method.Params, method.IsNSError, idCtx, m, imports)
		ret := buildGoReturn(method, idCtx, m, idTypeName, imports)

		// Pre-render the method body by writing to a strings.Builder.
		var bodyBuf strings.Builder
		writeIDProtocolMethodBody(&bodyBuf, method, cFunc, framework.Framework, idTypeName, idCtx, m, framework.Classes, imports)
		bodyLines := bodyBuf.String()

		if strings.Contains(bodyLines, "blocks.MakeBlock_") {
			needsBlocks = true
		}

		methods = append(methods, protocolProxyMethodModel{
			GoName:       goMethodName,
			Params:       strings.Join(args, ", "),
			Ret:          ret,
			AvailComment: availabilityComment(method.Availability),
			BodyLines:    bodyLines,
		})
	}

	model = protocolProxyTypeModel{
		IDTypeName:      idTypeName,
		ProtoName:       name,
		NSObjectEmbed:   nsObjectEmbed,
		NSObjectWithPtr: nsObjectWithPtr,
		AvailComment:    availabilityComment(p.Availability),
		Methods:         methods,
	}
	return model, needsBlocks
}

// writeIDProtocolMethodBody writes the CGo bridge call body for an id<Protocol>
// wrapper method. Mirrors writeMethodBody in classes.go but uses "p" as the
// receiver variable (for *<Proto>IDProtocol receivers).
func writeIDProtocolMethodBody(w io.Writer, method macosplatformmetadata.Method, cFunc string, framework, idTypeName string, ctx typemap.Context, m *typemap.Mapper, fmClasses map[string]macosplatformmetadata.Class, imports typemap.ImportSet) {
	var preambles []string
	var keepAlives []string
	cgoCallArgs := buildIDProtocolCGOCallArgs(method.Params, method.IsNSError, ctx, m, &preambles, &keepAlives, imports)

	// Keep the receiver and every ObjC-object argument alive across the CGo call.
	fmt.Fprintf(w, "\tdefer cgo.KeepAlive(p)\n")
	for _, ka := range keepAlives {
		fmt.Fprintf(w, "\tdefer cgo.KeepAlive(%s)\n", ka)
	}

	for _, preamble := range preambles {
		fmt.Fprintf(w, "\t%s\n", preamble)
	}

	fmt.Fprintf(w, "\tvar _exc unsafe.Pointer\n")
	if method.IsNSError {
		fmt.Fprintf(w, "\tvar _nsErr unsafe.Pointer\n")
	}

	retType := primaryReturnType(method, ctx, m, imports)
	isNullableStr := retType == "string" && method.Return.IsNullable
	isValueStruct := isValueStructReturn(retType, m)

	rawCall := fmt.Sprintf("C.%s(%s)", cFunc, cgoCallArgs)
	if retType == "cgo.Object" || isObjectReturn(retType) || isNullableStr || isValueStruct {
		fmt.Fprintf(w, "\t_ptr := unsafe.Pointer(%s)\n", rawCall)
	} else if retType != "" {
		fmt.Fprintf(w, "\t_result := %s\n", cgoReturnConvert(rawCall, retType, m))
	} else {
		fmt.Fprintf(w, "\t%s\n", rawCall)
	}

	fmt.Fprintf(w, "\tcgo.RaiseIfException(_exc)\n")

	if method.IsNSError {
		if retType == "" {
			fmt.Fprintf(w, "\tif _nsErr != nil {\n\t\treturn cgo.NSErrorToError(_nsErr)\n\t}\n")
			fmt.Fprintf(w, "\treturn nil\n")
		} else if retType == "cgo.Object" {
			fmt.Fprintf(w, "\tif _nsErr != nil {\n\t\treturn nil, cgo.NSErrorToError(_nsErr)\n\t}\n")
			fmt.Fprintf(w, "\treturn cgo.WrapObject(_ptr), nil\n")
		} else if isObjectReturn(retType) {
			structType := extractStructType(retType, false)
			fmt.Fprintf(w, "\tif _nsErr != nil {\n\t\treturn nil, cgo.NSErrorToError(_nsErr)\n\t}\n")
			writeObjectReturnWithError(w, structType, fmClasses, m)
		} else if isValueStruct {
			fmt.Fprintf(w, "\tif _nsErr != nil {\n")
			fmt.Fprintf(w, "\t\tif _ptr != nil {\n\t\t\tcgo.FreePtr(_ptr)\n\t\t}\n")
			fmt.Fprintf(w, "\t\treturn %s{}, cgo.NSErrorToError(_nsErr)\n\t}\n", retType)
			fmt.Fprintf(w, "\tif _ptr == nil {\n\t\treturn %s{}, nil\n\t}\n", retType)
			fmt.Fprintf(w, "\t_result := *(*%s)(unsafe.Pointer(_ptr))\n", retType)
			fmt.Fprintf(w, "\tcgo.FreePtr(_ptr)\n")
			fmt.Fprintf(w, "\treturn _result, nil\n")
		} else {
			fmt.Fprintf(w, "\tif _nsErr != nil {\n\t\treturn _result, cgo.NSErrorToError(_nsErr)\n\t}\n")
			fmt.Fprintf(w, "\treturn _result, nil\n")
		}
		return
	}

	if retType == "" {
		return
	}
	if retType == "cgo.Object" {
		fmt.Fprintf(w, "\treturn cgo.WrapObject(_ptr)\n")
		return
	}
	if isObjectReturn(retType) {
		structType := extractStructType(retType, false)
		writeObjectReturn(w, structType, fmClasses, m)
		return
	}
	if isValueStruct {
		fmt.Fprintf(w, "\tif _ptr == nil {\n\t\treturn %s{}\n\t}\n", retType)
		fmt.Fprintf(w, "\t_result := *(*%s)(unsafe.Pointer(_ptr))\n", retType)
		fmt.Fprintf(w, "\tcgo.FreePtr(_ptr)\n")
		fmt.Fprintf(w, "\treturn _result\n")
		return
	}
	if isNullableStr {
		fmt.Fprintf(w, "\tif _ptr == nil {\n\t\treturn nil\n\t}\n")
		fmt.Fprintf(w, "\t_s := C.GoString((*C.char)(_ptr))\n")
		fmt.Fprintf(w, "\treturn &_s\n")
		return
	}
	fmt.Fprintf(w, "\treturn _result\n")
}

// buildIDProtocolCGOCallArgs builds the CGo call argument list for an id<Protocol>
// wrapper method. Prepends "p.Ptr()" as the receiver (self) argument.
func buildIDProtocolCGOCallArgs(args []macosplatformmetadata.Param, hasNSError bool, ctx typemap.Context, m *typemap.Mapper, preambles, keepAlives *[]string, imports typemap.ImportSet) string {
	var parts []string
	parts = append(parts, "p.Ptr()")

	resolved := buildParamNames(args)
	for i, arg := range args {
		blockObjCType := ""
		if arg.IsBlock {
			blockObjCType = arg.ObjCType
		} else if target, ok := m.TypedefIndex[typemap.Normalise(arg.ObjCType)]; ok && typemap.IsBlock(target) {
			blockObjCType = target
		}
		if blockObjCType != "" {
			sigName := BlockSigName(blockObjCType, m)
			if sigName != "" {
				blkVar := "_blk_" + resolved[i]
				argExpr := resolved[i]
				if blockNeedsWrapper(blockObjCType, ctx, m) {
					argExpr = buildBlockWrapper(blockObjCType, resolved[i], ctx, m, imports)
				}
				*preambles = append(*preambles,
					blkVar+" := blocks.MakeBlock_"+sigName+"("+argExpr+")",
					"defer blocks.FreeBlock("+blkVar+")",
				)
				parts = append(parts, blkVar)
			} else {
				parts = append(parts, "unsafe.Pointer(nil)")
			}
			continue
		}
		goType := m.GoType(arg.ObjCType, ctx, imports)
		if goType == "" {
			goType = "unsafe.Pointer"
		}
		parts = append(parts, goCGoArgExpr(goType, resolved[i], m, preambles, keepAlives))
	}

	if hasNSError {
		parts = append(parts, "&_nsErr")
	}
	parts = append(parts, "&_exc")

	return strings.Join(parts, ", ")
}
