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

// ErgonomicBlockEnum emits ergonomic wrappers for BlockEnumeration-tagged methods.
//
// The generated function accepts a Go callback (fn) with the enumeration arguments
// (excluding the BOOL *stop parameter). Returning false from fn sets *stop = YES,
// which causes ObjC to stop the enumeration:
//
//	func EnumerateObjects[T objc.Object](ctx context.Context, o *raw.NSArray[T],
//	    fn func(obj T, idx uint64) bool)
//
// For non-generic classes, the Go callback uses the detected element type directly.
func EmitBlockEnum(w io.Writer, pkgName, rawImportPath string, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, nt *NameTracker) error {
	type entry struct {
		className     string
		methodName    string
		nonBlockArgs  []argInfo   // args before the block
		enumArgs      []argInfo   // block params excluding BOOL *stop
		isGenericCls  bool        // class has generic params
	}

	var entries []entry
	for _, className := range sortedKeys(framework.Classes) {
		cls := framework.Classes[className]
		if cls.Availability.IsUnavailable {
			continue
		}
		ctx := m.BaseContext(framework.Framework, knownClasses)
		ctx.ClassName = className
		localImports := make(typemap.ImportSet)

		for _, method := range cls.Methods {
			if method.Availability.IsUnavailable || method.IsClassMethod || len(method.Params) == 0 {
				continue
			}
			tags := classify.ClassifyMethod(method, cls, framework)
			if !containsPatternTag(tags, classify.BlockEnumeration) {
				continue
			}

			lastArg := method.Params[len(method.Params)-1]
			enumArgs := extractEnumBlockParams(lastArg.ObjCType, ctx, m, framework, localImports)
			if len(enumArgs) == 0 {
				continue
			}

			var nonBlock []argInfo
			for i, arg := range method.Params[:len(method.Params)-1] {
				goType := resolveOpinionatedArgType(arg.ObjCType, ctx, m, framework, localImports)
				argName := naming.ParamName(arg.Name)
				if argName == "" {
					argName = fmt.Sprintf("arg%d", i)
				}
				nonBlock = append(nonBlock, argInfo{name: argName, goType: goType})
			}

			entries = append(entries, entry{
				className:    className,
				methodName:   naming.MethodName(method.Selector),
				nonBlockArgs: nonBlock,
				enumArgs:     enumArgs,
				isGenericCls: len(cls.GenericParams) > 0,
			})
		}
	}

	if len(entries) == 0 {
		return nil
	}

	usedImports := make(map[string]string)
	needsObjc := false
	for _, e := range entries {
		for _, a := range e.nonBlockArgs {
			recordOpinionatedImports(a.goType, m, usedImports)
		}
		for _, a := range e.enumArgs {
			recordOpinionatedImports(a.goType, m, usedImports)
		}
		if e.isGenericCls {
			needsObjc = true
		}
	}

	var body bytes.Buffer

	for _, e := range entries {
		if !nt.Claim(e.methodName, "block_enum") {
			continue
		}

		// Build callback type.
		var fnParts []string
		for _, a := range e.enumArgs {
			fnParts = append(fnParts, a.name+" "+a.goType)
		}
		fnType := fmt.Sprintf("func(%s) bool", strings.Join(fnParts, ", "))

		// Build function params.
		var recvType string
		if e.isGenericCls {
			recvType = fmt.Sprintf("raw.%s[T]", e.className)
		} else {
			recvType = fmt.Sprintf("raw.%s", e.className)
		}

		params := []string{"ctx context.Context", fmt.Sprintf("o *%s", recvType)}
		for _, a := range e.nonBlockArgs {
			params = append(params, a.name+" "+a.goType)
		}
		params = append(params, "fn "+fnType)

		var genericSuffix string
		if e.isGenericCls {
			genericSuffix = "[T objc.Object]"
		}

		fmt.Fprintf(&body, "func %s%s(%s) {\n", e.methodName, genericSuffix, strings.Join(params, ", "))

		// Build call args for the raw method (all non-block args + the closure).
		var callArgs []string
		callArgs = append(callArgs, "ctx")
		for _, a := range e.nonBlockArgs {
			callArgs = append(callArgs, a.name)
		}

		// Build closure body.
		var closureParams []string
		var callbackArgs []string
		for _, a := range e.enumArgs {
			closureParams = append(closureParams, a.name+" "+a.goType)
			callbackArgs = append(callbackArgs, a.name)
		}
		// ObjC enumerate blocks end with "BOOL *stop"
		closureParams = append(closureParams, "stop *bool")

		callArgs = append(callArgs, fmt.Sprintf("func(%s) { if !fn(%s) { *stop = true } }",
			strings.Join(closureParams, ", "),
			strings.Join(callbackArgs, ", "),
		))

		fmt.Fprintf(&body, "\to.%s(%s)\n", e.methodName, strings.Join(callArgs, ", "))
		fmt.Fprintf(&body, "}\n\n")
	}

	if body.Len() == 0 {
		return nil
	}

	recordSpecialImports(body.Bytes(), usedImports)
	writeErgonomicHeader(w, pkgName, rawImportPath, usedImports, needsObjc)
	_, err := w.Write(body.Bytes())
	return err
}

// extractEnumBlockParams parses the non-stop block params from a "void (^)(…, BOOL *)" type.
// Returns argInfo entries for each param excluding the trailing BOOL * stop.
func extractEnumBlockParams(blockType string, ctx typemap.Context, m *typemap.Mapper, framework *meta.FrameworkMeta, imports typemap.ImportSet) []argInfo {
	inner := extractBlockParams(blockType)
	if inner == "" || inner == "void" {
		return nil
	}
	parts := parseBlockParams(inner)
	var out []argInfo
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "void" {
			continue
		}
		// Skip the BOOL * stop trailing param.
		if strings.TrimSpace(part) == "BOOL *" {
			continue
		}
		// "id" → objc.Object; other types mapped normally.
		var goType string
		if strings.TrimSpace(part) == "id" {
			goType = "objc.Object"
		} else {
			goType = resolveOpinionatedArgType(part, ctx, m, framework, imports)
		}
		if goType == "" {
			goType = "unsafe.Pointer"
		}
		out = append(out, argInfo{name: fmt.Sprintf("p%d", i), goType: goType})
	}
	return out
}
