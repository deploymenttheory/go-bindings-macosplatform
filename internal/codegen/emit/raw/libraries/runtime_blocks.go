package rawlib

import (
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
)

// BlockSignatureModel describes a unique primitive-typed block signature.
// Two ObjC block types that differ only in ObjC object types (e.g. id vs NSString *)
// produce the same BlockSignatureModel because they share the same C/Go representation.
type BlockSignatureModel struct {
	Name      string // canonical C-safe name: "void_ptr_ptr", "int64_ptr_ptr"
	IsVoidRet bool   // true when block returns void
	RetC      string // C return type: "int64_t", "bool", "" for void
	RetCGo    string // CGo type in //export: "C.int64_t", "C.bool", "" for void
	RetGo     string // Go return type: "int64", "bool", "" for void
	Args      []BlockSigArg
}

// BlockSigArg describes a single argument in a BlockSignatureModel.
type BlockSigArg struct {
	CType   string // C type in the trampoline: "void *", "int64_t", "bool"
	CGOType string // CGo type in the //export function: "unsafe.Pointer", "C.int64_t"
	GoType  string // Go type in MakeBlock factory: "unsafe.Pointer", "int64", "bool"
}

// blockSigTypeInfo maps a token string to its C, CGo, and Go representations.
type blockSigTypeInfo struct{ cType, cgoType, goType string }

var tokenTypeMap = map[string]blockSigTypeInfo{
	"ptr":     {"void *", "unsafe.Pointer", "unsafe.Pointer"},
	"bool":    {"bool", "C.bool", "bool"},
	"int8":    {"int8_t", "C.int8_t", "int8"},
	"int16":   {"int16_t", "C.int16_t", "int16"},
	"int32":   {"int32_t", "C.int32_t", "int32"},
	"int64":   {"int64_t", "C.int64_t", "int64"},
	"uint8":   {"uint8_t", "C.uint8_t", "uint8"},
	"uint16":  {"uint16_t", "C.uint16_t", "uint16"},
	"uint32":  {"uint32_t", "C.uint32_t", "uint32"},
	"uint64":  {"uint64_t", "C.uint64_t", "uint64"},
	"float32": {"float", "C.float", "float32"},
	"float64": {"double", "C.double", "float64"},
}

// cTypeToken maps a C type string to its canonical MakeBlock_* token name.
// "ptr" means the type was not recognised — it is a fallback that maps to void*/unsafe.Pointer.
func cTypeToken(cType string) string {
	switch cType {
	case "void", "":
		return "void"
	case "void *":
		return "ptr"
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
	}
	return "ptr"
}

// BlockSigName returns the canonical C-safe name for an ObjC block type string.
// Two ObjC block types that share the same primitive C representation produce the
// same name (e.g. "void (^)(id)" and "void (^)(NSError *)" both → "void_ptr").
// Returns "" when the block type cannot be parsed.
func BlockSigName(objcType string, m *typemap.Mapper) string {
	return blockSigNameFromParsed(objcType, m, typemap.Context{})
}

func blockSigNameFromParsed(objcType string, m *typemap.Mapper, ctx typemap.Context) string {
	retType, args, ok := typemap.ParseBlock(objcType)
	if !ok {
		return ""
	}
	emptyCtx := typemap.Context{ClassNameIndex: ctx.ClassNameIndex}
	var parts []string
	parts = append(parts, cTypeToken(m.CType(retType, emptyCtx, nil)))
	for _, a := range args {
		parts = append(parts, cTypeToken(m.CType(a, emptyCtx, nil)))
	}
	return strings.Join(parts, "_")
}

// BlockSigFromObjC builds a BlockSignatureModel from an ObjC block type string.
// Returns the zero value and false if the type cannot be parsed.
func BlockSigFromObjC(objcType string, m *typemap.Mapper) (BlockSignatureModel, bool) {
	retType, args, ok := typemap.ParseBlock(objcType)
	if !ok {
		return BlockSignatureModel{}, false
	}

	var retToken string
	var retC, retCGo, retGo string
	isVoid := typemap.IsVoid(retType) || retType == ""

	ctx := typemap.Context{}
	if isVoid {
		retToken = "void"
	} else {
		retC = m.CType(retType, ctx, nil)
		retToken = cTypeToken(retC)
		if info, ok := tokenTypeMap[retToken]; ok {
			retC = info.cType
			retCGo = info.cgoType
			retGo = info.goType
		} else {
			retC = "void *"
			retCGo = "unsafe.Pointer"
			retGo = "unsafe.Pointer"
			retToken = "ptr"
		}
	}

	var sigArgs []BlockSigArg
	var argTokens []string
	for _, a := range args {
		if typemap.IsVoid(a) || a == "" {
			continue
		}
		cType := m.CType(a, ctx, nil)
		tok := cTypeToken(cType)
		argTokens = append(argTokens, tok)
		info := tokenTypeMap[tok]
		if tok == "void" || tok == "" {
			continue
		}
		if (blockSigTypeInfo{}) == info {
			info = tokenTypeMap["ptr"]
			tok = "ptr"
		}
		sigArgs = append(sigArgs, BlockSigArg{
			CType:   info.cType,
			CGOType: info.cgoType,
			GoType:  info.goType,
		})
	}

	nameParts := []string{retToken}
	nameParts = append(nameParts, argTokens...)
	name := strings.Join(nameParts, "_")

	return BlockSignatureModel{
		Name:      name,
		IsVoidRet: isVoid,
		RetC:      retC,
		RetCGo:    retCGo,
		RetGo:     retGo,
		Args:      sigArgs,
	}, true
}
