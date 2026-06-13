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

// asyncBlockResult holds one non-error result value parsed from a block type string.
type asyncBlockResult struct {
	name   string
	goType string
}

// ErgonomicAsync emits sync-blocking wrappers for AsyncCompletion-tagged methods.
//
// Each generated function:
//   - Removes the trailing block argument
//   - Creates a buffered channel of one result struct
//   - Passes a Go closure as the completion handler
//   - Selects on the channel or ctx.Done()
//   - Returns (blockDataValues..., error)
func EmitAsync(w io.Writer, pkgName, rawImportPath string, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, nt *NameTracker) error {
	type entry struct {
		className         string
		methodName        string
		nonBlockArgs      []argInfo
		asyncBlockResults []asyncBlockResult // non-error, non-BOOL params from the block
		hasBlockError     bool               // block has an NSError * param
		// rawBlockParams holds the complete Go param types from the block func type
		// (e.g. ["*foundation.NSData", "error"]) — used to build the closure signature
		// so it exactly matches what the raw binding expects.
		rawBlockParams []string
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
			if !containsPatternTag(tags, classify.AsyncCompletion) {
				continue
			}

			lastArg := method.Params[len(method.Params)-1]

			// Use GoBlockUserFuncType to get the exact func type that the raw binding
			// expects. This ensures our closure signature exactly matches the raw method.
			blockFuncType := m.GoBlockUserFuncType(lastArg.ObjCType, ctx, localImports)
			if blockFuncType == "" {
				continue // unparseable block type — skip
			}

			rawParamsRaw := extractFuncParams(blockFuncType)
			rawParams := make([]string, len(rawParamsRaw))
			for i, p := range rawParamsRaw {
				rawParams[i] = qualifyWithRawPackage(strings.TrimSpace(p), framework)
			}
			nonBlockArgs, asyncBRs, hasBlockErr := parseAsyncFromFuncType(blockFuncType, method.Params[:len(method.Params)-1], ctx, m, framework, localImports)

			entries = append(entries, entry{
				className:         className,
				methodName:        naming.MethodName(method.Selector),
				nonBlockArgs:      nonBlockArgs,
				asyncBlockResults: asyncBRs,
				hasBlockError:     hasBlockErr,
				rawBlockParams:    rawParams,
			})
		}
	}

	if len(entries) == 0 {
		return nil
	}

	usedImports := make(map[string]string)
	needsObjc := false
	needsUnsafe := false
	for _, e := range entries {
		for _, a := range e.nonBlockArgs {
			recordOpinionatedImports(a.goType, m, usedImports)
			if strings.Contains(a.goType, "objc.") {
				needsObjc = true
			}
			if strings.Contains(a.goType, "unsafe.") {
				needsUnsafe = true
			}
		}
		for _, r := range e.asyncBlockResults {
			recordOpinionatedImports(r.goType, m, usedImports)
			if strings.Contains(r.goType, "objc.") {
				needsObjc = true
			}
			if strings.Contains(r.goType, "unsafe.") {
				needsUnsafe = true
			}
		}
	}
	if needsUnsafe {
		usedImports["unsafe"] = "unsafe"
	}

	var body bytes.Buffer

	for _, e := range entries {
		// Deduplicate by emitted Go function name across all ergonomic emitters.
		if !nt.Claim(e.methodName, "async") {
			continue
		}

		// Parameters: ctx, receiver, then non-block args.
		params := []string{
			"ctx context.Context",
			fmt.Sprintf("o *%s", buildRawReceiverType(e.className, m)),
		}
		var callArgs []string
		for _, a := range e.nonBlockArgs {
			params = append(params, a.name+" "+a.goType)
			callArgs = append(callArgs, a.name)
		}

		// Build return type.
		var retParts []string
		for _, r := range e.asyncBlockResults {
			retParts = append(retParts, r.goType)
		}
		if e.hasBlockError {
			retParts = append(retParts, "error")
		}

		var retSig string
		switch len(retParts) {
		case 0:
			retSig = ""
		case 1:
			retSig = " " + retParts[0]
		default:
			retSig = " (" + strings.Join(retParts, ", ") + ")"
		}

		// Build result struct type for the channel.
		hasResults := len(e.asyncBlockResults) > 0 || e.hasBlockError

		if hasResults {
			// Declare a local result type for the channel.
			fmt.Fprintf(&body, "type _%sResult struct {\n", e.methodName)
			for _, r := range e.asyncBlockResults {
				goName := strings.ToUpper(r.name[:1]) + r.name[1:]
				fmt.Fprintf(&body, "\t%s %s\n", goName, r.goType)
			}
			if e.hasBlockError {
				fmt.Fprintf(&body, "\tErr error\n")
			}
			fmt.Fprintf(&body, "}\n\n")
		}

		fmt.Fprintf(&body, "func %s(%s)%s {\n", e.methodName, strings.Join(params, ", "), retSig)

		if hasResults {
			fmt.Fprintf(&body, "\t_ch := make(chan _%sResult, 1)\n", e.methodName)
		} else {
			fmt.Fprintf(&body, "\t_ch := make(chan struct{}, 1)\n")
		}

		// Build closure arguments from the FULL raw block signature so the
		// generated closure type exactly matches what the raw binding expects.
		// asyncBlockResults names use "v<rawIndex>" so they align with rawBlockParams.
		var closureParams []string
		for i, rawType := range e.rawBlockParams {
			rawType = strings.TrimSpace(rawType)
			switch {
			case rawType == "error":
				closureParams = append(closureParams, "err error")
			case rawType == "bool" || rawType == "*bool":
				closureParams = append(closureParams, fmt.Sprintf("_p%d %s", i, rawType))
			default:
				// Check if captured as an asyncBlockResult (name matches v<i>).
				captured := false
				for _, r := range e.asyncBlockResults {
					if r.name == fmt.Sprintf("v%d", i) {
						closureParams = append(closureParams, fmt.Sprintf("v%d %s", i, rawType))
						captured = true
						break
					}
				}
				if !captured {
					closureParams = append(closureParams, fmt.Sprintf("_p%d %s", i, rawType))
				}
			}
		}

		// Build send expression.
		if hasResults {
			var fields []string
			for _, r := range e.asyncBlockResults {
				goName := strings.ToUpper(r.name[:1]) + r.name[1:]
				fields = append(fields, fmt.Sprintf("%s: %s", goName, r.name))
			}
			if e.hasBlockError {
				fields = append(fields, "Err: err")
			}
			fmt.Fprintf(&body, "\to.%s(%s, func(%s) { _ch <- _%sResult{%s} })\n",
				e.methodName,
				strings.Join(append([]string{"ctx"}, callArgs...), ", "),
				strings.Join(closureParams, ", "),
				e.methodName,
				strings.Join(fields, ", "),
			)
		} else {
			fmt.Fprintf(&body, "\to.%s(%s, func(%s) { _ch <- struct{}{} })\n",
				e.methodName,
				strings.Join(append([]string{"ctx"}, callArgs...), ", "),
				strings.Join(closureParams, ", "),
			)
		}

		fmt.Fprintf(&body, "\tselect {\n")
		if hasResults {
			fmt.Fprintf(&body, "\tcase _r := <-_ch:\n")
			var zeroRets []string
			var retNames []string
			for _, r := range e.asyncBlockResults {
				goName := strings.ToUpper(r.name[:1]) + r.name[1:]
				retNames = append(retNames, "_r."+goName)
				zeroRets = append(zeroRets, buildZeroValue(r.goType))
			}
			if e.hasBlockError {
				retNames = append(retNames, "_r.Err")
				zeroRets = append(zeroRets, "ctx.Err()")
			}
			fmt.Fprintf(&body, "\t\treturn %s\n", strings.Join(retNames, ", "))
			fmt.Fprintf(&body, "\tcase <-ctx.Done():\n")
			fmt.Fprintf(&body, "\t\treturn %s\n", strings.Join(zeroRets, ", "))
		} else {
			fmt.Fprintf(&body, "\tcase <-_ch:\n")
			fmt.Fprintf(&body, "\tcase <-ctx.Done():\n")
		}
		fmt.Fprintf(&body, "\t}\n}\n\n")
	}

	if body.Len() == 0 {
		return nil
	}

	recordSpecialImports(body.Bytes(), usedImports)
	writeErgonomicHeader(w, pkgName, rawImportPath, usedImports, needsObjc)
	_, err := w.Write(body.Bytes())
	return err
}

