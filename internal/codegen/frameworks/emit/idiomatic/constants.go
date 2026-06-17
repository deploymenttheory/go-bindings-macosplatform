//go:build darwin

package idiomatic

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// emitConstants writes <pkgname>_constants_generated.go: one idiomatic accessor
// per CoreFoundation-reference extern (const CF<Type>Ref globals such as
// kSecClass). The raw layer emits these as `func KSecClass() uintptr` returning
// the symbol *address*; using them means dereferencing the address to the
// CFTypeRef value by hand. The idiomatic accessor returns the dereferenced value
// typed as an objc.ID (CoreFoundation is toll-free bridged), so callers can pass
// it straight into a dictionary or a C function via the runtime CF helpers:
//
//	func KSecClass() objc.ID { return purego.CFConstant(raw.KSecClass()) }
//
// Only externs the raw layer actually emitted are wrapped (the raw skip rules are
// mirrored here) so the generated body always references a real raw accessor.
func emitConstants(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fw *meta.FrameworkMeta,
	handFuncs, takenNames map[string]bool,
) error {
	// Mirror EmitExterns' name reservations: an extern is skipped when its C name
	// collides with a function (raw or exported Go name), or its Go accessor name
	// duplicates another extern's.
	funcNames := make(map[string]bool, len(fw.Functions)*2)
	for _, fn := range fw.Functions {
		funcNames[fn.Name] = true
		if goName := naming.ExportedFunctionName(fn.Name); goName != "" {
			funcNames[goName] = true
		}
	}

	seen := make(map[string]bool)
	seenGoNames := make(map[string]bool)
	var body bytes.Buffer

	for _, ext := range fw.Externs {
		if ext.Availability.IsUnavailable || seen[ext.Name] || funcNames[ext.Name] {
			continue
		}
		goName := externGoName(ext.Name)
		if goName == "" || seenGoNames[goName] || funcNames[goName] {
			continue
		}
		seen[ext.Name] = true
		seenGoNames[goName] = true

		if !isCFRefExtern(ext.ObjCType) {
			continue
		}
		if takenNames[goName] || handFuncs[goName] {
			continue
		}
		takenNames[goName] = true

		if ext.Doc != "" {
			fmt.Fprintf(&body, "// %s\n", ext.Doc)
		}
		fmt.Fprintf(
			&body,
			"// %s returns the CoreFoundation constant %s as a toll-free-bridged objc.ID.\n",
			goName, ext.Name,
		)
		fmt.Fprintf(
			&body,
			"func %s() objc.ID { return purego.CFConstant(%s.%s()) }\n\n",
			goName, rawPkgAlias, goName,
		)
	}

	if body.Len() == 0 {
		return nil
	}

	imports := map[string]string{
		rawPkgAlias: rawPkgPath,
		"purego":    pureobjcImportPath,
		"objc":      objcImportPath,
	}

	var buf bytes.Buffer
	fmt.Fprint(&buf, generatedHeader+"\n")
	fmt.Fprint(&buf, buildTag+"\n")
	fmt.Fprintf(&buf, "package %s\n\n", pkgName)
	writeImportBlock(&buf, usedImports(body.Bytes(), imports))
	buf.Write(body.Bytes())

	fname := pkgName + "_constants_generated.go"
	if err := os.WriteFile(filepath.Join(outDir, fname), buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", fname, err)
	}
	return nil
}

// isCFRefExtern reports whether an extern's ObjC type is a CoreFoundation
// reference value (const CF<Type>Ref) — the toll-free-bridged constants that can
// be dereferenced to an objc.ID. Pointer-to-ref out-parameters (containing '*')
// are excluded.
func isCFRefExtern(objcType string) bool {
	if strings.Contains(objcType, "*") {
		return false
	}
	for _, tok := range strings.Fields(objcType) {
		if strings.HasPrefix(tok, "CF") && strings.HasSuffix(tok, "Ref") && len(tok) > 4 {
			return true
		}
	}
	return false
}

// externGoName maps an extern symbol to its exported Go accessor name, matching
// exportedExternName in the raw externs emitter (trim leading underscores,
// upper-case the first letter).
func externGoName(symbol string) string {
	trimmed := strings.TrimLeft(symbol, "_")
	if trimmed == "" {
		return ""
	}
	return strings.ToUpper(trimmed[:1]) + trimmed[1:]
}
