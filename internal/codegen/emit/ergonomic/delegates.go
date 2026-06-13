package ergonomic

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/classify"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// ErgonomicDelegates emits Go delegate interfaces and the Set<Class>Delegate bridge
// functions for every class that has a delegate/dataSource property backed by a
// DelegateProtocol-tagged protocol.
//
// Output per framework:
//   - <fw>_delegates_generated.go  — Go interfaces + SetXxxDelegate functions
func EmitDelegates(w io.Writer, pkgName, rawImportPath string, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, nt *NameTracker) error {
	type delegateEntry struct {
		className    string
		propName     string // "delegate" or "dataSource"
		protoName    string // ObjC protocol name
		goIfaceName  string // Go interface name (protoName)
		rawProto     meta.Protocol
	}

	var entries []delegateEntry
	seen := make(map[string]bool)

	for _, className := range sortedKeys(framework.Classes) {
		cls := framework.Classes[className]
		if cls.Availability.IsUnavailable {
			continue
		}

		for _, dp := range classify.DelegateProperties(cls) {
			proto, ok := framework.Protocols[dp.ProtocolName]
			if !ok {
				continue
			}
			if !classify.IsDelegateProtocol(proto) {
				continue
			}
			key := dp.ProtocolName
			if seen[key] {
				// Protocol interface already emitted; still need Set function for this class.
			}
			entries = append(entries, delegateEntry{
				className:   className,
				propName:    dp.PropertyName,
				protoName:   dp.ProtocolName,
				goIfaceName: dp.ProtocolName,
				rawProto:    proto,
			})
			seen[key] = true
		}
	}

	if len(entries) == 0 {
		return nil
	}

	usedImports := make(map[string]string)
	usedImports["unsafe"] = "unsafe"

	ctx := m.BaseContext(framework.Framework, knownClasses)
	localImports := make(typemap.ImportSet)

	// Collect cross-framework imports from protocol method signatures.
	protoSeen := make(map[string]bool)
	for _, e := range entries {
		if protoSeen[e.protoName] {
			continue
		}
		protoSeen[e.protoName] = true
		for _, pm := range e.rawProto.Methods {
			for _, arg := range pm.Params {
				goType := resolveOpinionatedArgType(arg.ObjCType, ctx, m, framework, localImports)
				recordOpinionatedImports(goType, m, usedImports)
			}
			if pm.Return.ObjCType != "" && pm.Return.ObjCType != "void" {
				goType := resolveOpinionatedArgType(pm.Return.ObjCType, ctx, m, framework, localImports)
				recordOpinionatedImports(goType, m, usedImports)
			}
		}
	}

	var body bytes.Buffer

	// Emit one Go interface per unique protocol.
	protoSeen = make(map[string]bool)
	for _, e := range entries {
		if protoSeen[e.protoName] {
			continue
		}
		protoSeen[e.protoName] = true

		proto := e.rawProto
		var requiredMethods []meta.Method
		var optionalMethods []meta.Method
		for _, pm := range proto.Methods {
			if pm.IsOptional {
				optionalMethods = append(optionalMethods, pm)
			} else {
				requiredMethods = append(requiredMethods, pm)
			}
		}

		// Go interface — required methods only.
		fmt.Fprintf(&body, "// %s is the Go interface for the ObjC %s protocol.\n", e.goIfaceName, e.protoName)
		fmt.Fprintf(&body, "type %s interface {\n", e.goIfaceName)
		requiredMethodSeen := make(map[string]bool)
		for _, pm := range requiredMethods {
			if pm.Availability.IsUnavailable {
				continue
			}
			goMethodName := naming.MethodName(pm.Selector)
			if requiredMethodSeen[goMethodName] {
				continue
			}
			requiredMethodSeen[goMethodName] = true
			sig := buildDelegateMethodGoSig(pm, ctx, m, framework, localImports)
			fmt.Fprintf(&body, "\t%s\n", sig)
		}
		fmt.Fprintf(&body, "}\n\n")

		// Default<Protocol> struct — no-ops for all optional methods.
		if len(optionalMethods) > 0 {
			fmt.Fprintf(&body, "// Default%s provides no-op implementations of all optional %s methods.\n",
				e.protoName, e.protoName)
			fmt.Fprintf(&body, "// Embed it in your implementation struct to avoid implementing unused optional methods.\n")
			fmt.Fprintf(&body, "type Default%s struct{}\n\n", e.protoName)
			optionalMethodSeen := make(map[string]bool)
			for _, pm := range optionalMethods {
				if pm.Availability.IsUnavailable {
					continue
				}
				goMethodName := naming.MethodName(pm.Selector)
				if optionalMethodSeen[goMethodName] {
					continue
				}
				optionalMethodSeen[goMethodName] = true
				sig := buildDelegateMethodGoSig(pm, ctx, m, framework, localImports)
				retParts := extractDelegateReturnParts(pm, ctx, m, framework, localImports)
				fmt.Fprintf(&body, "func (*Default%s) %s {", e.protoName, sig)
				if len(retParts) > 0 {
					// Use var declarations for zero values — safe for any type including
					// structs (where nil would be invalid). Each statement ends with
					// a semicolon so multiple return values don't run together.
					for i, r := range retParts {
						fmt.Fprintf(&body, " var _r%d %s; _ = _r%d;", i, r, i)
					}
					retNames := make([]string, len(retParts))
					for i := range retParts {
						retNames[i] = fmt.Sprintf("_r%d", i)
					}
					fmt.Fprintf(&body, "; return %s ", strings.Join(retNames, ", "))
				}
				fmt.Fprintf(&body, "}\n")
			}
			fmt.Fprintf(&body, "\n")
		}
	}

	// Emit Set<ClassName>Delegate functions.
	for _, e := range entries {
		fnName := "Set" + e.className + strings.ToUpper(e.propName[:1]) + e.propName[1:]
		if !nt.Claim(fnName, "delegates") {
			continue
		}

		setterSel := "set" + strings.ToUpper(e.propName[:1]) + e.propName[1:] + ":"
		setterMethod := naming.MethodName(setterSel)

		fmt.Fprintf(&body, "// %s installs impl as the %s of o.\n", fnName, e.propName)
		fmt.Fprintf(&body, "// All required protocol methods are wired unconditionally; optional methods\n")
		fmt.Fprintf(&body, "// are wired when impl also satisfies the corresponding one-method interface.\n")
		fmt.Fprintf(&body, "func %s(ctx context.Context, o *%s, impl %s) {\n",
			fnName, buildRawReceiverType(e.className, m), e.goIfaceName)
		fmt.Fprintf(&body, "\trawImpl := &raw.%sImpl{\n", e.protoName)
		writeDelegateAdapterFields(&body, e.rawProto, ctx, m, framework, localImports)
		fmt.Fprintf(&body, "\t}\n")
		fmt.Fprintf(&body, "\tshim := raw.New%sImpl(ctx, rawImpl)\n", e.protoName)
		fmt.Fprintf(&body, "\to.%s(ctx, shim)\n", setterMethod)
		fmt.Fprintf(&body, "}\n\n")
	}

	if body.Len() == 0 {
		return nil
	}

	recordSpecialImports(body.Bytes(), usedImports)
	writeErgonomicHeader(w, pkgName, rawImportPath, usedImports, false)
	_, err := w.Write(body.Bytes())
	return err
}

