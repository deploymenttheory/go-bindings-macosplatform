package library

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/emit/raw"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// return classifications for an idiomatic wrapper.
const (
	retVoid   = iota // no return
	retPlain         // pass the raw return type through
	retError         // OSStatus / kern_return_t → error
	retHandle        // opaque handle → wrapper type
)

// reservedParamNames are package aliases the generated wrappers reference; a C
// parameter resolving to one of these would shadow the import.
var reservedParamNames = map[string]bool{"raw": true, "fmt": true, "unsafe": true}

// goBuiltins are the predeclared Go type names that must never be qualified with
// the raw package alias.
var goBuiltins = map[string]bool{
	"bool": true, "string": true, "error": true, "any": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"uintptr": true, "byte": true, "rune": true,
	"float32": true, "float64": true, "complex64": true, "complex128": true,
}

// EmitCFunctions writes the idiomatic wrapper layer for a C library's plain
// functions into <pkgName>_generated.go, transforming the already-clean raw
// bindings:
//
//   - Handles-as-methods: opaque C handle typedefs (typedefs ending in _t that
//     resolve to unsafe.Pointer and are neither enums, structs, nor
//     function-pointer typedefs) become Go wrapper structs; a function whose
//     first argument is such a handle becomes a method on that wrapper.
//   - Prefix stripping: the common C symbol prefix (es_, xpc_, dispatch_, …) is
//     dropped and the remainder PascalCased, so es_message_size → MessageSize.
//   - Error conversion: OSStatus / kern_return_t returns become a Go error.
//
// Functions with block / function-pointer arguments, or whose raw Go symbol is
// unexported (leading-underscore C symbols), are skipped — the raw binding
// remains available for those.
func EmitCFunctions(w io.Writer, pkgName, rawImportPath string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) error {
	ctx := m.BaseContext(framework.Framework, knownClasses)
	prefix := detectSymbolPrefix(framework)

	// Eligible functions: exported raw symbol, no block/function-pointer args.
	type efn struct {
		fn        macosplatformmetadata.Function
		rawGoName string
	}
	var eligible []efn
	for _, fn := range raw.EmittableFunctions(framework) {
		rawGoName := naming.GoTypeName(fn.Name)
		if !isExportedName(rawGoName) || fnHasBlockArg(fn, m) {
			continue
		}
		eligible = append(eligible, efn{fn: fn, rawGoName: rawGoName})
	}

	// Every opaque handle referenced anywhere (params or returns) gets a wrapper
	// struct, so a type reference never dangles.
	handleUsed := map[string]bool{}
	for _, e := range eligible {
		for _, p := range e.fn.Params {
			if base, ok := handleBaseName(p.ObjCType, ctx, m, framework); ok {
				handleUsed[base] = true
			}
		}
		if base, ok := handleBaseName(e.fn.Return.ObjCType, ctx, m, framework); ok {
			handleUsed[base] = true
		}
	}
	goToBase := map[string]string{} // wrapper Go name → objc base (deduped)
	for base := range handleUsed {
		goToBase[handleGoName(base, prefix)] = base
	}

	usedImports := map[string]string{}
	methodsByGo := map[string]*bytes.Buffer{}
	seenByGo := map[string]map[string]bool{}
	var freeFns bytes.Buffer
	freeSeen := map[string]bool{}

	for _, e := range eligible {
		params := e.fn.Params
		var recvGo, recvBase string
		if len(params) > 0 {
			if base, ok := handleBaseName(params[0].ObjCType, ctx, m, framework); ok && handleUsed[base] {
				recvBase, recvGo = base, handleGoName(base, prefix)
				params = params[1:]
			}
		}

		sig, callArgs, ok := buildIdiomaticParams(params, ctx, m, framework, handleUsed, prefix, usedImports)
		if !ok {
			continue
		}
		retKind, retType := classifyReturn(e.fn.Return.ObjCType, ctx, m, framework, handleUsed, prefix, usedImports)

		if recvGo != "" {
			buf := methodsByGo[recvGo]
			if buf == nil {
				buf = &bytes.Buffer{}
				methodsByGo[recvGo] = buf
				seenByGo[recvGo] = map[string]bool{}
			}
			name := dedup(methodGoName(e.fn.Name, recvBase, prefix), seenByGo[recvGo])
			writeWrapperFunc(buf, "(h "+recvGo+") ", name, e.rawGoName,
				append([]string{"h.ptr"}, callArgs...), sig, retKind, retType)
		} else {
			name := dedup(freeFuncGoName(e.fn.Name, prefix), freeSeen)
			writeWrapperFunc(&freeFns, "", name, e.rawGoName, callArgs, sig, retKind, retType)
		}
	}

	if freeFns.Len() == 0 && len(goToBase) == 0 {
		return nil
	}

	var body bytes.Buffer
	goNames := make([]string, 0, len(goToBase))
	for name := range goToBase {
		goNames = append(goNames, name)
	}
	sort.Strings(goNames)
	for _, goName := range goNames {
		base := goToBase[goName]
		fmt.Fprintf(&body, "// %s wraps the C handle %s.\n", goName, base)
		fmt.Fprintf(&body, "type %s struct{ ptr unsafe.Pointer }\n\n", goName)
		fmt.Fprintf(&body, "// Wrap%s adopts an existing %s handle.\n", goName, base)
		fmt.Fprintf(&body, "func Wrap%s(p unsafe.Pointer) %s { return %s{ptr: p} }\n\n", goName, goName, goName)
		fmt.Fprintf(&body, "// Ptr returns the underlying %s handle.\n", base)
		fmt.Fprintf(&body, "func (h %s) Ptr() unsafe.Pointer { return h.ptr }\n\n", goName)
		if buf := methodsByGo[goName]; buf != nil {
			body.Write(buf.Bytes())
		}
	}
	body.Write(freeFns.Bytes())
	if body.Len() == 0 {
		return nil
	}

	extraImports := map[string]string{}
	if bytes.Contains(body.Bytes(), []byte("unsafe.")) {
		extraImports["unsafe"] = "unsafe"
	}
	if bytes.Contains(body.Bytes(), []byte("fmt.")) {
		extraImports["fmt"] = "fmt"
	}
	writeOpinionatedHeader(w, pkgName, rawImportPath, extraImports, usedImports, false)
	_, err := w.Write(body.Bytes())
	return err
}

