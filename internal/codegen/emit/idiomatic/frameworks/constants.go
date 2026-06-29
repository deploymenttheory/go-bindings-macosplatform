//go:build darwin

package idiomatic

import (
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/idiomatic/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/idiomatic/view"
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
	framework *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	handFuncs, takenNames map[string]bool,
	trialNames trialNameMap,
) error {
	ctx := typemap.Context{Framework: framework.Framework}
	// Mirror EmitExterns' name reservations: an extern is skipped when its C name
	// collides with a function (raw or exported Go name), or its Go accessor name
	// duplicates another extern's.
	funcNames := make(map[string]bool, len(framework.Functions)*2)
	for _, fn := range framework.Functions {
		funcNames[fn.Name] = true
		if goName := naming.ExportedFunctionName(fn.Name); goName != "" {
			funcNames[goName] = true
		}
	}

	seen := make(map[string]bool)
	seenGoNames := make(map[string]bool)
	// Constants are grouped to preserve the original emission order: all
	// CoreFoundation references first, then Foundation *String constants, then
	// non-Foundation objc.ID string constants, then same-framework ObjC class
	// pointer globals.
	var cfRefs, strItems, strIDs, classItems []view.Constant

	for _, ext := range framework.Externs {
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
		// surfaced in the foundation package as the local idiomatic *String
		// wrapper. In every OTHER framework they are surfaced as objc.ID
		// accessors instead (the *String wrapper and StringFromID are foundation
		// -local and not imported cross-package), which is exactly what callers
		// need for an NSAboutPanelOptionKey / NSPasteboardType dictionary key.
		isFoundationString := pkgName == foundationPkgName && externIsNSString(ext, mapper, ctx)
		isObjcIDString, typedString := false, false
		isObjcClass, objcIdiomaticType := false, ""
		if pkgName != foundationPkgName {
			rawGoType := ext.GoType
			if rawGoType == "" {
				rawGoType = mapper.GoType(ext.ObjCType, ctx, make(typemap.ImportSet))
			}
			switch {
			case strings.Contains(rawGoType, "NSString") && !strings.Contains(rawGoType, "**"):
				// Raw emitted a typed *NSString accessor (already dereferenced);
				// take its objc.ID via .Ptr().
				isObjcIDString, typedString = true, true
			case (rawGoType == "" || rawGoType == "unsafe.Pointer") && externNSStringViaTypedef(ext, framework):
				// Raw could not resolve the type (e.g. `API_AVAILABLE const
				// NSAboutPanelOptionKey`) so it emitted a uintptr symbol-address
				// accessor; CFConstant dereferences it to the NSString objc.ID.
				isObjcIDString = true
			default:
				// Same-framework ObjC class pointer: *ClassName with no package
				// qualifier and no generics. Look up the idiomatic trial name so
				// the accessor returns the wrapper type, not a raw pointer.
				if after, ok := strings.CutPrefix(rawGoType, "*"); ok &&
					!strings.Contains(after, "*") &&
					!strings.Contains(after, ".") &&
					!strings.Contains(after, "[") {
					if idiomaticName, found := trialNames[after]; found {
						isObjcClass = true
						objcIdiomaticType = idiomaticName
					}
				}
			}
		}
		if !isCFRefExtern(ext.ObjCType) && !isFoundationString && !isObjcIDString && !isObjcClass {
			continue
		}
		if takenNames[goName] || handFuncs[goName] {
			continue
		}
		takenNames[goName] = true

		_ = typedString // current accessors do not vary by typed-ness; reserved
		comment := ""
		if doc := cleanDoc(ext.Doc); doc != "" {
			comment = "// " + strings.ReplaceAll(doc, "\n", "\n// ") + "\n"
		}
		switch {
		case isCFRefExtern(ext.ObjCType):
			cfRefs = append(
				cfRefs,
				view.Constant{
					GoName:       goName,
					ExternName:   ext.Name,
					CommentBlock: comment,
					Kind:         view.ConstCFRef,
				},
			)
		case isFoundationString:
			strItems = append(
				strItems,
				view.Constant{
					GoName:       goName,
					ExternName:   ext.Name,
					CommentBlock: comment,
					Kind:         view.ConstNSString,
				},
			)
		case isObjcClass:
			classItems = append(
				classItems,
				view.Constant{
					GoName:       goName,
					ExternName:   ext.Name,
					CommentBlock: comment,
					Kind:         view.ConstObjcClass,
					GoTypeName:   objcIdiomaticType,
				},
			)
		default: // isObjcIDString
			strIDs = append(
				strIDs,
				view.Constant{
					GoName:       goName,
					ExternName:   ext.Name,
					CommentBlock: comment,
					Kind:         view.ConstObjcID,
				},
			)
		}
	}

	constants := make([]view.Constant, 0, len(cfRefs)+len(strItems)+len(strIDs)+len(classItems))
	constants = append(constants, cfRefs...)
	constants = append(constants, strItems...)
	constants = append(constants, strIDs...)
	constants = append(constants, classItems...)
	if len(constants) == 0 {
		return nil
	}

	body, err := render.Constants(constants)
	if err != nil {
		return err
	}

	_ = rawPkgPath
	_ = rawPkgAlias
	// Imports computed from the resolved constants, not by scanning the rendered
	// text: every accessor calls purego.CFConstant; the obj wrapper is used only
	// by the CF-reference and objc.ID-string accessors (the *String accessor uses
	// StringFromID instead). ObjC class pointer accessors additionally need objc
	// and unsafe to dereference the symbol address.
	imports := map[string]string{"purego": pureobjcImportPath}
	if len(cfRefs) > 0 || len(strIDs) > 0 {
		imports["obj"] = objImportPath
	}
	if len(classItems) > 0 {
		imports["objc"] = objcImportPath
		imports["unsafe"] = "unsafe"
	}

	fname := pkgName + "_constants_generated.go"
	file := assembleFile(pkgName, imports, body)
	return emit.WriteGoFile(filepath.Join(outDir, fname), file)
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
func externIsNSString(ext meta.Extern, mapper *typemap.Mapper, ctx typemap.Context) bool {
	goType := ext.GoType
	if goType == "" {
		goType = mapper.GoType(ext.ObjCType, ctx, make(typemap.ImportSet))
	}
	if strings.Contains(goType, "**") || !strings.Contains(goType, "*") {
		return false
	}
	return strings.Contains(goType, "NSString")
}

// externNSStringViaTypedef reports whether an extern's ObjC type resolves to
// NSString through a framework typedef chain (e.g. `API_AVAILABLE const
// NSAboutPanelOptionKey`, where NSAboutPanelOptionKey is typedef'd to
// `NSString *`). It is used for externs the mapper could not type, which the raw
// layer emitted as bare uintptr symbol-address accessors.
func externNSStringViaTypedef(ext meta.Extern, framework *meta.FrameworkMeta) bool {
	for _, tok := range strings.Fields(ext.ObjCType) {
		if resolvesToNSString(strings.TrimSuffix(tok, "*"), framework, map[string]bool{}) {
			return true
		}
	}
	return false
}

// resolvesToNSString walks framework.Typedefs from name, reporting whether it reaches
// the NSString class. seen guards against typedef cycles.
func resolvesToNSString(name string, framework *meta.FrameworkMeta, seen map[string]bool) bool {
	name = strings.TrimSpace(name)
	if name == "" || seen[name] {
		return false
	}
	seen[name] = true
	if name == "NSString" {
		return true
	}
	next, ok := framework.Typedefs[name]
	if !ok {
		return false
	}
	next = strings.TrimPrefix(strings.TrimSpace(next), "const ")
	for _, tok := range strings.Fields(next) {
		if resolvesToNSString(strings.TrimSuffix(tok, "*"), framework, seen) {
			return true
		}
	}
	return false
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