// writeDelegateAdapterFields writes one field initialiser per IMP-safe protocol method
// into the raw*Impl struct literal. Required methods are wired unconditionally;
// optional methods are guarded by a one-method interface type assertion so they are
// only wired when impl provides a real implementation.
func writeDelegateAdapterFields(body *bytes.Buffer, proto meta.Protocol, ctx typemap.Context, m *typemap.Mapper, framework *meta.FrameworkMeta, imports typemap.ImportSet) {
	seen := make(map[string]bool)
	for _, pm := range proto.Methods {
		if pm.Availability.IsUnavailable {
			continue
		}
		goName := naming.MethodName(pm.Selector)
		if seen[goName] {
			continue
		}
		// Only handle methods whose args/return are all IMP-safe (pointer or primitive).
		if !delegateMethodIsAdaptable(pm, m) {
			continue
		}
		seen[goName] = true

		// Build raw func signature: (self unsafe.Pointer, arg0T, arg1T, ...) retT
		rawArgTypes, rawRetType := buildDelegateRawFuncTypes(pm, ctx, m)

		// Ergonomic arg names and constructors.
		var rawParams []string // raw closure parameter declarations
		rawParams = append(rawParams, "_self unsafe.Pointer")
		var callArgs []string // expressions passed to impl.Method(ctx, ...)
		callArgs = append(callArgs, "ctx")
		for i, arg := range pm.Params {
			rawT := rawArgTypes[i]
			argVar := fmt.Sprintf("_a%d", i)
			rawParams = append(rawParams, argVar+" "+rawT)
			ergoT := resolveOpinionatedArgType(arg.ObjCType, ctx, m, framework, imports)
			callArgs = append(callArgs, convertDelegateArg(argVar, rawT, ergoT, m))
		}

		// Build return conversion.
		ergoRetParts := extractDelegateReturnParts(pm, ctx, m, framework, imports)
		ergoRetType := ""
		if len(ergoRetParts) == 1 {
			ergoRetType = ergoRetParts[0]
		}

		rawFuncSig := "func(" + strings.Join(rawParams, ", ") + ")"
		if rawRetType != "" {
			rawFuncSig += " " + rawRetType
		}

		callExpr := fmt.Sprintf("impl.%s(%s)", goName, strings.Join(callArgs, ", "))

		var body2 strings.Builder
		if rawRetType == "" {
			body2.WriteString(callExpr)
		} else {
			body2.WriteString(convertDelegateReturn(callExpr, ergoRetType, rawRetType))
		}

		if pm.IsOptional {
			// Wrap in type assertion so the field is only set when impl provides the method.
			// We generate an anonymous interface for the specific method.
			ergoSig := buildDelegateMethodGoSig(pm, ctx, m, framework, imports)
			fmt.Fprintf(body, "\t\t// optional — wired only if impl implements %s\n", goName)
			fmt.Fprintf(body, "\t\t%s: func() %s {\n", goName, rawFuncSig)
			fmt.Fprintf(body, "\t\t\ttype opt interface{ %s }\n", ergoSig)
			fmt.Fprintf(body, "\t\t\tif h, ok := impl.(opt); ok {\n")
			fmt.Fprintf(body, "\t\t\t\treturn %s {\n", rawFuncSig)
			fmt.Fprintf(body, "\t\t\t\t\t_ = _self\n")
			// Rewrite callArgs to use h. instead of impl.
			callArgsH := make([]string, len(callArgs))
			copy(callArgsH, callArgs)
			callArgsH[0] = "ctx"
			callExprH := fmt.Sprintf("h.%s(%s)", goName, strings.Join(callArgsH, ", "))
			if rawRetType == "" {
				fmt.Fprintf(body, "\t\t\t\t\t%s\n", callExprH)
			} else {
				fmt.Fprintf(body, "\t\t\t\t\treturn %s\n", convertDelegateReturn(callExprH, ergoRetType, rawRetType))
			}
			fmt.Fprintf(body, "\t\t\t\t}\n")
			fmt.Fprintf(body, "\t\t\t}\n")
			fmt.Fprintf(body, "\t\t\treturn nil\n")
			fmt.Fprintf(body, "\t\t}(),\n")
		} else {
			fmt.Fprintf(body, "\t\t%s: %s {\n", goName, rawFuncSig)
			fmt.Fprintf(body, "\t\t\t_ = _self\n")
			if rawRetType == "" {
				fmt.Fprintf(body, "\t\t\t%s\n", body2.String())
			} else {
				fmt.Fprintf(body, "\t\t\treturn %s\n", body2.String())
			}
			fmt.Fprintf(body, "\t\t},\n")
		}
	}
}