// writeWrapperFunc emits one idiomatic function or method forwarding to the raw
// binding. recvPrefix is "" for a free function or "(h T) " for a method.
func writeWrapperFunc(w io.Writer, recvPrefix, goName, rawGoName string, callArgs, sig []string, retKind int, retType string) {
	call := fmt.Sprintf("raw.%s(%s)", rawGoName, strings.Join(callArgs, ", "))
	params := strings.Join(sig, ", ")
	switch retKind {
	case retVoid:
		fmt.Fprintf(w, "func %s%s(%s) {\n\t%s\n}\n\n", recvPrefix, goName, params, call)
	case retError:
		fmt.Fprintf(w, "func %s%s(%s) error {\n", recvPrefix, goName, params)
		fmt.Fprintf(w, "\tif _rc := %s; _rc != 0 {\n", call)
		fmt.Fprintf(w, "\t\treturn fmt.Errorf(%q+\": status %%d\", _rc)\n", goName)
		fmt.Fprintf(w, "\t}\n\treturn nil\n}\n\n")
	case retHandle:
		fmt.Fprintf(w, "func %s%s(%s) %s {\n\treturn Wrap%s(%s)\n}\n\n", recvPrefix, goName, params, retType, retType, call)
	default: // retPlain
		fmt.Fprintf(w, "func %s%s(%s) %s {\n\treturn %s\n}\n\n", recvPrefix, goName, params, retType, call)
	}
}

// buildIdiomaticParams maps the (non-receiver) parameters to idiomatic Go types
// and the matching raw-call argument expressions. Handle parameters become
// wrapper types (passed as name.ptr); everything else keeps its raw Go type,
// qualified with the raw package alias. ok=false if any parameter is a
// block/function pointer or unresolvable.
func buildIdiomaticParams(params []macosplatformmetadata.Param, ctx typemap.Context, m *typemap.Mapper, framework *macosplatformmetadata.FrameworkMeta, handleUsed map[string]bool, prefix string, usedImports map[string]string) (sig, callArgs []string, ok bool) {
	seen := map[string]int{}
	for i, p := range params {
		name := naming.ParamName(p.Name)
		if name == "" || reservedParamNames[name] {
			name = fmt.Sprintf("arg%d", i)
		}
		seen[name]++
		if seen[name] > 1 {
			name = fmt.Sprintf("%s%d", name, seen[name])
		}
		if base, isHandle := handleBaseName(p.ObjCType, ctx, m, framework); isHandle && handleUsed[base] {
			sig = append(sig, name+" "+handleGoName(base, prefix))
			callArgs = append(callArgs, name+".ptr")
			continue
		}
		throwaway := make(typemap.ImportSet)
		goType := m.GoType(p.ObjCType, ctx, throwaway)
		if goType == "" || strings.Contains(goType, "func(") {
			return nil, nil, false
		}
		goType = qualifyRawTokens(goType)
		recordOpinionatedImports(goType, m, usedImports)
		sig = append(sig, name+" "+goType)
		callArgs = append(callArgs, name)
	}
	return sig, callArgs, true
}

