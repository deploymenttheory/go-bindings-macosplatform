package raw

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// MethodSigModel describes a unique primitive-typed IMP method signature.
// Two ObjC methods that differ only in ObjC object types share the same
// MethodSigModel because they use the same C/Go representation.
//
// Unlike block signatures, self (id) is always implicit — it is NOT reflected
// in the Name tokens. The Go callback ALWAYS receives self as its first arg.
type MethodSigModel struct {
	Name          string // "void_ptr_ptr" — retToken[_argToken...]  (no self token)
	IsVoidRet     bool
	CReturnType   string // C return type: "void", "int64_t", "bool"
	CGoReturnType string // CGo type: "C.int64_t", "C.bool", "" for void
	GoReturnType  string // Go return type: "int64", "bool", "" for void
	Params        []MethodSigArg
	ObjCEnc       string // ObjC type encoding for class_addMethod: "v@:@"
}

// MethodSigArg describes one explicit argument (not self or _cmd).
type MethodSigArg struct {
	CType   string // C type in the trampoline: "void *", "int64_t", "bool"
	CGOType string // CGo type in the //export function
	GoType  string // Go type in the callback func
	Enc     string // ObjC type encoding character(s)
}

// methodSigTypeInfo maps token → (C, CGo, Go, ObjC-enc).
type methodSigTypeInfo struct{ cType, cgoType, goType, enc string }

var methodTokenTypeMap = map[string]methodSigTypeInfo{
	"ptr":     {"void *", "unsafe.Pointer", "unsafe.Pointer", "@"},
	"bool":    {"bool", "C.bool", "bool", "B"},
	"int8":    {"int8_t", "C.int8_t", "int8", "c"},
	"int16":   {"int16_t", "C.int16_t", "int16", "s"},
	"int32":   {"int32_t", "C.int32_t", "int32", "i"},
	"int64":   {"int64_t", "C.int64_t", "int64", "q"},
	"uint8":   {"uint8_t", "C.uint8_t", "uint8", "C"},
	"uint16":  {"uint16_t", "C.uint16_t", "uint16", "S"},
	"uint32":  {"uint32_t", "C.uint32_t", "uint32", "I"},
	"uint64":  {"uint64_t", "C.uint64_t", "uint64", "Q"},
	"float32": {"float", "C.float", "float32", "f"},
	"float64": {"double", "C.double", "float64", "d"},
}

// CollectMethodSigsFromFrameworks scans all class and protocol methods in frameworks
// and returns the deduplicated, sorted set of MethodSigs that need IMP trampolines.
// Methods with struct-by-value arguments/returns are excluded (not IMP-safe).
func CollectMethodSigsFromFrameworks(frameworks []*macosplatformmetadata.FrameworkMeta, m *typemap.Mapper) []MethodSigModel {
	seen := make(map[string]MethodSigModel)

	add := func(method macosplatformmetadata.Method) {
		if !isMethodIMPSafe(method, m) {
			return
		}
		sig, ok := methodSigFromMethod(method, m)
		if !ok {
			return
		}
		if _, exists := seen[sig.Name]; !exists {
			seen[sig.Name] = sig
		}
	}

	for _, framework := range frameworks {
		for _, cls := range framework.Classes {
			for _, method := range cls.Methods {
				add(method)
			}
		}
		for _, proto := range framework.Protocols {
			for _, method := range proto.Methods {
				add(method)
			}
		}
	}

	sigs := make([]MethodSigModel, 0, len(seen))
	for _, sig := range seen {
		sigs = append(sigs, sig)
	}
	sort.Slice(sigs, func(i, j int) bool { return sigs[i].Name < sigs[j].Name })
	return sigs
}

// isMethodIMPSafe returns true when all argument and return types can be
// represented as primitive tokens in the IMP trampoline. Methods with
// struct-by-value types, variadic args, error out-params, or init prefixes
// are excluded.
func isMethodIMPSafe(method macosplatformmetadata.Method, m *typemap.Mapper) bool {
	if method.IsClassMethod || method.IsInit || method.IsNSError || method.IsVariadic {
		return false
	}
	if strings.HasPrefix(method.Selector, "init") {
		return false
	}

	ctx := typemap.Context{}

	// Check return type.
	if !typemap.IsVoid(method.Return.ObjCType) && method.Return.ObjCType != "" {
		cType := m.CType(method.Return.ObjCType, ctx, nil)
		if !isKnownCPrimitive(cType) {
			return false
		}
	}

	// Check each argument type.
	for _, arg := range method.Params {
		cType := m.CType(arg.ObjCType, ctx, nil)
		if !isKnownCPrimitive(cType) {
			return false
		}
	}
	return true
}

