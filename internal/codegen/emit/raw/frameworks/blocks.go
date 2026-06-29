package emit

import (
	"fmt"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/view"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// Block bridging.
//
// ObjC block parameters cannot be passed to purego-registered functions (or
// objc.Send) as plain Go funcs — the callee expects a real block *object*.
// Generated wrappers therefore accept an idiomatic Go closure and build the
// block via objc.NewBlock, whose callback must take objc.Block as its first
// parameter (purego v0.10.1 contract).
//
// purego's callback ABI restricts what a block callback can receive and
// return: args may be bool, integer kinds (including named enum types and
// objc.ID/SEL/Class/Block, all uintptr-kinds), floats, pointers, and
// unsafe.Pointer — but NOT strings, struct values, slices, funcs, or
// interfaces. Returns are further restricted to bool, integer kinds, and
// pointers (no floats, no structs). Components outside this set degrade the
// whole block parameter to objc.Block, letting callers construct their own
// block when they need the exotic signature.
//
// Block lifetime: objc.NewBlock returns a +1 heap block. A callee that needs
// the block beyond the call (async completion handlers) Block_copy-s it, so
// the generated `defer Release()` cannot free it early; purego's dispose
// helper drops the Go closure from its registry when the refcount hits zero.

// blockAdapterParam describes one parameter of a generated block adapter.
type blockAdapterParam struct {
	PublicGoType string // type in the public Go closure signature
	ABIType      string // type in the objc.NewBlock callback signature
	ConvertFmt   string // fmt verb with one %s converting the ABI value to the public value; "" = pass-through
	NeedsRetain  bool   // ObjC object arg — retain before wrapping so the Go finalizer owns +1
}

// blockAdapterModel describes one generated objc.NewBlock adapter.
type blockAdapterModel struct {
	ParamName     string
	PublicGoType  string // full public closure type, or "objc.Block" when degraded
	Params        []blockAdapterParam
	RetGoType     string // shared public/ABI return type; "" for void blocks
	Degraded      bool   // public param type is objc.Block; no adapter is emitted
	DegradeReason string
	UsesPureobjc  bool // adapter references the pureobjc package (char* conversion)
}

// abiPassThroughTypes are Go types that cross the purego block-callback ABI
// unchanged in argument position.
var abiPassThroughTypes = map[string]bool{
	"bool": true, "uintptr": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
	"unsafe.Pointer": true,
	"objc.ID":        true, "objc.SEL": true, "objc.Class": true, "objc.Block": true,
}

// buildBlockAdapter maps a block-typed parameter to its adapter model.
// objcType is the parameter's ObjC type (literal "(^" form or a block
// typedef); imports gains any cross-framework packages the public closure
// signature needs.
func buildBlockAdapter(
	paramName, objcType string,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	imports typemap.ImportSet,
	ownerIndex map[string]string,
) blockAdapterModel {
	adapter := blockAdapterModel{ParamName: paramName}

	signature, ok := mapper.ResolveBlockSignature(objcType)
	if !ok {
		return degradeAdapter(adapter, "unparseable block type")
	}

	// Resolve into a scratch import set so a degraded adapter (whose public
	// type is just objc.Block) does not leave unused imports behind.
	scratch := make(typemap.ImportSet)

	// Return component.
	retObjC := strings.TrimSpace(signature.ReturnObjCType)
	if retObjC != "" && retObjC != "void" {
		retGoType := mapper.GoType(retObjC, ctx, scratch)
		if !isBlockRetEncodable(retGoType, mapper, ownerIndex) {
			return degradeAdapter(
				adapter,
				fmt.Sprintf("block return type %q not bridgeable", retGoType),
			)
		}
		adapter.RetGoType = retGoType
	}

	// Parameter components.
	for _, paramObjC := range signature.ParamObjCTypes {
		component, ok := buildBlockComponent(paramObjC, ctx, mapper, scratch, ownerIndex)
		if !ok {
			return degradeAdapter(
				adapter,
				fmt.Sprintf("block component %q not bridgeable", paramObjC),
			)
		}
		if strings.Contains(component.ConvertFmt, "purego.") {
			adapter.UsesPureobjc = true
		}
		adapter.Params = append(adapter.Params, component)
	}

	// Assemble the public closure type.
	publicParams := make([]string, len(adapter.Params))
	for i, component := range adapter.Params {
		publicParams[i] = component.PublicGoType
	}
	adapter.PublicGoType = "func(" + strings.Join(publicParams, ", ") + ")"
	if adapter.RetGoType != "" {
		adapter.PublicGoType += " " + adapter.RetGoType
	}
	if imports != nil {
		for alias, path := range scratch {
			imports[alias] = path
		}
	}
	return adapter
}

// BlockGoFuncType returns the public Go type the raw emitter gives a
// block-typed parameter: the adapter's closure type, or objc.Block when the
// block cannot be bridged. Downstream emitters (idiomatic layer) must use
// this instead of Mapper.GoType so their signatures match the raw bindings.
func BlockGoFuncType(
	objcType string,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	imports typemap.ImportSet,
	ownerIndex map[string]string,
) string {
	return buildBlockAdapter("_", objcType, ctx, mapper, imports, ownerIndex).PublicGoType
}

// degradeAdapter marks the adapter as unbridgeable; the public parameter
// becomes objc.Block and is passed through verbatim.
func degradeAdapter(adapter blockAdapterModel, reason string) blockAdapterModel {
	adapter.Degraded = true
	adapter.DegradeReason = reason
	adapter.PublicGoType = "objc.Block"
	adapter.Params = nil
	adapter.RetGoType = ""
	return adapter
}

// buildBlockComponent classifies one block parameter component.
func buildBlockComponent(
	paramObjC string,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	imports typemap.ImportSet,
	ownerIndex map[string]string,
) (blockAdapterParam, bool) {
	// id<Protocol> resolves to a Go interface — not encodable; surface the raw id.
	if strings.HasPrefix(paramObjC, "id<") {
		return blockAdapterParam{PublicGoType: "objc.ID", ABIType: "objc.ID"}, true
	}

	goType := mapper.GoType(paramObjC, ctx, imports)
	if goType == "" {
		goType = "unsafe.Pointer"
	}

	// Nested block — surface as objc.Block rather than a second adapter level.
	if strings.HasPrefix(goType, "func(") {
		return blockAdapterParam{PublicGoType: "objc.Block", ABIType: "objc.Block"}, true
	}

	// char* — ABI carries the raw address; convert to a Go string.
	if goType == "string" {
		return blockAdapterParam{
			PublicGoType: "string",
			ABIType:      "uintptr",
			ConvertFmt:   "purego.GoCString(%s)",
		}, true
	}

	// ObjC object — ABI carries the raw id; retain and wrap.
	if isObjCObjectType(goType) && isObjCClass(goType, ownerIndex) {
		return blockAdapterParam{
			PublicGoType: goType,
			ABIType:      "objc.ID",
			ConvertFmt:   buildWrapExprFromTypeExpr(goType, "%s"),
			NeedsRetain:  true,
		}, true
	}

	// Encodable pass-through: primitives, enums, struct/primitive pointers.
	if abiPassThroughTypes[goType] || mapper.IsEnumType(goType) {
		return blockAdapterParam{PublicGoType: goType, ABIType: goType}, true
	}
	if strings.HasPrefix(goType, "*") && !strings.Contains(goType, "[") {
		return blockAdapterParam{PublicGoType: goType, ABIType: goType}, true
	}

	// Struct values, interfaces, arrays — not representable in a purego callback.
	return blockAdapterParam{}, false
}

// isBlockRetEncodable reports whether a Go type can be returned from a purego
// block callback (bool, integer kinds, enums, non-object pointers).
func isBlockRetEncodable(goType string, mapper *typemap.Mapper, ownerIndex map[string]string) bool {
	switch goType {
	case "bool", "uintptr",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"unsafe.Pointer", "objc.ID", "objc.SEL", "objc.Class", "objc.Block":
		return true
	}
	if mapper.IsEnumType(goType) {
		return true
	}
	// Non-object pointers (e.g. *uint8) pass through; object wrappers would
	// need a .Ptr() unwrap with nil handling — degrade those for now.
	if strings.HasPrefix(goType, "*") && !strings.Contains(goType, "[") &&
		(!isObjCObjectType(goType) || !isObjCClass(goType, ownerIndex)) {
		return true
	}
	return false
}

// blockAdapterRenderView resolves a block adapter model into its renderable view:
// the callback signature, the per-argument retain guards, and the converted call
// into the public closure. A degraded adapter renders nothing (the public
// objc.Block parameter passes through). gofmt indents the rendered body, so the
// view carries no leading whitespace.
func blockAdapterRenderView(adapter blockAdapterModel) view.BlockAdapter {
	built := view.BlockAdapter{Degraded: adapter.Degraded, ParamName: adapter.ParamName}
	if adapter.Degraded {
		return built
	}
	built.BlockVar = "__block_" + adapter.ParamName

	callbackParams := make([]string, 0, len(adapter.Params)+1)
	callbackParams = append(callbackParams, "_ objc.Block")
	for i, component := range adapter.Params {
		callbackParams = append(callbackParams, fmt.Sprintf("blockParam%d %s", i, component.ABIType))
	}
	retSig := ""
	if adapter.RetGoType != "" {
		retSig = " " + adapter.RetGoType
	}
	built.CallbackSig = "func(" + strings.Join(callbackParams, ", ") + ")" + retSig
	built.HasReturn = adapter.RetGoType != ""

	callArgs := make([]string, len(adapter.Params))
	for i, component := range adapter.Params {
		argName := fmt.Sprintf("blockParam%d", i)
		built.Params = append(built.Params, view.BlockCallbackParam{ArgName: argName, NeedsRetain: component.NeedsRetain})
		if component.ConvertFmt != "" {
			callArgs[i] = fmt.Sprintf(component.ConvertFmt, argName)
		} else {
			callArgs[i] = argName
		}
	}
	built.CallExpr = fmt.Sprintf("%s(%s)", adapter.ParamName, strings.Join(callArgs, ", "))
	return built
}
