package rawlib

import (
	"fmt"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/view"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
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

// CollectBlockSignaturesFromFrameworks scans all methods/protocols/extensions
// in the provided list of framework metas and returns the deduplicated, sorted
// set of BlockSignatures that need Go trampolines.
func CollectBlockSignaturesFromFrameworks(frameworks []*macosplatformmetadata.FrameworkMeta, m *typemap.Mapper) []BlockSignatureModel {
	seen := make(map[string]BlockSignatureModel)

	// resolveBlockObjCType returns the concrete ObjC block type string for an arg,
	// whether the block syntax is inline (IsBlock, e.g. "void (^)(NSError *)") or
	// hidden behind a named typedef (e.g. "vmnet_start_interface_completion_handler_t").
	// typedefs may be nil when the caller has no typedef map to consult.
	// Returns "" when the arg is not a block type.
	resolveBlockObjCType := func(arg macosplatformmetadata.Param, typedefs map[string]string) string {
		if arg.IsBlock {
			return arg.ObjCType
		}
		if typedefs == nil {
			return ""
		}
		n := typemap.Normalise(arg.ObjCType)
		if target, ok := typedefs[n]; ok && typemap.IsBlock(target) {
			return target
		}
		return ""
	}

	addArgs := func(args []macosplatformmetadata.Param, typedefs map[string]string) {
		for _, arg := range args {
			blockType := resolveBlockObjCType(arg, typedefs)
			if blockType == "" {
				continue
			}
			sig, ok := BlockSigFromObjC(blockType, m)
			if !ok {
				continue
			}
			if _, exists := seen[sig.Name]; !exists {
				seen[sig.Name] = sig
			}
		}
	}

	for _, framework := range frameworks {
		for _, cls := range framework.Classes {
			for _, method := range cls.Methods {
				addArgs(method.Params, framework.Typedefs)
			}
		}
		for _, methods := range framework.ForeignExtensions {
			for _, method := range methods {
				addArgs(method.Params, framework.Typedefs)
			}
		}
		for _, proto := range framework.Protocols {
			for _, method := range proto.Methods {
				addArgs(method.Params, framework.Typedefs)
			}
		}
		for _, fn := range framework.Functions {
			addArgs(fn.Params, framework.Typedefs)
		}
	}

	sigs := make([]BlockSignatureModel, 0, len(seen))
	for _, sig := range seen {
		sigs = append(sigs, sig)
	}
	sort.Slice(sigs, func(i, j int) bool { return sigs[i].Name < sigs[j].Name })
	return sigs
}

// EmitRuntimeBlocksGo writes the generated runtime/blocks_generated.go file.
// This file contains //export goCallBlock_* callbacks and MakeBlock_* factories.
func EmitRuntimeBlocksGo(w io.Writer, sigs []BlockSignatureModel, pkg string) error {
	return render.Execute(w, "block_trampolines_go_file", view.BlockTrampolinesGoFileModel{
		PkgName: pkg,
		Sigs:    buildBlockTrampolineSigModels(sigs),
	})
}

// EmitRuntimeBlocksTrampolineHeader writes runtime/block_trampolines_generated.h.
func EmitRuntimeBlocksTrampolineHeader(w io.Writer, sigs []BlockSignatureModel) error {
	return render.Execute(w, "block_trampolines_h_file", view.BlockTrampolinesHFileModel{
		Sigs: buildBlockTrampolineSigModels(sigs),
	})
}

// EmitRuntimeBlocksTrampolineImpl writes runtime/block_trampolines_generated.m.
func EmitRuntimeBlocksTrampolineImpl(w io.Writer, sigs []BlockSignatureModel) error {
	return render.Execute(w, "block_trampolines_m_file", view.BlockTrampolinesMFileModel{
		Sigs: buildBlockTrampolineSigModels(sigs),
	})
}

func buildBlockTrampolineSigModels(sigs []BlockSignatureModel) []view.BlockTrampolineSigModel {
	models := make([]view.BlockTrampolineSigModel, len(sigs))
	for i, sig := range sigs {
		models[i] = buildBlockTrampolineSigModel(sig)
	}
	return models
}

func buildBlockTrampolineSigModel(sig BlockSignatureModel) view.BlockTrampolineSigModel {
	// GoParams: CGo parameter list for the exported function.
	var goParamParts []string
	goParamParts = append(goParamParts, "key C.uint64_t")
	for i, arg := range sig.Args {
		if arg.CGOType == "unsafe.Pointer" {
			goParamParts = append(goParamParts, fmt.Sprintf("arg%d unsafe.Pointer", i))
		} else {
			goParamParts = append(goParamParts, fmt.Sprintf("arg%d %s", i, arg.CGOType))
		}
	}

	// RetDecl: named return declaration for non-void functions.
	retDecl := ""
	if !sig.IsVoidRet {
		retDecl = " (result " + sig.RetCGo + ")"
	}

	// ClosureType: Go func type for type-asserting the stored closure.
	var closureArgTypes []string
	for _, arg := range sig.Args {
		closureArgTypes = append(closureArgTypes, arg.GoType)
	}
	closureType := "func(" + strings.Join(closureArgTypes, ", ") + ")"
	if !sig.IsVoidRet {
		closureType += " " + sig.RetGo
	}

	// GoCallArgs: arguments to pass to the closure.
	var goCallArgs []string
	for i, arg := range sig.Args {
		switch {
		case arg.CGOType == "unsafe.Pointer":
			goCallArgs = append(goCallArgs, fmt.Sprintf("arg%d", i))
		case arg.GoType != arg.CGOType:
			goCallArgs = append(goCallArgs, fmt.Sprintf("%s(arg%d)", arg.GoType, i))
		default:
			goCallArgs = append(goCallArgs, fmt.Sprintf("arg%d", i))
		}
	}
	callArgsStr := strings.Join(goCallArgs, ", ")

	// CallBody: the statement(s) that invoke the closure (tab-indented by template).
	var callBody string
	if sig.IsVoidRet {
		callBody = fmt.Sprintf("fn(%s)", callArgsStr)
	} else {
		callBody = fmt.Sprintf("result = %s(fn(%s))\n\treturn", sig.RetCGo, callArgsStr)
	}

	// C file fields.
	cRetType := sig.RetC
	if cRetType == "" {
		cRetType = "void"
	}

	// CForwardParams: C parameter list for the extern forward declaration.
	var cForwardParts []string
	cForwardParts = append(cForwardParts, "uint64_t key")
	for i, arg := range sig.Args {
		cForwardParts = append(cForwardParts, fmt.Sprintf("%s arg%d", arg.CType, i))
	}

	// CTrampolineParams: C parameter list for the static trampoline function.
	var cTrampolineParts []string
	cTrampolineParts = append(cTrampolineParts, "struct GoBlock *block")
	for i, arg := range sig.Args {
		cTrampolineParts = append(cTrampolineParts, fmt.Sprintf("%s arg%d", arg.CType, i))
	}

	// CTrampolineCall: the call body of the static trampoline.
	var trampolineCallArgs []string
	trampolineCallArgs = append(trampolineCallArgs, "block->goKey")
	for i := range sig.Args {
		trampolineCallArgs = append(trampolineCallArgs, fmt.Sprintf("arg%d", i))
	}
	callStr := fmt.Sprintf("goCallBlock_%s(%s)", sig.Name, strings.Join(trampolineCallArgs, ", "))
	var cTrampolineCall string
	if sig.IsVoidRet {
		cTrampolineCall = callStr + ";"
	} else {
		cTrampolineCall = fmt.Sprintf("return (%s)%s;", cRetType, callStr)
	}

	return view.BlockTrampolineSigModel{
		Name:              sig.Name,
		GoParams:          strings.Join(goParamParts, ", "),
		RetDecl:           retDecl,
		ClosureType:       closureType,
		CallBody:          callBody,
		CRetType:          cRetType,
		CForwardParams:    strings.Join(cForwardParts, ", "),
		CTrampolineParams: strings.Join(cTrampolineParts, ", "),
		CTrampolineCall:   cTrampolineCall,
	}
}