// parseAsyncFromFuncType extracts non-block method args and async result params from
// the resolved Go func type string (e.g. "func(*NSData, *NSURL, error)").
// This is more reliable than parsing ObjC type strings directly.
func parseAsyncFromFuncType(goFuncType string, nonBlockMeta []meta.Param, ctx typemap.Context, m *typemap.Mapper, framework *meta.FrameworkMeta, imports typemap.ImportSet) (nonBlock []argInfo, results []asyncBlockResult, hasErr bool) {
	for i, arg := range nonBlockMeta {
		goType := resolveOpinionatedArgType(arg.ObjCType, ctx, m, framework, imports)
		argName := naming.ParamName(arg.Name)
		if argName == "" {
			argName = fmt.Sprintf("arg%d", i)
		}
		nonBlock = append(nonBlock, argInfo{name: argName, goType: goType})
	}

	params := extractFuncParams(goFuncType)
	for i, p := range params {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "error" {
			hasErr = true
			continue
		}
		if p == "*bool" || p == "bool" {
			continue
		}
		results = append(results, asyncBlockResult{name: fmt.Sprintf("v%d", i), goType: qualifyWithRawPackage(p, framework)})
	}
	return
}

// extractBlockParams returns the comma-separated param types inside a block type string,
// e.g. "void (^)(NSData *, NSError *)" → "NSData *, NSError *".
func extractBlockParams(blockType string) string {
	_, after, ok := strings.Cut(blockType, "(^)(")
	if !ok {
		return ""
	}
	end := strings.LastIndex(after, ")")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(after[:end])
}
