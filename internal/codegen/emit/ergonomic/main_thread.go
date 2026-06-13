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

// ErgonomicMainThread emits package-level wrappers for MainThreadRequired-tagged methods.
//
// Each generated function dispatches the raw method call through objc.RunOnMainThread.
// Return values are captured via a closure variable that is readable after RunOnMainThread
// returns (it blocks until the closure completes):
//
//	func UpdateUI(ctx context.Context, o *raw.NSView) {
//	    objc.RunOnMainThread(func() { o.UpdateUI(ctx) })
//	}
func EmitMainThread(w io.Writer, pkgName, rawImportPath string, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, nt *NameTracker) error {
	type entry struct {
		className  string
		methodName string
		args       []argInfo
		retParts   []string // Go return types (empty if void)
		hasError   bool
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
			if !containsPatternTag(tags, classify.MainThreadRequired) {
				continue
			}
			// AsyncCompletion methods are handled by ErgonomicAsync with a channel-based
			// wrapper that is more ergonomic than a plain RunOnMainThread call. Skip here
			// to avoid conflicting declarations.
			if containsPatternTag(tags, classify.AsyncCompletion) {
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

			var retParts []string
			retObjC := strings.TrimSpace(method.Return.ObjCType)
			if retObjC != "" && retObjC != "void" {
				retType := resolveOpinionatedArgType(retObjC, ctx, m, framework, localImports)
				if retType != "" && retType != "unsafe.Pointer" {
					retParts = append(retParts, retType)
				}
			}

			entries = append(entries, entry{
				className:  className,
				methodName: naming.MethodName(method.Selector),
				args:       args,
				retParts:   retParts,
				hasError:   method.IsNSError,
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
		for _, r := range e.retParts {
			recordOpinionatedImports(r, m, usedImports)
		}
	}
	usedImports["objc"] = "github.com/deploymenttheory/go-bindings-macosplatform/internal/objc"

	var body bytes.Buffer

	for _, e := range entries {
		if !nt.Claim(e.methodName, "main_thread") {
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

		allRetParts := append([]string(nil), e.retParts...)
		if e.hasError {
			allRetParts = append(allRetParts, "error")
		}

		var retSig string
		switch len(allRetParts) {
		case 0:
			retSig = ""
		case 1:
			retSig = " " + allRetParts[0]
		default:
			retSig = " (" + strings.Join(allRetParts, ", ") + ")"
		}

		fmt.Fprintf(&body, "func %s(%s)%s {\n", e.methodName, strings.Join(params, ", "), retSig)

		if len(allRetParts) == 0 {
			fmt.Fprintf(&body, "\tobjc.RunOnMainThread(func() {\n")
			fmt.Fprintf(&body, "\t\to.%s(%s)\n", e.methodName, strings.Join(allCallArgs, ", "))
			fmt.Fprintf(&body, "\t})\n}\n\n")
		} else {
			// Capture return values via closure variables.
			for i, r := range e.retParts {
				fmt.Fprintf(&body, "\tvar _ret%d %s\n", i, r)
			}
			if e.hasError {
				fmt.Fprintf(&body, "\tvar _err error\n")
			}
			fmt.Fprintf(&body, "\tobjc.RunOnMainThread(func() {\n")

			var lhs []string
			for i := range e.retParts {
				lhs = append(lhs, fmt.Sprintf("_ret%d", i))
			}
			if e.hasError {
				lhs = append(lhs, "_err")
			}
			fmt.Fprintf(&body, "\t\t%s = o.%s(%s)\n", strings.Join(lhs, ", "), e.methodName, strings.Join(allCallArgs, ", "))
			fmt.Fprintf(&body, "\t})\n")

			var retVals []string
			for i := range e.retParts {
				retVals = append(retVals, fmt.Sprintf("_ret%d", i))
			}
			if e.hasError {
				retVals = append(retVals, "_err")
			}
			fmt.Fprintf(&body, "\treturn %s\n}\n\n", strings.Join(retVals, ", "))
		}
	}

	if body.Len() == 0 {
		return nil
	}

	recordSpecialImports(body.Bytes(), usedImports)
	writeErgonomicHeader(w, pkgName, rawImportPath, usedImports, false)
	_, err := w.Write(body.Bytes())
	return err
}
