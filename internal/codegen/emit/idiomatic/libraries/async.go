package library

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// Async writes blocking wrappers for completion-handler methods.
//
// For every instance method whose last block argument matches the pattern
// "void (^...)(NSError *)", a package-level function is emitted that blocks
// via a buffered channel until the handler fires or ctx is cancelled.
//
// pkgName is the Go package name (e.g. "virtualization"); rawImportPath is the
// full import path of the raw framework package.
func EmitAsync(w io.Writer, pkgName, rawImportPath string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) error {
	type entry struct {
		className     string
		goName        string // opinionated name (CompletionHandler suffix stripped)
		rawMethod     string // original Go method name on the raw type
		isClassMethod bool
		nonBlockArgs  []argInfo
	}

	var entries []entry
	for _, className := range sortedKeys(framework.Classes) {
		class := framework.Classes[className]
		if class.Availability.IsUnavailable {
			continue
		}
		ctx := m.BaseContext(framework.Framework, knownClasses)
		ctx.ClassName = className
		localImports := make(typemap.ImportSet)

		for _, method := range class.Methods {
			if method.Availability.IsUnavailable {
				continue
			}
			blkIdx := completionHandlerIndex(method.Params)
			if blkIdx < 0 {
				continue
			}
			rawMethodName := naming.MethodName(method.Selector)
			opinionatedName := removeCompletionSuffix(rawMethodName)
			if opinionatedName == "" {
				continue
			}
			var nonBlock []argInfo
			for i, arg := range method.Params {
				if i == blkIdx {
					continue
				}
				goType := resolveOpinionatedArgType(arg.ObjCType, ctx, m, framework, localImports)
				argName := naming.ParamName(arg.Name)
				if argName == "" {
					argName = fmt.Sprintf("arg%d", i)
				}
				nonBlock = append(nonBlock, argInfo{name: argName, goType: goType})
			}
			entries = append(entries, entry{
				className:     className,
				goName:        opinionatedName,
				rawMethod:     rawMethodName,
				isClassMethod: method.IsClassMethod,
				nonBlockArgs:  nonBlock,
			})
		}
	}

	if len(entries) == 0 {
		return nil
	}

	usedImports := make(map[string]string)
	for _, e := range entries {
		for _, a := range e.nonBlockArgs {
			recordOpinionatedImports(a.goType, m, usedImports)
		}
	}

	var body bytes.Buffer
	seen := make(map[string]bool)
	for _, e := range entries {
		key := e.className + "." + e.goName
		if seen[key] {
			continue
		}
		seen[key] = true

		var params []string
		var callArgs []string
		if e.isClassMethod {
			params = []string{"ctx context.Context"}
		} else {
			params = []string{
				"ctx context.Context",
				fmt.Sprintf("recv *raw.%s", e.className),
			}
		}
		for _, a := range e.nonBlockArgs {
			params = append(params, a.name+" "+a.goType)
			callArgs = append(callArgs, a.name)
		}
		callArgs = append(callArgs, "func(err error) { _ch <- err }")

		fmt.Fprintf(&body, "// %s calls %s.%s and blocks until the operation completes or ctx is cancelled.\n",
			e.goName, e.className, e.rawMethod)
		fmt.Fprintf(&body, "func %s(%s) error {\n", e.goName, strings.Join(params, ", "))
		fmt.Fprintf(&body, "\t_ch := make(chan error, 1)\n")
		if e.isClassMethod {
			fmt.Fprintf(&body, "\traw.%s%s(%s)\n", e.className, e.rawMethod, strings.Join(callArgs, ", "))
		} else {
			fmt.Fprintf(&body, "\trecv.%s(%s)\n", e.rawMethod, strings.Join(callArgs, ", "))
		}
		fmt.Fprintf(&body, "\tselect {\n")
		fmt.Fprintf(&body, "\tcase err := <-_ch:\n\t\treturn err\n")
		fmt.Fprintf(&body, "\tcase <-ctx.Done():\n\t\treturn ctx.Err()\n")
		fmt.Fprintf(&body, "\t}\n")
		fmt.Fprintf(&body, "}\n\n")
	}

	if body.Len() == 0 {
		return nil
	}

	// The emitted blocking wrappers take ctx for cancellation (select on ctx.Done()).
	writeOpinionatedHeader(w, pkgName, rawImportPath, map[string]string{"context": "context"}, usedImports, false)
	_, err := w.Write(body.Bytes())
	return err
}
