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

// ErgonomicConstructors emits package-level ergonomic wrappers for BoolNSError and
// NSErrorOut methods. BoolNSError wrappers drop the redundant bool return; NSErrorOut
// wrappers are pass-through aliases with the ergonomic package-level function shape.
func EmitConstructors(w io.Writer, pkgName, rawImportPath string, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, nt *NameTracker) error {
	type entry struct {
		className  string
		methodName string
		args       []argInfo
		retType    string // "" means bool return that gets dropped; otherwise the real return type
		isBool     bool   // true → BoolNSError
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
			if method.Availability.IsUnavailable || method.IsClassMethod {
				continue
			}
			tags := classify.ClassifyMethod(method, cls, framework)
			hasBool := containsPatternTag(tags, classify.BoolNSError)
			hasNSErr := containsPatternTag(tags, classify.NSErrorOut)
			if !hasBool && !hasNSErr {
				continue
			}

			var args []argInfo
			for i, arg := range method.Params {
				goType := resolveOpinionatedArgType(arg.ObjCType, ctx, m, framework, localImports)
				argName := naming.ParamName(arg.Name)
				if argName == "" {
					argName = fmt.Sprintf("arg%d", i)
				}
				args = append(args, argInfo{name: argName, goType: goType})
			}

			retType := ""
			if hasNSErr {
				retType = resolveOpinionatedArgType(method.Return.ObjCType, ctx, m, framework, localImports)
			}

			entries = append(entries, entry{
				className:  className,
				methodName: naming.MethodName(method.Selector),
				args:       args,
				retType:    retType,
				isBool:     hasBool,
			})
		}
	}

	if len(entries) == 0 {
		return nil
	}

	usedImports := make(map[string]string)
	for _, e := range entries {
		for _, a := range e.args {
			recordOpinionatedImports(a.goType, m, usedImports)
		}
		if e.retType != "" {
			recordOpinionatedImports(e.retType, m, usedImports)
		}
	}

	var body bytes.Buffer
	for _, e := range entries {
		if !nt.Claim(e.methodName, "constructors") {
			continue
		}

		params := []string{
			"ctx context.Context",
			fmt.Sprintf("o *%s", buildRawReceiverType(e.className, m)),
		}
		var callArgs []string
		for _, a := range e.args {
			params = append(params, a.name+" "+a.goType)
			callArgs = append(callArgs, a.name)
		}
		allCallArgs := append([]string{"ctx"}, callArgs...)

		if e.isBool {
			fmt.Fprintf(&body, "func %s(%s) error {\n", e.methodName, strings.Join(params, ", "))
			fmt.Fprintf(&body, "\t_, err := o.%s(%s)\n", e.methodName, strings.Join(allCallArgs, ", "))
			fmt.Fprintf(&body, "\treturn err\n}\n\n")
		} else {
			retType := e.retType
			if retType == "" {
				retType = "any"
			}
			fmt.Fprintf(&body, "func %s(%s) (%s, error) {\n", e.methodName, strings.Join(params, ", "), retType)
			fmt.Fprintf(&body, "\treturn o.%s(%s)\n}\n\n", e.methodName, strings.Join(allCallArgs, ", "))
		}
	}

	if body.Len() == 0 {
		return nil
	}

	// Add objc/unsafe imports when referenced in the body.
	recordSpecialImports(body.Bytes(), usedImports)
	writeErgonomicHeader(w, pkgName, rawImportPath, usedImports, false)
	_, err := w.Write(body.Bytes())
	return err
}

