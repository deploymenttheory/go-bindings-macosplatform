//go:build darwin

package idiomatic

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/emit"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/typemap"
)

// emitGenericFunctionWrappers writes <pkgname>_cfunctions_generated.go with
// one exported forwarding wrapper per emittable raw C function, giving
// C-function-only frameworks (e.g. vmnet) a usable idiomatic surface:
//
//   - The framework-name prefix is stripped when present
//     (raw.VmnetStartInterface → vmnet.StartInterface).
//   - Parameter and return types mirror the raw signature exactly (block
//     params surface as the same Go closure types the raw wrappers take).
//   - Functions already wrapped by the CFError pass are skipped.
//
// Name collisions fall back to the full exported name; if that is also taken
// the function is skipped with a diagnostic. Iteration is in sorted order so
// the outcome is deterministic.
func emitGenericFunctionWrappers(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fw *meta.FrameworkMeta,
	m *typemap.Mapper,
	takenNames map[string]bool,
) error {
	ctx := typemap.Context{Framework: fw.Framework}
	prefix := naming.GoTypeName(strings.ToLower(fw.Framework))

	type wrapEntry struct {
		goName    string // wrapper name in the idiomatic package
		rawGoName string // exported name in the raw package
		params    []string
		callArgs  []string
		retType   string
	}

	var entries []wrapEntry
	var body bytes.Buffer

	for _, fn := range emit.EmittableFunctions(fw, nil) {
		// CFError out-param functions are wrapped (with error conversion) by
		// emitFunctionWrappers.
		if len(fn.Params) > 0 && isCFErrorOutParam(fn.Params[len(fn.Params)-1].ObjCType) {
			continue
		}

		rawGoName := naming.ExportedFunctionName(fn.Name)
		goName := rawGoName
		if stripped := strings.TrimPrefix(rawGoName, prefix); stripped != rawGoName &&
			stripped != "" && stripped[0] >= 'A' && stripped[0] <= 'Z' {
			goName = stripped
		}
		if takenNames[goName] {
			goName = rawGoName
		}
		if takenNames[goName] {
			m.AppendDiagnostic(
				"%s: idiomatic wrapper for %s skipped (name %s already taken)",
				fw.Framework, fn.Name, goName,
			)
			continue
		}
		takenNames[goName] = true

		// Mirror the raw emitter's signature exactly so the wrapper forwards
		// arguments unchanged.
		var params, callArgs []string
		usedNames := make(map[string]int)
		unexportedRef := false
		for _, param := range fn.Params {
			paramName := naming.ParamName(param.Name)
			usedNames[paramName]++
			if usedNames[paramName] > 1 {
				paramName = fmt.Sprintf("%s%d", paramName, usedNames[paramName])
			}
			impSet := make(typemap.ImportSet)
			goType := rawParamGoType(param.ObjCType, ctx, m, impSet)
			if isUnexportedXPkgType(goType) {
				goType = "unsafe.Pointer"
			}
			goType = qualifyRaw(goType, fw, rawPkgAlias, nil)
			if referencesUnexportedQualified(goType) {
				unexportedRef = true
			}
			params = append(params, paramName+" "+goType)
			callArgs = append(callArgs, paramName)
		}

		retType := ""
		if _, retIsBlock := m.ResolveBlockSignature(fn.Return.ObjCType); retIsBlock {
			retType = "objc.Block"
		} else if fn.Return.ObjCType != "void" && fn.Return.ObjCType != "" {
			impSet := make(typemap.ImportSet)
			retType = m.GoReturnType(fn.Return.ObjCType, ctx, impSet)
			if retType == "" || isUnexportedXPkgType(retType) {
				retType = "unsafe.Pointer"
			}
			retType = qualifyRaw(retType, fw, rawPkgAlias, nil)
			if referencesUnexportedQualified(retType) {
				unexportedRef = true
			}
		}

		// Some raw types stay unexported (underscore-prefixed enums like
		// _CGLError) — they cannot be referenced from another package, so the
		// function cannot be wrapped.
		if unexportedRef {
			delete(takenNames, goName)
			m.AppendDiagnostic(
				"%s: idiomatic wrapper for %s skipped (signature references an unexported raw type)",
				fw.Framework,
				fn.Name,
			)
			continue
		}

		entries = append(entries, wrapEntry{
			goName:    goName,
			rawGoName: rawGoName,
			params:    params,
			callArgs:  callArgs,
			retType:   retType,
		})
	}

	if len(entries) == 0 {
		return nil
	}

	for _, entry := range entries {
		fmt.Fprintf(
			&body,
			"// %s calls [%s.%s] (C function %s).\n",
			entry.goName, rawPkgAlias, entry.rawGoName, cFunctionNameFor(entry.rawGoName, fw),
		)
		retSig := ""
		if entry.retType != "" {
			retSig = " " + entry.retType
		}
		fmt.Fprintf(
			&body,
			"func %s(%s)%s {\n",
			entry.goName, strings.Join(entry.params, ", "), retSig,
		)
		call := fmt.Sprintf(
			"%s.%s(%s)",
			rawPkgAlias,
			entry.rawGoName,
			strings.Join(entry.callArgs, ", "),
		)
		if entry.retType != "" {
			fmt.Fprintf(&body, "\treturn %s\n", call)
		} else {
			fmt.Fprintf(&body, "\t%s\n", call)
		}
		fmt.Fprint(&body, "}\n\n")
	}

	// Render-then-scan imports: only include packages the body references.
	bodyStr := body.String()
	imports := map[string]string{rawPkgAlias: rawPkgPath}
	if strings.Contains(bodyStr, "unsafe.") {
		imports["unsafe"] = "unsafe"
	}
	if referencesPackage(bodyStr, "objc") {
		imports["objc"] = objcImportPath
	}
	if referencesPackage(bodyStr, "foundation") {
		imports["foundation"] = foundationImportPath
	}
	for _, crossPkg := range collectCrossPackageRefs(bodyStr, rawPkgAlias) {
		imports[crossPkg] = strings.TrimSuffix(
			rawPkgPath,
			naming.PackageName(fw.Framework),
		) + crossPkg
	}

	var buf bytes.Buffer
	fmt.Fprint(&buf, generatedHeader+"\n")
	fmt.Fprint(&buf, buildTag+"\n")
	fmt.Fprintf(&buf, "package %s\n\n", pkgName)
	writeImportBlock(&buf, imports)
	buf.Write(body.Bytes())

	fname := pkgName + "_cfunctions_generated.go"
	if err := os.WriteFile(filepath.Join(outDir, fname), buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", fname, err)
	}
	return nil
}