// isKnownCPrimitive returns true when cType is a void pointer, scalar, or void.
// Named struct types (e.g. "CGRect") are NOT in this set.
func isKnownCPrimitive(cType string) bool {
	switch cType {
	case "", "void", "void *",
		"bool",
		"int8_t", "int16_t", "int32_t", "int64_t",
		"uint8_t", "uint16_t", "uint32_t", "uint64_t",
		"float", "double",
		"const char *":
		return true
	}
	// Named integer types from enum resolution
	if strings.HasSuffix(cType, "_t") {
		return true
	}
	return false
}

// methodSigFromMethod builds a MethodSigModel for an IMP-safe method.
func methodSigFromMethod(method macosplatformmetadata.Method, m *typemap.Mapper) (MethodSigModel, bool) {
	ctx := typemap.Context{}

	// Determine return token.
	var retTok, CReturnType, CGoReturnType, GoReturnType, retEnc string
	isVoid := typemap.IsVoid(method.Return.ObjCType) || method.Return.ObjCType == ""
	if isVoid {
		retTok = "void"
		CReturnType = "void"
		retEnc = "v"
	} else {
		cType := m.CType(method.Return.ObjCType, ctx, nil)
		retTok = cTypeToken(cType)
		if retTok == "void" || retTok == "" {
			retTok = "ptr"
		}
		if info, ok := methodTokenTypeMap[retTok]; ok {
			CReturnType = info.cType
			CGoReturnType = info.cgoType
			GoReturnType = info.goType
			retEnc = info.enc
		} else {
			CReturnType = "void *"
			CGoReturnType = "unsafe.Pointer"
			GoReturnType = "unsafe.Pointer"
			retEnc = "@"
			retTok = "ptr"
		}
	}

	// Build argument tokens.
	var args []MethodSigArg
	var argToks []string
	argEncs := ""
	for _, arg := range method.Params {
		cType := m.CType(arg.ObjCType, ctx, nil)
		tok := cTypeToken(cType)
		if tok == "" || tok == "void" {
			continue
		}
		info, ok := methodTokenTypeMap[tok]
		if !ok {
			info = methodTokenTypeMap["ptr"]
			tok = "ptr"
		}
		args = append(args, MethodSigArg{
			CType:   info.cType,
			CGOType: info.cgoType,
			GoType:  info.goType,
			Enc:     info.enc,
		})
		argToks = append(argToks, tok)
		argEncs += info.enc
	}

	// Build ObjC encoding: retEnc + "@:" + argEncs (@ = self, : = _cmd)
	enc := retEnc + "@:" + argEncs

	// Build canonical name.
	nameParts := []string{retTok}
	nameParts = append(nameParts, argToks...)
	name := strings.Join(nameParts, "_")

	return MethodSigModel{
		Name:          name,
		IsVoidRet:     isVoid,
		CReturnType:   CReturnType,
		CGoReturnType: CGoReturnType,
		GoReturnType:  GoReturnType,
		Params:        args,
		ObjCEnc:       enc,
	}, true
}

// goCallbackFuncType returns the Go func type for the callback stored in the
// callbacks registry for a given MethodSigModel.
// Self (unsafe.Pointer) is always the first parameter.
func (sig MethodSigModel) goCallbackFuncType() string {
	var params []string
	params = append(params, "unsafe.Pointer") // self
	for _, a := range sig.Params {
		params = append(params, a.GoType)
	}
	paramList := strings.Join(params, ", ")
	if sig.IsVoidRet {
		return fmt.Sprintf("func(%s)", paramList)
	}
	return fmt.Sprintf("func(%s) %s", paramList, sig.GoReturnType)
}

// EmitRuntimeCallbacksGo writes callbacks_generated.go for the bindings/runtime/callbacks package.
func EmitRuntimeCallbacksGo(w io.Writer, sigs []MethodSigModel, pkg string) error {
	return executeTemplate(w, "method_trampolines_go_file", methodTrampolinesGoFileModel{
		PkgName: pkg,
		Sigs:    buildMethodTrampolineSigModels(sigs),
	})
}

// EmitRuntimeCallbacksTrampolineHeader writes method_trampolines_generated.h.
func EmitRuntimeCallbacksTrampolineHeader(w io.Writer, sigs []MethodSigModel) error {
	return executeTemplate(w, "method_trampolines_h_file", methodTrampolinesHFileModel{
		Sigs: buildMethodTrampolineSigModels(sigs),
	})
}

// EmitRuntimeCallbacksTrampolineImpl writes method_trampolines_generated.m.
func EmitRuntimeCallbacksTrampolineImpl(w io.Writer, sigs []MethodSigModel) error {
	return executeTemplate(w, "method_trampolines_m_file", methodTrampolinesMFileModel{
		Sigs: buildMethodTrampolineSigModels(sigs),
	})
}

func buildMethodTrampolineSigModels(sigs []MethodSigModel) []methodTrampolineSigModel {
	models := make([]methodTrampolineSigModel, len(sigs))
	for i, sig := range sigs {
		models[i] = buildMethodTrampolineSigModel(sig)
	}
	return models
}