// delegateMethodIsAdaptable reports whether all of a method's args and return
// can be handled by the adapter (pointer or scalar types only; no structs by value).
func delegateMethodIsAdaptable(pm meta.Method, m *typemap.Mapper) bool {
	ctx := typemap.Context{}
	check := func(objcType string) bool {
		ct := m.CType(objcType, ctx, nil)
		return ct == "void *" || ct == "void" || ct == "" ||
			ct == "bool" ||
			ct == "int8_t" || ct == "int16_t" || ct == "int32_t" || ct == "int64_t" ||
			ct == "uint8_t" || ct == "uint16_t" || ct == "uint32_t" || ct == "uint64_t" ||
			ct == "float" || ct == "double"
	}
	for _, arg := range pm.Params {
		if !check(arg.ObjCType) {
			return false
		}
	}
	return check(pm.Return.ObjCType)
}

// buildDelegateRawFuncTypes returns the raw Go types for each arg (excluding self)
// and the raw return Go type ("" for void).
func buildDelegateRawFuncTypes(pm meta.Method, ctx typemap.Context, m *typemap.Mapper) (argTypes []string, retType string) {
	rawType := func(objcType string) string {
		ct := m.CType(objcType, ctx, nil)
		switch ct {
		case "bool":
			return "bool"
		case "int8_t":
			return "int8"
		case "int16_t":
			return "int16"
		case "int32_t":
			return "int32"
		case "int64_t":
			return "int64"
		case "uint8_t":
			return "uint8"
		case "uint16_t":
			return "uint16"
		case "uint32_t":
			return "uint32"
		case "uint64_t":
			return "uint64"
		case "float":
			return "float32"
		case "double":
			return "float64"
		default:
			return "unsafe.Pointer"
		}
	}
	for _, arg := range pm.Params {
		argTypes = append(argTypes, rawType(arg.ObjCType))
	}
	if !typemap.IsVoid(pm.Return.ObjCType) && pm.Return.ObjCType != "" {
		retType = rawType(pm.Return.ObjCType)
	}
	return
}