// cFunctionNameFor recovers the original C symbol for an exported Go function
// name by scanning the framework's function list. Used only for doc comments.
func cFunctionNameFor(rawGoName string, fw *meta.FrameworkMeta) string {
	for _, fn := range fw.Functions {
		if naming.ExportedFunctionName(fn.Name) == rawGoName {
			return fn.Name
		}
	}
	return rawGoName
}

// isUnexportedXPkgType reports whether a Go type string references an
// unexported identifier from another package (compile error if emitted).
func isUnexportedXPkgType(goType string) bool {
	trimmed := strings.TrimPrefix(goType, "*")
	dotIdx := strings.LastIndex(trimmed, ".")
	if dotIdx < 0 {
		return false
	}
	after := trimmed[dotIdx+1:]
	if brIdx := strings.Index(after, "["); brIdx >= 0 {
		after = after[:brIdx]
	}
	return len(after) > 0 && after[0] >= 'a' && after[0] <= 'z'
}

// referencesUnexportedQualified reports whether a (possibly qualified) Go
// type string contains a package-qualified unexported identifier such as
// raw._CGLError — a compile error when emitted outside the raw package.
func referencesUnexportedQualified(goType string) bool {
	for i := 1; i < len(goType)-1; i++ {
		if goType[i] != '.' || !isIdentByte(goType[i-1], false) {
			continue
		}
		c := goType[i+1]
		if c == '_' || (c >= 'a' && c <= 'z') {
			return true
		}
	}
	return false
}

// referencesPackage reports whether the body references pkg as a package
// qualifier ("pkg.") at a non-identifier boundary, avoiding false hits on
// names like "pureobjc." when checking for "objc.".
func referencesPackage(body, pkg string) bool {
	needle := pkg + "."
	for idx := strings.Index(body, needle); idx >= 0; {
		if idx == 0 || !isIdentByte(body[idx-1], false) {
			return true
		}
		next := strings.Index(body[idx+1:], needle)
		if next < 0 {
			return false
		}
		idx += 1 + next
	}
	return false
}

// collectCrossPackageRefs finds cross-framework package qualifiers used in the
// body (lowercase identifiers followed by '.' and an exported identifier),
// excluding the raw alias and well-known packages handled explicitly.
func collectCrossPackageRefs(body, rawPkgAlias string) []string {
	known := map[string]bool{
		rawPkgAlias: true, "unsafe": true, "objc": true, "foundation": true,
		"fmt": true, "strings": true, "pureobjc": true,
	}
	var refs []string
	seen := map[string]bool{}
	for i := 0; i < len(body); {
		if !isIdentByte(body[i], true) {
			i++
			continue
		}
		j := i + 1
		for j < len(body) && isIdentByte(body[j], false) {
			j++
		}
		word := body[i:j]
		prevIsDot := i > 0 && body[i-1] == '.'
		if !prevIsDot && j < len(body)-1 && body[j] == '.' &&
			word == strings.ToLower(word) && !known[word] && !seen[word] &&
			body[j+1] >= 'A' && body[j+1] <= 'Z' {
			seen[word] = true
			refs = append(refs, word)
		}
		i = j
	}
	return refs
}
