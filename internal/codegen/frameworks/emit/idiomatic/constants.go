//go:build darwin

package idiomatic

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// emitConstants writes <pkgname>_constants_generated.go: one idiomatic accessor
// per CoreFoundation-reference extern (const CF<Type>Ref globals such as
// kSecClass). The raw layer emits these as `func KSecClass() uintptr` returning
// the symbol *address*; using them means dereferencing the address to the
// reference value by hand. The idiomatic accessor returns the dereferenced value
// typed as an objc.ID (the reference and an objc.ID are the same pointer), so
// callers can pass it straight into a dictionary or a C function:
//
//	func KSecClass() objc.ID { return purego.CFConstant(raw.KSecClass()) }
//
// Only externs the raw layer actually emitted are wrapped (the raw skip rules are
// mirrored here) so the generated body always references a real raw accessor.
func emitConstants(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fw *meta.FrameworkMeta,
	m *typemap.Mapper,
	handFuncs, takenNames map[string]bool,
) error {
	ctx := typemap.Context{Framework: fw.Framework}
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
	view := cfConstantsView{RawAlias: rawPkgAlias}

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

		// NSString externs (NSURLResourceKey / notification-name globals) are
		// surfaced only in the foundation package itself, where the *String
		// wrapper and StringFromID are local; cross-package the idiomatic
		// foundation is not imported (generated code aliases the RAW foundation
		// as `foundation`), so they stay raw-only elsewhere.
		isString := pkgName == foundationPkgName && externIsNSString(ext, m, ctx)
		if !isCFRefExtern(ext.ObjCType) && !isString {
			continue
		}
		if takenNames[goName] || handFuncs[goName] {
			continue
		}
		takenNames[goName] = true

		comment := ""
		if ext.Doc != "" {
			comment = "// " + ext.Doc + "\n"
		}
		item := constantItem{
			GoName:       goName,
			ExternName:   ext.Name,
			CommentBlock: comment,
		}
		if isCFRefExtern(ext.ObjCType) {
			view.Items = append(view.Items, item)
		} else {
			view.StringItems = append(view.StringItems, item)
		}
	}

	if len(view.Items) == 0 && len(view.StringItems) == 0 {
		return nil
	}

	var body bytes.Buffer
	if err := executeTemplate(&body, "cf_constants", view); err != nil {
		return err
	}

	imports := map[string]string{
		rawPkgAlias: rawPkgPath,
		"purego":    pureobjcImportPath,
		"objc":      objcImportPath,
	}

	fname := pkgName + "_constants_generated.go"
	file := assembleFile(pkgName, usedImports(body.Bytes(), imports), body.Bytes())
	return emit.WriteGoFile(filepath.Join(outDir, fname), file)
}

// constantItem / cfConstantsView are the template data for constants.tmpl.
type constantItem struct {
	GoName       string
	ExternName   string
	CommentBlock string // "// <doc>\n" when the extern has a doc, else ""
}

type cfConstantsView struct {
	RawAlias string
	Items    []constantItem // CoreFoundation-reference externs → objc.ID
	// StringItems are NSString externs → *String (foundation package only).
	StringItems []constantItem
}

// foundationPkgName is the idiomatic foundation package name, used to decide
// whether the local *String wrapper is referenced bare or as foundation.String.
const foundationPkgName = "foundation"

// externIsNSString reports whether an extern resolves to an NSString * constant
// — directly or through a typedef such as NSURLResourceKey / NSNotificationName
// (e.g. NSURLVolumeAvailableCapacityKey is `const NSURLResourceKey`, which the
// mapper resolves to *NSString). The Go type is resolved the same way the raw
// externs emitter does (ext.GoType, else the mapper). Pointer-to-pointer
// out-parameters (containing "**") are excluded.
func externIsNSString(ext meta.Extern, m *typemap.Mapper, ctx typemap.Context) bool {
	goType := ext.GoType
	if goType == "" {
		goType = m.GoType(ext.ObjCType, ctx, make(typemap.ImportSet))
	}
	if strings.Contains(goType, "**") || !strings.Contains(goType, "*") {
		return false
	}
	return strings.Contains(goType, "NSString")
}

// isCFRefExtern reports whether an extern's ObjC type is a CoreFoundation
// reference value (const CF<Type>Ref) whose dereferenced value can be used as an
// objc.ID. Pointer-to-ref out-parameters (containing '*') are excluded.
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