// classifyReturn determines the idiomatic return shape for a function's ObjC
// return type.
func classifyReturn(objcRet string, ctx typemap.Context, m *typemap.Mapper, framework *macosplatformmetadata.FrameworkMeta, handleUsed map[string]bool, prefix string, usedImports map[string]string) (kind int, goType string) {
	base := cleanTypeToken(objcRet)
	if objcRet == "" || (base == "void" && !strings.Contains(objcRet, "*")) {
		return retVoid, ""
	}
	if base == "OSStatus" || base == "kern_return_t" {
		return retError, "error"
	}
	if hb, ok := handleBaseName(objcRet, ctx, m, framework); ok && handleUsed[hb] {
		return retHandle, handleGoName(hb, prefix)
	}
	throwaway := make(typemap.ImportSet)
	gt := m.GoReturnType(objcRet, ctx, throwaway)
	if gt == "" || strings.Contains(gt, "func(") {
		return retVoid, ""
	}
	gt = qualifyRawTokens(gt)
	recordOpinionatedImports(gt, m, usedImports)
	return retPlain, gt
}

// handleBaseName reports whether an ObjC type is an opaque C handle: a typedef
// whose name ends in _t, which the mapper resolves to unsafe.Pointer, and which
// is not a named enum, struct, or function-pointer/block typedef. Returns the
// bare typedef name (e.g. "xpc_object_t").
func handleBaseName(objcType string, ctx typemap.Context, m *typemap.Mapper, framework *macosplatformmetadata.FrameworkMeta) (string, bool) {
	base := cleanTypeToken(objcType)
	if !strings.HasSuffix(base, "_t") {
		return "", false
	}
	if _, isEnum := framework.Enums[base]; isEnum {
		return "", false
	}
	if _, isStruct := framework.Structs[base]; isStruct {
		return "", false
	}
	// Function-pointer / block typedefs (target contains '(') are not handles.
	if target, ok := m.TypedefIndex[base]; ok && strings.Contains(target, "(") {
		return "", false
	}
	throwaway := make(typemap.ImportSet)
	if m.GoType(objcType, ctx, throwaway) != "unsafe.Pointer" {
		return "", false
	}
	return base, true
}

