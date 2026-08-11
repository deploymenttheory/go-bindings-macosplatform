package rawlib

import (
	"fmt"
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
