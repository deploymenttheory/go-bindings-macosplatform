package emit

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/typemap"
)

// EmitProtocols writes Go interface types for all ObjC protocols in the framework.
func EmitProtocols(
	w io.Writer,
	framework *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	classNameOwner map[string]string,
) (typemap.ImportSet, error) {
	imports := make(typemap.ImportSet)

	names := make([]string, 0, len(framework.Protocols))
	for name := range framework.Protocols {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		proto := framework.Protocols[name]
		if proto.Availability.IsUnavailable {
			continue
		}
		if err := emitProtocol(
			w,
			name,
			proto,
			framework.Framework,
			mapper,
			classNameOwner,
			imports,
		); err != nil {
			return nil, err
		}
	}
	return imports, nil
}

func emitProtocol(
	w io.Writer,
	name string,
	proto meta.Protocol,
	framework string,
	mapper *typemap.Mapper,
	classNameOwner map[string]string,
	imports typemap.ImportSet,
) error {
	goName := naming.ProtocolGoTypeName(name, classNameOwner)

	fmt.Fprintf(w, "// %s wraps the ObjC protocol %s.\n", goName, name)
	fmt.Fprintf(w, "type %s interface {\n", goName)

	// Embed inherited protocols.
	for _, parent := range proto.InheritedProtocols {
		if parent == "NSObject" {
			continue // NSObject protocol is implicit
		}
		parentGoName := naming.ProtocolGoTypeName(parent, classNameOwner)
		// Check if cross-framework
		if owner, ok := mapper.ProtocolIndex[parent]; ok && owner != framework {
			pkg := strings.ToLower(owner)
			path := mapper.ModulePrefix + "/" + pkg
			imports[pkg] = path
			parentGoName = pkg + "." + parentGoName
		}
		fmt.Fprintf(w, "\t%s\n", parentGoName)
	}

	ctx := typemap.Context{
		Framework:      framework,
		ClassNameIndex: classNameOwner,
	}

	// Methods
	seenGoNames := make(map[string]bool)
	for _, method := range proto.Methods {
		if method.Availability.IsUnavailable || method.IsOptional {
			continue
		}
		goMethodName := naming.MethodName(method.Selector)
		if seenGoNames[goMethodName] {
			continue
		}
		seenGoNames[goMethodName] = true

		sig := buildMethodSignature(method, ctx, mapper, imports, false)
		fmt.Fprintf(w, "\t%s%s\n", goMethodName, sig)
	}

	fmt.Fprintf(w, "}\n\n")
	return nil
}

// buildMethodSignature builds the Go method signature (params and return) for
// an ObjC method. className is empty for protocol methods.
func buildMethodSignature(
	method meta.Method,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	imports typemap.ImportSet,
	includeReceiver bool,
) string {
	// Build parameter list.
	var params []string
	for _, param := range method.Params {
		if param.IsBlock {
			blockType := buildBlockFuncType(param.ObjCType, ctx, mapper, imports)
			paramName := naming.ParamName(param.Name)
			params = append(params, paramName+" "+blockType)
			continue
		}
		goType := mapper.GoType(param.ObjCType, ctx, imports)
		if goType == "" {
			goType = "unsafe.Pointer"
		}
		paramName := naming.ParamName(param.Name)
		params = append(params, paramName+" "+goType)
	}

	// Build return type.
	retType := ""
	hasError := method.IsNSError
	if method.Return.ObjCType != "void" && method.Return.ObjCType != "" {
		switch {
		case method.Return.IsInstancetype && ctx.ClassName != "":
			retType = "*" + ctx.ClassName
		default:
			if _, retIsBlock := mapper.ResolveBlockSignature(method.Return.ObjCType); retIsBlock {
				// Returned blocks surface as objc.Block (matches writeMethod).
				retType = "objc.Block"
			} else {
				retType = mapper.GoReturnType(method.Return.ObjCType, ctx, imports)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("(")
	sb.WriteString(strings.Join(params, ", "))
	sb.WriteString(")")

	switch {
	case retType != "" && hasError:
		sb.WriteString(" (")
		sb.WriteString(retType)
		sb.WriteString(", error)")
	case hasError:
		sb.WriteString(" error")
	case retType != "":
		sb.WriteString(" ")
		sb.WriteString(retType)
	}

	return sb.String()
}

// buildBlockFuncType builds the public Go type for a block-typed parameter.
// It mirrors what writeMethod emits for class implementations (the adapter's
// closure type, or objc.Block when the block cannot be bridged) so protocol
// signatures stay aligned with the generated methods.
func buildBlockFuncType(
	blockType string,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	imports typemap.ImportSet,
) string {
	return buildBlockAdapter("_", blockType, ctx, mapper, imports, ctx.ClassNameIndex).PublicGoType
}