// convertDelegateArg generates an expression that converts a raw closure arg
// (argVar of rawT) to the ergonomic type ergoT.
func convertDelegateArg(argVar, rawT, ergoT string, m *typemap.Mapper) string {
	if rawT != "unsafe.Pointer" {
		// Primitive — may need enum cast.
		if ergoT == rawT {
			return argVar
		}
		// Enum or differently-typed primitive: cast.
		return ergoT + "(" + argVar + ")"
	}
	// ObjC pointer → ergonomic typed wrapper.
	if ergoT == "unsafe.Pointer" || ergoT == "" {
		return argVar
	}
	if !strings.HasPrefix(ergoT, "*") {
		return argVar
	}
	typeName := strings.TrimPrefix(ergoT, "*")
	// Strip generic type parameters from the class name before constructing the
	// New<ClassName> call — NewNSArray is not generic, but *NSArray[T] is the type.
	baseTypeName := typeName
	if bracket := strings.Index(typeName, "["); bracket > 0 {
		baseTypeName = typeName[:bracket]
	}
	if strings.Contains(baseTypeName, ".") {
		// Cross-framework: e.g. "foundation.NSNotification" → "foundation.NewNSNotification(argVar)"
		dot := strings.LastIndex(baseTypeName, ".")
		pkg := baseTypeName[:dot]
		cls := baseTypeName[dot+1:]
		return pkg + ".New" + cls + "(" + argVar + ")"
	}
	// Same-framework (raw. prefix): e.g. "raw.NSFoo" → "raw.NewNSFoo(argVar)"
	if strings.HasPrefix(baseTypeName, "raw.") {
		cls := strings.TrimPrefix(baseTypeName, "raw.")
		return "raw.New" + cls + "(" + argVar + ")"
	}
	// Fallback.
	return argVar
}

// convertDelegateReturn generates an expression that converts the ergonomic return
// value (callExpr) to the raw return type rawT.
func convertDelegateReturn(callExpr, ergoT, rawT string) string {
	if rawT == "unsafe.Pointer" {
		// Ergonomic returns an ObjC pointer wrapper.
		if strings.HasPrefix(ergoT, "*") {
			return "_r := " + callExpr + "; if _r == nil { return nil }; return _r.Ptr()"
		}
		return callExpr
	}
	if ergoT == rawT {
		return callExpr
	}
	// Different primitive (e.g. enum → uint64): cast.
	return rawT + "(" + callExpr + ")"
}

// buildDelegateMethodGoSig returns the Go method signature string for a protocol method.
func buildDelegateMethodGoSig(pm meta.Method, ctx typemap.Context, m *typemap.Mapper, framework *meta.FrameworkMeta, imports typemap.ImportSet) string {
	methodName := naming.MethodName(pm.Selector)

	var params []string
	params = append(params, "ctx context.Context")
	for i, arg := range pm.Params {
		goType := resolveOpinionatedArgType(arg.ObjCType, ctx, m, framework, imports)
		if goType == "" {
			goType = "unsafe.Pointer"
		}
		argName := naming.ParamName(arg.Name)
		if argName == "" {
			argName = fmt.Sprintf("arg%d", i)
		}
		params = append(params, argName+" "+goType)
	}

	retParts := extractDelegateReturnParts(pm, ctx, m, framework, imports)
	var retSig string
	switch len(retParts) {
	case 0:
		retSig = ""
	case 1:
		retSig = " " + retParts[0]
	default:
		retSig = " (" + strings.Join(retParts, ", ") + ")"
	}

	return fmt.Sprintf("%s(%s)%s", methodName, strings.Join(params, ", "), retSig)
}

// extractDelegateReturnParts returns the Go return types for a protocol method.
func extractDelegateReturnParts(pm meta.Method, ctx typemap.Context, m *typemap.Mapper, framework *meta.FrameworkMeta, imports typemap.ImportSet) []string {
	var parts []string
	retObjC := strings.TrimSpace(pm.Return.ObjCType)
	if retObjC != "" && retObjC != "void" {
		rt := resolveOpinionatedArgType(retObjC, ctx, m, framework, imports)
		if rt != "" && rt != "unsafe.Pointer" {
			parts = append(parts, rt)
		}
	}
	if pm.IsNSError {
		parts = append(parts, "error")
	}
	return parts
}