// qualifyRawTokens prefixes every bare identifier in goType that is neither a Go
// builtin nor a package qualifier with "raw." — because any non-builtin bare
// type name the mapper returns is a type the raw package defines (raw and the
// idiomatic layer share the same type mapper). Identifiers preceded or followed
// by '.' (package qualifiers, e.g. bsd.Timespec, unsafe.Pointer) are left alone.
func qualifyRawTokens(goType string) string {
	var b strings.Builder
	for i := 0; i < len(goType); {
		c := goType[i]
		if isIdentStart(c) {
			j := i + 1
			for j < len(goType) && isIdentPart(goType[j]) {
				j++
			}
			ident := goType[i:j]
			prevDot := i > 0 && goType[i-1] == '.'
			nextDot := j < len(goType) && goType[j] == '.'
			if !prevDot && !nextDot && !goBuiltins[ident] {
				b.WriteString("raw.")
			}
			b.WriteString(ident)
			i = j
		} else {
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func isExportedName(name string) bool {
	return name != "" && name[0] >= 'A' && name[0] <= 'Z'
}

// handleGoName turns an opaque-handle typedef name into an exported Go wrapper
// type name: strip the library prefix and the _t suffix, then PascalCase.
// xpc_connection_t (prefix "xpc") → "Connection"; es_client_t → "Client".
func handleGoName(objcBase, prefix string) string {
	s := strings.TrimSuffix(objcBase, "_t")
	if prefix != "" {
		s = strings.TrimPrefix(s, prefix+"_")
	}
	name := snakeToPascal(s)
	if name == "" {
		name = snakeToPascal(strings.TrimSuffix(objcBase, "_t"))
	}
	return name
}

// methodGoName strips the receiver handle's prefix from a C symbol and
// PascalCases the remainder. xpc_connection_send_message on xpc_connection_t →
// "SendMessage". Falls back to the library-prefix-stripped name.
func methodGoName(cName, objcBase, prefix string) string {
	handlePrefix := strings.TrimSuffix(objcBase, "_t") + "_" // e.g. "xpc_connection_"
	if rest := strings.TrimPrefix(cName, handlePrefix); rest != cName && rest != "" {
		return snakeToPascal(rest)
	}
	return freeFuncGoName(cName, prefix)
}

// freeFuncGoName strips the library prefix from a C symbol and PascalCases it.
func freeFuncGoName(cName, prefix string) string {
	s := cName
	if prefix != "" {
		s = strings.TrimPrefix(s, prefix+"_")
	}
	name := snakeToPascal(s)
	if name == "" {
		name = naming.GoTypeName(cName)
	}
	return name
}

// detectSymbolPrefix returns the most common leading token (before the first
// underscore) across the library's emittable function names.
func detectSymbolPrefix(framework *macosplatformmetadata.FrameworkMeta) string {
	counts := map[string]int{}
	for _, fn := range raw.EmittableFunctions(framework) {
		if i := strings.IndexByte(fn.Name, '_'); i > 0 {
			counts[fn.Name[:i]]++
		}
	}
	best, bestN := "", 0
	for tok, n := range counts {
		if n > bestN {
			best, bestN = tok, n
		}
	}
	return best
}

// fnHasBlockArg reports whether any parameter is a block / function pointer.
func fnHasBlockArg(fn macosplatformmetadata.Function, m *typemap.Mapper) bool {
	for _, p := range fn.Params {
		if p.IsBlock {
			return true
		}
		if target, ok := m.TypedefIndex[typemap.Normalise(p.ObjCType)]; ok && typemap.IsBlock(target) {
			return true
		}
	}
	return false
}

// cleanTypeToken extracts the bare type name from an ObjC type string, stripping
// pointers, generics, qualifiers, nullability, and attribute macros (e.g.
// DISPATCH_RETURNS_RETAINED / XPC_RETURNS_RETAINED). Returns the last remaining
// type token, e.g. "DISPATCH_RETURNS_RETAINED dispatch_data_t  _Nullable" →
// "dispatch_data_t" and "void *" → "void".
func cleanTypeToken(objcType string) string {
	s := objcType
	if i := strings.IndexByte(s, '<'); i >= 0 {
		s = s[:i]
	}
	s = strings.ReplaceAll(s, "*", " ")
	var last string
	for f := range strings.FieldsSeq(s) {
		switch f {
		case "const", "__kindof", "volatile", "struct", "union", "enum",
			"_Nullable", "_Nonnull", "_Null_unspecified":
			continue
		}
		if isAllCapsMacro(f) {
			continue
		}
		last = f
	}
	return last
}

// isAllCapsMacro reports whether s is an UPPER_SNAKE attribute macro (letters
// present, none lowercase) such as XPC_RETURNS_RETAINED — never a real C type
// name (OSStatus has lowercase; kern_return_t is lowercase).
func isAllCapsMacro(s string) bool {
	hasLetter := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return false
		}
		if r >= 'A' && r <= 'Z' {
			hasLetter = true
		}
	}
	return hasLetter
}

// snakeToPascal converts snake_case to PascalCase, dropping empty segments.
func snakeToPascal(s string) string {
	var b strings.Builder
	for part := range strings.SplitSeq(s, "_") {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		b.WriteString(part[1:])
	}
	return b.String()
}

// dedup ensures a name is unique within seen, appending 2, 3, … on collision.
func dedup(name string, seen map[string]bool) string {
	if name == "" {
		name = "Fn"
	}
	if !seen[name] {
		seen[name] = true
		return name
	}
	for i := 2; ; i++ {
		cand := fmt.Sprintf("%s%d", name, i)
		if !seen[cand] {
			seen[cand] = true
			return cand
		}
	}
}