func buildMethodTrampolineSigModel(sig MethodSigModel) methodTrampolineSigModel {
	// GoParams: "key C.uint64_t, self unsafe.Pointer" + per-arg CGo params.
	var goParamParts []string
	goParamParts = append(goParamParts, "key C.uint64_t, self unsafe.Pointer")
	for i, a := range sig.Params {
		if a.CGOType == "unsafe.Pointer" {
			goParamParts = append(goParamParts, fmt.Sprintf("arg%d unsafe.Pointer", i))
		} else {
			goParamParts = append(goParamParts, fmt.Sprintf("arg%d %s", i, a.CGOType))
		}
	}

	// RetDecl: named return declaration for non-void functions.
	retDecl := ""
	if !sig.IsVoidRet {
		retDecl = " (result " + sig.CGoReturnType + ")"
	}

	// CallBody: the statement(s) that invoke the callback (tab-indented by template).
	cbFuncType := sig.goCallbackFuncType()
	goArgs := "self"
	for i, a := range sig.Params {
		switch a.GoType {
		case "unsafe.Pointer":
			goArgs += fmt.Sprintf(", arg%d", i)
		case "bool":
			goArgs += fmt.Sprintf(", bool(arg%d)", i)
		case "float32":
			goArgs += fmt.Sprintf(", float32(arg%d)", i)
		case "float64":
			goArgs += fmt.Sprintf(", float64(arg%d)", i)
		default:
			goArgs += fmt.Sprintf(", %s(arg%d)", a.GoType, i)
		}
	}
	var callBody string
	if sig.IsVoidRet {
		callBody = fmt.Sprintf("fn.(%s)(%s)", cbFuncType, goArgs)
	} else {
		switch sig.GoReturnType {
		case "bool":
			callBody = fmt.Sprintf("result = C.bool(fn.(%s)(%s))\n\treturn", cbFuncType, goArgs)
		case "unsafe.Pointer":
			callBody = fmt.Sprintf("result = fn.(%s)(%s)\n\treturn", cbFuncType, goArgs)
		default:
			callBody = fmt.Sprintf("result = %s(fn.(%s)(%s))\n\treturn", sig.CGoReturnType, cbFuncType, goArgs)
		}
	}

	// IMPCDecl and GoCallCDecl use the package-level helper functions, which are
	// also tested directly by unit tests.
	impCDecl := impCSignature(sig)
	goCallCDecl := goCallIMPCDecl(sig)

	// IMPBody: pre-rendered body lines for the goIMP_* C function (without braces).
	impCallArgs := "key, (void*)self"
	for i := range sig.Params {
		impCallArgs += fmt.Sprintf(", arg%d", i)
	}
	var impBody string
	if sig.IsVoidRet {
		impBody = fmt.Sprintf("    uint64_t key = goBridge_Callback_Lookup((void*)self, _cmd);\n    if (!key) { return; }\n    goCallIMP_%s(%s);", sig.Name, impCallArgs)
	} else {
		var zeroVal string
		switch sig.CReturnType {
		case "void *":
			zeroVal = "nil"
		case "bool":
			zeroVal = "false"
		default:
			zeroVal = "0"
		}
		impBody = fmt.Sprintf("    uint64_t key = goBridge_Callback_Lookup((void*)self, _cmd);\n    if (!key) { return (%s)%s; }\n    return goCallIMP_%s(%s);", sig.CReturnType, zeroVal, sig.Name, impCallArgs)
	}

	return methodTrampolineSigModel{
		Name:        sig.Name,
		ObjCEnc:     sig.ObjCEnc,
		GoParams:    strings.Join(goParamParts, ", "),
		RetDecl:     retDecl,
		CallBody:    callBody,
		IMPCDecl:    impCDecl,
		GoCallCDecl: goCallCDecl,
		IMPBody:     impBody,
	}
}

// impCSignature returns the C function declaration for a goIMP_* function.
// e.g. "void goIMP_void_ptr(id self, SEL _cmd, void* arg0)"
func impCSignature(sig MethodSigModel) string {
	params := "id self, SEL _cmd"
	for i, a := range sig.Params {
		params += fmt.Sprintf(", %s arg%d", a.CType, i)
	}
	return fmt.Sprintf("%s goIMP_%s(%s)", sig.CReturnType, sig.Name, params)
}

// goCallIMPCDecl returns the extern C declaration for a goCallIMP_* function.
func goCallIMPCDecl(sig MethodSigModel) string {
	params := "uint64_t key, void* self"
	for i, a := range sig.Params {
		params += fmt.Sprintf(", %s arg%d", a.CType, i)
	}
	if sig.IsVoidRet {
		return fmt.Sprintf("void goCallIMP_%s(%s)", sig.Name, params)
	}
	return fmt.Sprintf("%s goCallIMP_%s(%s)", sig.CReturnType, sig.Name, params)
}
