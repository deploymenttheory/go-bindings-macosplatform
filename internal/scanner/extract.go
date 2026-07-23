//go:build darwin

package scanner

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// sdkRelativePath converts an absolute SDK header path to a short relative form.
// "/Applications/Xcode.app/.../Foundation.framework/Headers/NSString.h"
// → "Foundation/NSString.h"
// Falls back to the basename if the path doesn't match the expected pattern.
func sdkRelativePath(sdkPath, absFile string) string {
	if absFile == "" {
		return ""
	}
	const fwPrefix = "System/Library/Frameworks/"
	rel := strings.TrimPrefix(absFile, sdkPath+"/")
	if after, found := strings.CutPrefix(rel, fwPrefix); found {
		rel = after
		// "Foundation.framework/Headers/NSString.h" → "Foundation/NSString.h"
		if before, after2, found2 := strings.Cut(rel, ".framework/Headers/"); found2 {
			rel = before + "/" + after2
		}
		return rel
	}
	// e.g. usr/include/objc/NSObject.h → keep as-is (already short)
	return filepath.Base(absFile)
}

// frameworkNameFromPath extracts the framework (leaf) name from an SDK header path.
// e.g. ".../CoreImage.framework/Headers/CIImage.h" → "CoreImage".
// Returns "" for non-framework paths (usr/include, etc.).
func frameworkNameFromPath(path string) string {
	idx := strings.Index(path, ".framework/")
	if idx < 0 {
		return ""
	}
	start := strings.LastIndex(path[:idx], "/")
	return path[start+1 : idx]
}

// trackDeclaredImport records framework-level #import relationships derived from
// the Clang AST's IncludedFrom chain. For each top-level AST node whose file
// belongs to a different framework, if that file was directly included by one of
// the current framework's own headers, we know the current framework explicitly
// imports the other framework. This information is used by the cycle-breaker in
// the code generator to distinguish intentional cross-framework edges from
// incidental ones discovered only via type-token scanning.
func trackDeclaredImport(node *ASTNode, framework *macosplatformmetadata.FrameworkMeta, f *frameworkFilter) {
	if node.Loc == nil {
		return
	}
	nodeFile := node.Loc.ResolvedFile()
	if nodeFile == "" {
		return
	}
	// Node must be from a framework header (not usr/include etc.)
	otherFw := frameworkNameFromPath(nodeFile)
	if otherFw == "" || otherFw == framework.Framework {
		return
	}
	// The node must not be from the current framework's own headers.
	if strings.HasPrefix(nodeFile, f.headerDir) {
		return
	}
	// The direct includer must be from the current framework's own headers.
	includedFrom := node.Loc.IncludedFromFile()
	if includedFrom == "" {
		return
	}
	if !strings.HasPrefix(includedFrom, f.headerDir) {
		return
	}
	if framework.DeclaredImports == nil {
		framework.DeclaredImports = make(map[string]bool)
	}
	framework.DeclaredImports[otherFw] = true
}

// nodeFile returns the effective header file path for a node, using the filter's
// rolling current-file as a fallback when the node's loc.file is empty.
func nodeFile(node *ASTNode, fallback string) string {
	if node.Loc == nil {
		return fallback
	}
	f := node.Loc.ResolvedFile()
	if f == "" {
		return fallback
	}
	return f
}

// nodeFileLine returns the (file, line) pair for a node using the filter's
// current file as a fallback for the file path.
func nodeFileLine(node *ASTNode, fallbackFile string) (string, int) {
	file := nodeFile(node, fallbackFile)
	if node.Loc == nil {
		return file, 0
	}
	line := node.Loc.ResolvedLine()
	return file, line
}

// Extract walks the Clang AST root node and produces a FrameworkMeta
// containing only declarations that originate from the named framework's headers.
func Extract(root *ASTNode, sdkPath, frameworkName, sdkVersion, arch string) *macosplatformmetadata.FrameworkMeta {
	filter := newFilter(sdkPath, frameworkName)
	f := &filter

	// Use the leaf name (strip "Parent/" prefix for sub-frameworks) as the
	// canonical framework name stored in metadata.
	canonicalName := frameworkName
	var parentFramework string
	if IsSubFramework(frameworkName) {
		parentFramework, canonicalName = SubFrameworkParts(frameworkName)
	}

	framework := &macosplatformmetadata.FrameworkMeta{
		Framework:       canonicalName,
		ParentFramework: parentFramework,
		SDKVersion:      sdkVersion,
		Arch:            arch,
		Classes:    make(map[string]macosplatformmetadata.Class),
		Protocols:  make(map[string]macosplatformmetadata.Protocol),
		Enums:      make(map[string]macosplatformmetadata.Enum),
		Structs:    make(map[string]macosplatformmetadata.Struct),
		BlockTypes: make(map[string]macosplatformmetadata.BlockType),
		Typedefs:   make(map[string]string),
	}

	// lastAnonEnum tracks the most recently seen anonymous EnumDecl so that a
	// TypedefDecl that immediately follows can promote it to a named enum.
	// Clang represents `typedef enum { A=0, B=1 } name_t` as two adjacent
	// top-level siblings: an anonymous EnumDecl (name="") then a TypedefDecl
	// (name="name_t", type.qualType="enum name_t"). Without promotion the enum
	// members are reachable only as _anon_* constants, not as the typedef type.
	var lastAnonEnum *ASTNode

	// lastAnonStruct tracks the most recently seen anonymous RecordDecl so that a
	// TypedefDecl that immediately follows can promote it to a named struct.
	// Clang represents `typedef struct { ... } Name` as two adjacent top-level
	// siblings: an anonymous RecordDecl (name="") then a TypedefDecl
	// (name="Name", type.qualType="struct Name"). Without promotion the struct's
	// fields are inaccessible by name — only the typedef target survives, which
	// the typemap cannot resolve to a concrete Go struct.
	var lastAnonStruct *ASTNode

	for i := range root.Inner {
		node := &root.Inner[i]
		if node.IsImplicit {
			continue
		}
		// Update rolling file tracker — Clang only emits loc.file when it changes.
		// Still needed for enums, structs, functions which use the path filter.
		f.UpdateFile(node.Loc)
		// Record explicit cross-framework #import relationships from the include chain.
		trackDeclaredImport(node, framework, f)
		switch node.Kind {
		case "ObjCInterfaceDecl":
			// AcceptClass uses the TBD export table when available — ground-truth
			// ownership that eliminates false positives from header inclusion leakage.
			// Falls back to the path filter when no .tbd file is present.
			if f.AcceptClass(node.Name) {
				scanClass(node, framework, f)
			}
			lastAnonEnum = nil
			lastAnonStruct = nil
		case "ObjCCategoryDecl":
			if f.Accept() {
				scanCategory(node, framework, f)
			}
			lastAnonEnum = nil
			lastAnonStruct = nil
		case "ObjCProtocolDecl":
			if f.Accept() {
				scanProtocol(node, framework, f)
			}
			lastAnonEnum = nil
			lastAnonStruct = nil
		case "EnumDecl":
			if f.Accept() {
				scanEnum(node, framework, f)
			}
			if node.Name == "" {
				// Track even if rejected by f.Accept(): the typedef that follows may
				// be accepted and needs the enum data to produce a named entry.
				lastAnonEnum = node
			} else {
				lastAnonEnum = nil
			}
			lastAnonStruct = nil
		case "RecordDecl":
			if f.Accept() {
				scanStruct(node, framework, f)
			}
			// Track anonymous complete struct declarations so the immediately
			// following TypedefDecl can promote them (typedef struct { ... } Name).
			if node.Name == "" && node.CompleteDefinition && node.TagUsed == "struct" {
				lastAnonStruct = node
			} else {
				lastAnonStruct = nil
			}
			lastAnonEnum = nil
		case "FunctionDecl":
			if f.Accept() {
				scanFunction(node, framework, f)
			}
			lastAnonEnum = nil
			lastAnonStruct = nil
		case "VarDecl":
			if f.Accept() {
				scanExtern(node, framework, f)
			}
			lastAnonEnum = nil
			lastAnonStruct = nil
		case "TypedefDecl":
			scanTypedef(node, framework, f)
			// If this typedef names the anonymous enum declared immediately before it,
			// extract that enum under the typedef name so the typemap can resolve it.
			// Pattern: typedef enum { ... } name_t  →  type.qualType == "enum name_t"
			// No f.Accept() guard here: system-header types (dispatch_*, acl_*, qos_class_t,
			// etc.) that arrive via transitive #includes must also be promoted so the
			// typemap's EnumIndex can resolve them as integer types rather than
			// unsafe.Pointer when they appear as function parameters across any framework.
			if lastAnonEnum != nil && node.Type != nil {
				qualType := node.Type.QualType
				enumTag := strings.TrimPrefix(qualType, "enum ")
				if enumTag != qualType && enumTag == node.Name {
					named := *lastAnonEnum
					named.Name = node.Name
					if named.Loc == nil {
						named.Loc = node.Loc
					}
					scanEnum(&named, framework, f)
				}
			}
			// If this typedef names the anonymous struct declared immediately before
			// it, promote the struct fields under the typedef name. Pattern:
			//   typedef struct { ... } Name  →  type.qualType == "struct Name"
			// Unlike anonymous enums, we DO apply f.Accept() here: struct fields can
			// reference other framework types, so admitting system-header structs
			// (mach kernel types, BSD types, CSSM types) would create spurious
			// import-graph edges and import cycles. Framework-owned structs like
			// NSOperatingSystemVersion are in the framework's own headers, so they
			// pass f.Accept() without issue.
			if lastAnonStruct != nil && node.Type != nil && f.Accept() {
				qualType := node.Type.QualType
				structTag := strings.TrimPrefix(qualType, "struct ")
				if structTag != qualType && structTag == node.Name {
					named := *lastAnonStruct
					named.Name = node.Name
					if named.Loc == nil {
						named.Loc = node.Loc
					}
					scanStruct(&named, framework, f)
				}
			}
			lastAnonEnum = nil
			lastAnonStruct = nil
		default:
			lastAnonEnum = nil
			lastAnonStruct = nil
		}
	}

	// Capture C structs the accepted API references by name but that are declared
	// in an external header (e.g. IOUSBHost's deviceDescriptor returns a
	// const IOUSBDeviceDescriptor *, defined in IOKit's usb/AppleUSBDefinitions.h).
	// The frameworkFilter rejects them during the main walk, so without this pass
	// they never reach metadata and the emitter falls back to unsafe.Pointer.
	captureReferencedStructs(root, framework, f)

	// Set LinkLib and the umbrella header include path for known Apple C
	// libraries that use -l rather than -framework. Shim-header libraries
	// (no SDK header) record the shim path instead of an umbrella include.
	if def, ok := knownCLibraries[frameworkName]; ok {
		framework.LinkLib = def.LinkLib
		if def.ShimHeader != "" {
			framework.ShimHeader = def.ShimHeader
		} else {
			framework.Header = CLibraryHeaderRelative(frameworkName)
		}
	}

	// Post-scan classification: detect frameworks that produced no declarations.
	// These fall into two categories:
	//   1. Swift-only — entirely Swift API, no ObjC surface for Clang to see.
	//   2. Umbrella — pure re-export of sub-frameworks; own headers have no
	//      declarations because all content is in the nested bundles.
	if isMetadataEmpty(framework) && framework.LinkLib == "" {
		bundlePath := FrameworkBundlePath(sdkPath, frameworkName)
		if subs := DetectSubFrameworkNames(bundlePath); len(subs) > 0 {
			framework.UmbrellaFor = subs
		} else if IsSwiftOnly(bundlePath) {
			framework.IsSwiftOnly = true
		}
		// If neither condition matches (e.g. stub-only headers like PushToTalk),
		// we leave SwiftOnly false and UmbrellaFor nil. The package still gets a
		// doc.go via the generator's empty-framework path.
	}

	return framework
}

// isMetadataEmpty reports whether a FrameworkMeta captured no API surface.
func isMetadataEmpty(framework *macosplatformmetadata.FrameworkMeta) bool {
	return len(framework.Classes) == 0 &&
		len(framework.Protocols) == 0 &&
		len(framework.Enums) == 0 &&
		len(framework.Structs) == 0 &&
		len(framework.Functions) == 0 &&
		len(framework.Externs) == 0 &&
		len(framework.ForeignExtensions) == 0
}

// --- Classes ---

func scanClass(node *ASTNode, framework *macosplatformmetadata.FrameworkMeta, f *frameworkFilter) {
	// Skip @class Foo; forward declarations. Three forms exist in the Clang AST:
	//
	//  Form 1 — pure forward declaration: Clang omits the inner field entirely
	//  (node.Inner == nil). Real class definitions always emit inner: [...].
	//
	//  Form 2 — forward declaration WITH availability attributes: Clang emits
	//  inner: [...] containing only AvailabilityAttr nodes. The canonical
	//  definition of the class is found via the previousDecl chain. Without
	//  filtering, these nodes create ghost entries for types owned by other
	//  frameworks, corrupting the ownership heuristic and producing false
	//  import-cycle edges (e.g. @class NSUserActivity; in CoreImage headers).
	//
	//  Form 3 — full definition with a previousDecl: when a framework contains
	//  the canonical @interface...@end definition but a forward declaration
	//  appeared earlier in the same TU, Clang sets previousDecl to the forward
	//  declaration's ID. These nodes have significant children and must be kept.
	if node.Inner == nil {
		return
	}

	// Redeclaration guard: a node with previousDecl set and no significant
	// children (form 2) is definitively a redeclaration that adds no API.
	// We skip it unless this framework legitimately owns the class, determined by:
	//   (a) The framework's TBD export table lists this class — ground truth, or
	//   (b) No TBD is available AND the declaration's source range spans multiple
	//       lines, which is characteristic of a real @interface...@end even when
	//       it has no methods (single-line range = @class Foo; redeclaration).
	if node.PreviousDecl != "" && !hasSignificantChildren(node.Inner) {
		if f.allowedClasses != nil {
			if !f.allowedClasses[node.Name] {
				return
			}
		} else {
			// No TBD: fall back to source-range heuristic.
			// A real @interface...@end spans at least two lines; @class Foo;
			// redeclarations are always single-line.
			multiLine := false
			if node.Range != nil {
				b, e := node.Range.Begin.Line, node.Range.End.Line
				multiLine = e > 0 && (b == 0 || e > b)
			}
			if !multiLine {
				return
			}
		}
	}

	// Merge into existing entry — categories processed before this node may have
	// already added methods; we must not lose them by overwriting.
	cls := framework.Classes[node.Name]

	if node.Super != nil && cls.Super == "" {
		cls.Super = node.Super.Name
	}
	if len(node.Protocols) > 0 && len(cls.Protocols) == 0 {
		for _, p := range node.Protocols {
			cls.Protocols = append(cls.Protocols, p.Name)
		}
	}
	absFile, line := nodeFileLine(node, f.currentFile)
	av := scanAvailability(node, absFile, line)
	if av.MacOSIntroduced != "" || av.IsUnavailable {
		cls.Availability = av
	}

	// Source location (only set on the primary definition pass).
	if cls.SDKFile == "" {
		cls.SDKFile = sdkRelativePath(f.sdkPath, absFile)
		cls.SDKLine = line
		cls.Doc = docForNode(absFile, line)
		cls.SwiftName = node.attrName("SwiftNameAttr")
	}

	// Generic type parameters: ObjCTypeParamDecl children.
	// Only collect if not already set — the same ObjCInterfaceDecl can appear
	// multiple times in the AST (re-declarations across headers), and we must
	// not duplicate the params on every pass.
	if len(cls.GenericParams) == 0 {
		seen := map[string]bool{}
		for i := range node.Inner {
			child := &node.Inner[i]
			if child.Kind == "ObjCTypeParamDecl" && !seen[child.Name] {
				seen[child.Name] = true
				cls.GenericParams = append(cls.GenericParams, child.Name)
			}
		}
	}
	// Methods and properties from the primary @interface block
	for i := range node.Inner {
		child := &node.Inner[i]
		absFile := nodeFile(child, f.currentFile)
		switch child.Kind {
		case "ObjCMethodDecl":
			if m, ok := scanMethod(child, absFile, f.sdkPath, cls.GenericParams); ok {
				cls.Methods = append(cls.Methods, m)
			}
		case "ObjCPropertyDecl":
			if prop, ok := scanProperty(child, absFile, f.sdkPath); ok {
				cls.Properties = append(cls.Properties, prop)
			}
		}
	}
	propagatePropertyUnavailability(&cls)
	framework.Classes[node.Name] = cls
}

func scanCategory(node *ASTNode, framework *macosplatformmetadata.FrameworkMeta, f *frameworkFilter) {
	// Categories extend an existing class. The target class is in node.Interface.
	className := ""
	if node.Interface != nil {
		className = node.Interface.Name
	}
	if className == "" {
		// Fallback: parse "ClassName(CategoryName)" format.
		className = node.Name
		if name, _, found := strings.Cut(className, "("); found {
			className = name
		}
	}

	cls, known := framework.Classes[className]
	if !known {
		// The category extends a class owned by another framework (e.g.
		// AppleScriptObjC extends NSBundle). Go cannot add methods to foreign
		// types, so these are recorded as ForeignExtensions and emitted as
		// package-level functions by the code generator.
		if framework.ForeignExtensions == nil {
			framework.ForeignExtensions = make(map[string][]macosplatformmetadata.Method)
		}
		var methods []macosplatformmetadata.Method
		for i := range node.Inner {
			child := &node.Inner[i]
			if child.Kind == "ObjCMethodDecl" {
				absFile := nodeFile(child, f.currentFile)
				if m, ok := scanMethod(child, absFile, f.sdkPath, nil); ok {
					methods = append(methods, m)
				}
			}
		}
		if len(methods) > 0 {
			framework.ForeignExtensions[className] = append(framework.ForeignExtensions[className], methods...)
		}
		return
	}

	for i := range node.Inner {
		child := &node.Inner[i]
		absFile := nodeFile(child, f.currentFile)
		switch child.Kind {
		case "ObjCMethodDecl":
			if m, ok := scanMethod(child, absFile, f.sdkPath, cls.GenericParams); ok {
				cls.Methods = append(cls.Methods, m)
			}
		case "ObjCPropertyDecl":
			if prop, ok := scanProperty(child, absFile, f.sdkPath); ok {
				cls.Properties = append(cls.Properties, prop)
			}
		}
	}
	propagatePropertyUnavailability(&cls)
	framework.Classes[className] = cls
}

// --- Protocols ---

func scanProtocol(node *ASTNode, framework *macosplatformmetadata.FrameworkMeta, f *frameworkFilter) {
	absFile, line := nodeFileLine(node, f.currentFile)
	proto := macosplatformmetadata.Protocol{
		Availability: scanAvailability(node, absFile, line),
	}
	for _, p := range node.Protocols {
		proto.InheritedProtocols = append(proto.InheritedProtocols, p.Name)
	}
	for i := range node.Inner {
		child := &node.Inner[i]
		if child.Kind == "ObjCMethodDecl" {
			childFile := nodeFile(child, f.currentFile)
			if m, ok := scanMethod(child, childFile, f.sdkPath, nil); ok {
				if isOptionalAt(childFile, m.SDKLine) {
					m.IsOptional = true
				}
				proto.Methods = append(proto.Methods, m)
			}
		}
	}
	framework.Protocols[node.Name] = proto
}

// --- Methods ---

func scanMethod(node *ASTNode, absFile, sdkPath string, classGenericParams []string) (macosplatformmetadata.Method, bool) {
	if node.IsImplicit {
		return macosplatformmetadata.Method{}, false
	}
	line := 0
	if node.Loc != nil {
		line = node.Loc.ResolvedLine()
	}
	m := macosplatformmetadata.Method{
		Selector:             node.Name,
		IsClassMethod:        !node.IsInstance, // Clang: instance=true → instance method; absent/false → class method
		IsVariadic:           node.IsVariadic,
		Availability:         scanAvailability(node, absFile, line),
		SDKFile:              sdkRelativePath(sdkPath, absFile),
		SDKLine:              line,
		IsDesignatedInit:     node.hasAttr("ObjCDesignatedInitializerAttr"),
		IsWarnUnused:         node.hasAttr("WarnUnusedResultAttr"),
		SwiftName:            node.attrName("SwiftNameAttr"),
		Doc:                  docForNode(absFile, line),
		IsMainThreadRequired: isMainThreadAt(absFile, line),
	}
	m.IsInit = strings.HasPrefix(node.Name, "init")

	if node.ReturnType != nil {
		m.Return = makeReturnType(node.ReturnType, node, classGenericParams)
	}

	for i := range node.Inner {
		child := &node.Inner[i]
		if child.Kind == "ParmVarDecl" {
			arg := makeParam(child)
			m.Params = append(m.Params, arg)
		}
	}

	// Detect trailing NSError** argument. Guard against block types that happen to
	// contain NSError inside their signature (e.g. void (^)(NSError *)) — those are
	// completion-handler blocks, not out-parameters, and must not be stripped here.
	if len(m.Params) > 0 {
		last := m.Params[len(m.Params)-1]
		if !isBlockType(last.ObjCType) && isNSErrorOut(last.ObjCType) {
			m.IsNSError = true
			m.Params = m.Params[:len(m.Params)-1] // elide from arg list; generator adds error return
		}
	}

	return m, true
}

func makeParam(node *ASTNode) macosplatformmetadata.Param {
	qt := ""
	if node.Type != nil {
		qt = bestQualType(node.Type)
	}
	return macosplatformmetadata.Param{
		Name:       node.Name,
		ObjCType:   qt,
		IsBlock:    isBlockType(qt),
		IsNullable: strings.Contains(qt, "_Nullable") || strings.Contains(qt, "__nullable"),
		IsNoescape: node.hasAttr("NoEscapeAttr"),
		Direction:  outParamModifier(qt),
	}
}

// outParamModifier returns "out" for parameters explicitly marked as output
// via the __autoreleasing storage qualifier (the standard ObjC out-param convention).
// NSError** is excluded here because it is handled separately by elision.
func outParamModifier(qt string) string {
	if strings.Contains(qt, "NSError") {
		return ""
	}
	if strings.Contains(qt, "__autoreleasing") && strings.Count(qt, "*") >= 2 {
		return "out"
	}
	return ""
}

// attrPrefixes lists ObjC/Swift attribute macros that clang embeds verbatim at
// the start of qualType strings. None of these affect the C ABI; stripping them
// lets the rest of the pipeline see the underlying type.
var attrPrefixes = []string{
	"API_AVAILABLE ", "API_DEPRECATED ", "API_UNAVAILABLE ",
	"NS_REFINED_FOR_SWIFT ", "NS_SWIFT_NAME ", "NS_SWIFT_UI_ACTOR ",
	"NS_SWIFT_UNAVAILABLE_FROM_ASYNC ", "NS_SWIFT_UNAVAILABLE ",
	"NS_AVAILABLE_MAC ", "NS_AVAILABLE_IOS ", "NS_AVAILABLE ",
	"NS_DEPRECATED_MAC ", "NS_DEPRECATED_IOS ", "NS_DEPRECATED ",
	"NS_UNAVAILABLE ", "NS_RETURNS_RETAINED ", "NS_NOT_RETAINED ", "NS_RETAINED ",
}

// bestQualType returns the ObjC type string for a clang AST type node.
// It prefers qualType (which carries nullability) but strips leading ObjC/Swift
// attribute macros that clang embeds verbatim in qualType strings.
func bestQualType(t *ASTType) string {
	if t == nil {
		return ""
	}
	qt := t.QualType
	for _, pfx := range attrPrefixes {
		if strings.HasPrefix(qt, pfx) {
			return strings.TrimSpace(qt[len(pfx):])
		}
	}
	return qt
}

func makeReturnType(t *ASTType, method *ASTNode, classGenericParams []string) macosplatformmetadata.ReturnType {
	qt := bestQualType(t)
	r := macosplatformmetadata.ReturnType{
		ObjCType:       qt,
		IsInstancetype: qt == "instancetype",
		IsNullable:     strings.Contains(qt, "_Nullable") || strings.Contains(qt, "__nullable"),
	}
	// Check already_retained: methods named new*, copy*, mutableCopy*, alloc* follow
	// the +1 create rule even without CF_RETURNS_RETAINED; detect via selector prefix.
	sel := method.Name
	for _, prefix := range []string{"new", "alloc", "copy", "mutableCopy"} {
		if sel == prefix || strings.HasPrefix(sel, prefix) && (len(sel) == len(prefix) || sel[len(prefix)] >= 'A' && sel[len(prefix)] <= 'Z') {
			r.IsAlreadyRetained = true
			break
		}
	}
	// Also honour the explicit NS_RETURNS_RETAINED / CF_RETURNS_RETAINED attributes
	// that appear on factory methods outside the standard naming conventions.
	if !r.IsAlreadyRetained && (method.hasAttr("NSReturnsRetainedAttr") || method.hasAttr("CFReturnsRetainedAttr")) {
		r.IsAlreadyRetained = true
	}
	// Detect generic return types: if the return qualType matches one of the class's
	// ObjC type parameter names (e.g. "ObjectType *" on NSArray), mark the return as
	// generic. Enables the emitter to use T instead of unsafe.Pointer.
	if !r.IsInstancetype && len(classGenericParams) > 0 {
		// Strip pointer suffix and nullability qualifiers to get the bare name.
		bare := strings.TrimSpace(qt)
		bare = strings.TrimSuffix(bare, " _Nullable")
		bare = strings.TrimSuffix(bare, " __nullable")
		bare = strings.TrimSuffix(bare, " *")
		bare = strings.TrimSpace(bare)
		for _, param := range classGenericParams {
			if bare == param {
				r.IsGeneric = true
				r.GenericParamName = param
				break
			}
		}
	}
	return r
}

// --- Properties ---

func scanProperty(node *ASTNode, absFile, sdkPath string) (macosplatformmetadata.Property, bool) {
	if node.IsImplicit {
		return macosplatformmetadata.Property{}, false
	}
	qt := ""
	if node.Type != nil {
		qt = bestQualType(node.Type)
	}
	attrs := node.PropertyAttributes
	readonly, weak, copy := false, false, false
	for _, a := range attrs {
		switch a {
		case "readonly":
			readonly = true
		case "weak":
			weak = true
		case "copy":
			copy = true
		}
	}
	getter := ""
	if node.Getter != nil {
		getter = node.Getter.Name
	}
	setter := ""
	if node.Setter != nil {
		setter = node.Setter.Name
	}
	line := 0
	if node.Loc != nil {
		line = node.Loc.ResolvedLine()
	}
	return macosplatformmetadata.Property{
		Name:         node.Name,
		ObjCType:     qt,
		IsReadOnly:   readonly,
		IsWeak:       weak,
		IsCopy:       copy,
		Getter:       getter,
		Setter:       setter,
		Availability: scanAvailability(node, absFile, line),
		SDKFile:      sdkRelativePath(sdkPath, absFile),
		SDKLine:      line,
		Doc:          docForNode(absFile, line),
	}, true
}

// propagatePropertyUnavailability copies unavailability from ObjC properties to
// their synthesized getter/setter ObjCMethodDecl entries. Clang synthesises these
// accessor methods as implicit nodes whose AST loc carries only a byte offset (no
// line number), so lineAvailability returns empty for them. The property node has
// the correct line number and availability, so we propagate it here.
func propagatePropertyUnavailability(cls *macosplatformmetadata.Class) {
	// Build lookup maps: getter selector → property, setter selector → property.
	getters := make(map[string]int, len(cls.Properties)) // selector → index in Properties
	setters := make(map[string]int, len(cls.Properties))
	for i, p := range cls.Properties {
		if !p.Availability.IsUnavailable {
			continue
		}
		getter := p.Getter
		if getter == "" {
			getter = p.Name
		}
		getters[getter] = i
		if !p.IsReadOnly {
			setter := p.Setter
			if setter == "" {
				setter = "set" + strings.ToUpper(p.Name[:1]) + p.Name[1:] + ":"
			}
			setters[setter] = i
		}
	}
	if len(getters) == 0 && len(setters) == 0 {
		return
	}
	for i := range cls.Methods {
		if cls.Methods[i].Availability.IsUnavailable {
			continue
		}
		sel := cls.Methods[i].Selector
		if _, ok := getters[sel]; ok {
			cls.Methods[i].Availability.IsUnavailable = true
		} else if _, ok := setters[sel]; ok {
			cls.Methods[i].Availability.IsUnavailable = true
		}
	}
}

// --- Enums ---

func scanEnum(node *ASTNode, framework *macosplatformmetadata.FrameworkMeta, f *frameworkFilter) {
	if node.Name == "" {
		// anonymous enum — members go into a top-level const block with no type
		// We still capture them under a synthetic key based on first member.
		scanAnonymousEnum(node, framework)
		return
	}
	absFile, line := nodeFileLine(node, f.currentFile)
	e := macosplatformmetadata.Enum{
		Availability: scanAvailability(node, absFile, line),
		SDKFile:      sdkRelativePath(f.sdkPath, absFile),
		SDKLine:      line,
		IsBitmask:    node.hasAttr("FlagEnumAttr"),
		IsExtensible: node.hasAttr("EnumExtensibilityAttr"),
		Doc:          docForNode(absFile, line),
	}
	if node.FixedUnderlyingType != nil {
		e.GoType = objcIntTypeToGo(node.FixedUnderlyingType.QualType)
	}
	if e.GoType == "" {
		e.GoType = "int64" // safe default
	}
	var nextVal int64
	for i := range node.Inner {
		child := &node.Inner[i]
		if child.Kind == "EnumConstantDecl" {
			memberLine := 0
			if child.Loc != nil {
				memberLine = child.Loc.ResolvedLine()
			}
			member := macosplatformmetadata.EnumMember{
				Name:         child.Name,
				Availability: scanAvailability(child, absFile, memberLine),
				Doc:          docForNode(absFile, memberLine),
			}
			// Determine the value. Clang provides an explicit initializer (IntegerLiteral
			// or ConstantExpr child) only when the source has one. Availability attrs and
			// other non-value children do NOT constitute an explicit initializer.
			// When neither the node's Value field nor any value-bearing Inner node is
			// present, the member has an implicit sequential value we must compute.
			hasExplicit := child.Value != "" || hasValueInitializer(child)
			if hasExplicit {
				member.Value = scanEnumValue(child)
			} else {
				member.Value = strconv.FormatInt(nextVal, 10)
			}
			// Advance the counter to (this value + 1) for the next implicit member.
			if v, err := strconv.ParseInt(member.Value, 10, 64); err == nil {
				nextVal = v + 1
			} else {
				nextVal++
			}
			e.Members = append(e.Members, member)
		}
	}
	framework.Enums[node.Name] = e
}

func scanAnonymousEnum(node *ASTNode, framework *macosplatformmetadata.FrameworkMeta) {
	// Collect members into a synthetic "anonymous" group named after first member.
	// Auto-increment implicit member values exactly like the named-enum path:
	// Clang only emits an initializer when the source explicitly provides one,
	// so without this counter every implicit member would fall through to "0",
	// breaking standalone anonymous enums (no typedef twin to compare against).
	var members []macosplatformmetadata.EnumMember
	var nextVal int64
	for i := range node.Inner {
		child := &node.Inner[i]
		if child.Kind != "EnumConstantDecl" {
			continue
		}
		member := macosplatformmetadata.EnumMember{
			Name:         child.Name,
			Availability: scanAvailability(child, "", 0),
		}
		hasExplicit := child.Value != "" || hasValueInitializer(child)
		if hasExplicit {
			member.Value = scanEnumValue(child)
		} else {
			member.Value = strconv.FormatInt(nextVal, 10)
		}
		// Advance counter to (current value + 1) for the next implicit member.
		if v, err := strconv.ParseInt(member.Value, 10, 64); err == nil {
			nextVal = v + 1
		} else {
			nextVal++
		}
		members = append(members, member)
	}
	if len(members) == 0 {
		return
	}
	key := "_anon_" + members[0].Name
	framework.Enums[key] = macosplatformmetadata.Enum{
		GoType:  "int64",
		Members: members,
		IsAnon:  true,
	}
}

func scanEnumValue(node *ASTNode) string {
	// ConstantExpr carries the folded compile-time value directly on the node.
	// This handles all forms of constant expression including bitwise-OR combinations
	// (e.g. FlagA | FlagB), shifts, and cross-enum references — Clang folds them all.
	if node.Value != "" {
		return node.Value.String()
	}
	// Otherwise descend into inner nodes to find a value-bearing child.
	for i := range node.Inner {
		child := &node.Inner[i]
		switch child.Kind {
		case "IntegerLiteral", "FloatingLiteral":
			return child.Value.String()
		case "ConstantExpr", "UnaryOperator", "ImplicitCastExpr":
			return scanEnumValue(child)
		case "BinaryOperator":
			// BinaryOperator covers expressions like (FlagA | FlagB) that appear
			// without a ConstantExpr wrapper in non-standard Clang outputs.
			if v := scanEnumValue(child); v != "0" {
				return v
			}
		}
	}
	return "0"
}

// --- Structs ---

func scanStruct(node *ASTNode, framework *macosplatformmetadata.FrameworkMeta, f *frameworkFilter) {
	if node.Name == "" || node.TagUsed == "union" {
		return
	}
	// Skip forward declarations entirely. Clang emits two RecordDecls for the
	// same struct when there is a `struct Foo;` somewhere ahead of the full
	// definition: one with completeDefinition=true and full FieldDecl children,
	// one with completeDefinition=false and an empty body. Without this guard
	// the forward-decl pass overwrites the complete one, leaving the metadata
	// with zero fields (CGSize, CGRect, CGPoint, CGVector all hit this).
	if !node.CompleteDefinition {
		// Only register a placeholder if we don't already have a complete entry.
		if _, ok := framework.Structs[node.Name]; ok {
			return
		}
	}
	absFile, line := nodeFileLine(node, f.currentFile)
	s := macosplatformmetadata.Struct{
		Packed:       hasPackedAttr(node),
		Availability: scanAvailability(node, absFile, line),
		SDKFile:      sdkRelativePath(f.sdkPath, absFile),
		SDKLine:      line,
		Doc:          docForNode(absFile, line),
	}
	// FieldDecl children sit at the top level of the RecordDecl's Inner slice,
	// but Clang interleaves attribute nodes (AnnotateAttr, ObjCBoxableAttr,
	// SwiftAttrAttr) when the source uses macros like CF_SWIFT_SENDABLE or
	// CF_BOXABLE between `struct` and the name. The scan is robust to these:
	// we filter by Kind=="FieldDecl" and ignore everything else.
	for i := range node.Inner {
		child := &node.Inner[i]
		if child.Kind == "FieldDecl" {
			qt := ""
			if child.Type != nil {
				qt = child.Type.QualType
			}
			s.Fields = append(s.Fields, macosplatformmetadata.StructField{
				Name:     child.Name,
				ObjCType: qt,
				GoType:   "", // filled by typemapper
			})
		}
	}
	// Last-write-wins is unsafe here: if the same RecordDecl appears more
	// than once (re-exported via #include chains, e.g. CGSize seen from both
	// CoreFoundation and CoreGraphics), keep whichever entry actually carries
	// fields. A complete struct shouldn't be replaced by a fields-less one.
	if existing, ok := framework.Structs[node.Name]; ok && len(existing.Fields) > 0 && len(s.Fields) == 0 {
		return
	}
	framework.Structs[node.Name] = s
}

// cScalarSizeAlign returns the size and alignment (bytes, LP64) of a fixed-width
// scalar C field type, and ok=false for anything else (a pointer, array, nested
// struct/union, enum, bitfield, or unrecognised typedef). Only types whose Go
// representation is an unambiguous fixed-width primitive are admitted, so a struct
// built from them lays out identically in Go and C.
func cScalarSizeAlign(cType string) (size, align int, ok bool) {
	t := cType
	for _, q := range []string{"const", "volatile", "_Nonnull", "_Nullable", "_Null_unspecified"} {
		t = strings.ReplaceAll(t, q, " ")
	}
	t = strings.Join(strings.Fields(t), " ")
	// Only fixed-width types whose Go mapping is unambiguous and identical in size
	// are admitted, so the scanner's packed-contiguity check agrees with the
	// emitter's (which computes from the resolved Go type). Native ints of
	// platform-dependent or codegen-dependent width (int, unsigned, short) are
	// deliberately excluded — a struct using them stays an opaque unsafe.Pointer
	// rather than risk a scanner/emitter layout disagreement.
	switch t {
	case "uint8_t", "int8_t", "char", "signed char", "unsigned char", "bool", "_Bool", "Boolean":
		return 1, 1, true
	case "uint16_t", "int16_t", "short", "unsigned short", "short int", "unsigned short int":
		// The mapper resolves short/unsigned short to int16/uint16 (2 bytes), so
		// these are safe. Plain int/unsigned are deliberately excluded: the mapper
		// resolves them to Go int/uint (8 bytes), which does not match C's 4-byte
		// int, so a struct using them stays an opaque unsafe.Pointer.
		return 2, 2, true
	case "uint32_t", "int32_t", "float":
		return 4, 4, true
	case "uint64_t", "int64_t", "double":
		return 8, 8, true
	}
	return 0, 0, false
}

// structCleanlyTypeable reports whether a RecordDecl can be surfaced as a plain
// Go value struct that reproduces the C field offsets: it must have at least one
// field, every field must be a fixed-width scalar, and — if packed — every field
// must already be naturally aligned at its packed offset (so Go inserts no
// inter-field padding). Structs that fail (arrays, nested structs, pointers,
// bitfields, enum fields, or a misaligned packed layout) are left uncaptured so
// their pointer stays an opaque unsafe.Pointer rather than a wrong-layout type.
func structCleanlyTypeable(node *ASTNode) bool {
	packed := hasPackedAttr(node)
	offset, nFields := 0, 0
	for i := range node.Inner {
		child := &node.Inner[i]
		if child.Kind != "FieldDecl" {
			continue
		}
		nFields++
		if child.Type == nil {
			return false
		}
		sz, al, ok := cScalarSizeAlign(child.Type.QualType)
		if !ok {
			return false
		}
		if packed && offset%al != 0 {
			return false
		}
		offset += sz
	}
	return nFields > 0
}

// hasPackedAttr reports whether a RecordDecl carries __attribute__((packed)).
// Clang emits a PackedAttr child node on the record for it.
func hasPackedAttr(node *ASTNode) bool {
	for i := range node.Inner {
		if node.Inner[i].Kind == "PackedAttr" {
			return true
		}
	}
	return false
}

// captureReferencedStructs admits C structs that the framework's accepted API
// references by name but whose definition lives in an external header the
// frameworkFilter rejects (e.g. IOUSBHost methods return const
// IOUSBDeviceDescriptor *, defined in IOKit's usb/AppleUSBDefinitions.h). Without
// this the struct never reaches metadata and the emitter can only surface the
// pointer as unsafe.Pointer.
//
// Only structs actually referenced by an accepted return/parameter/property or a
// captured struct's own field are admitted — never every external struct — so no
// spurious import-graph edges are introduced. The walk repeats to a fixpoint so a
// referenced struct whose fields reference further structs pulls those in too.
func captureReferencedStructs(root *ASTNode, framework *macosplatformmetadata.FrameworkMeta, f *frameworkFilter) {
	for {
		want := referencedStructNames(framework)
		if len(want) == 0 {
			return
		}
		captured := 0
		var lastAnon *ASTNode
		for i := range root.Inner {
			node := &root.Inner[i]
			if node.IsImplicit {
				continue
			}
			// Re-track the rolling file location so scanStruct records the true
			// (external) header path for each admitted struct.
			f.UpdateFile(node.Loc)
			switch node.Kind {
			case "RecordDecl":
				if node.Name != "" && node.CompleteDefinition && node.TagUsed == "struct" &&
					want[node.Name] && structCleanlyTypeable(node) {
					if _, ok := framework.Structs[node.Name]; !ok {
						scanStruct(node, framework, f)
						captured++
					}
				}
				if node.Name == "" && node.CompleteDefinition && node.TagUsed == "struct" {
					lastAnon = node
				} else {
					lastAnon = nil
				}
			case "TypedefDecl":
				// typedef struct { ... } Name — promote the preceding anonymous record
				// under the typedef name when that name is wanted.
				if lastAnon != nil && node.Type != nil && want[node.Name] &&
					structCleanlyTypeable(lastAnon) {
					structTag := strings.TrimPrefix(node.Type.QualType, "struct ")
					if structTag != node.Type.QualType && structTag == node.Name {
						if _, ok := framework.Structs[node.Name]; !ok {
							named := *lastAnon
							named.Name = node.Name
							if named.Loc == nil {
								named.Loc = node.Loc
							}
							scanStruct(&named, framework, f)
							captured++
						}
					}
				}
				lastAnon = nil
			default:
				lastAnon = nil
			}
		}
		if captured == 0 {
			return
		}
	}
}

// referencedStructNames returns the set of bare C struct-name tokens the
// framework's accepted API references (method returns/params/properties, plain
// functions, externs, and the fields of already-captured structs) that are not
// yet present in framework.Structs. Names that turn out not to be structs simply
// never match a RecordDecl and are harmless.
func referencedStructNames(framework *macosplatformmetadata.FrameworkMeta) map[string]bool {
	want := map[string]bool{}
	add := func(objcType string) {
		if name := bareStructIdent(objcType); name != "" {
			if _, ok := framework.Structs[name]; !ok {
				want[name] = true
			}
		}
	}
	for _, class := range framework.Classes {
		for _, m := range class.Methods {
			add(m.Return.ObjCType)
			for _, p := range m.Params {
				add(p.ObjCType)
			}
		}
		for _, p := range class.Properties {
			add(p.ObjCType)
		}
	}
	for _, fn := range framework.Functions {
		add(fn.Return.ObjCType)
		for _, p := range fn.Params {
			add(p.ObjCType)
		}
	}
	for _, ex := range framework.Externs {
		add(ex.ObjCType)
	}
	for _, s := range framework.Structs {
		for _, fld := range s.Fields {
			add(fld.ObjCType)
		}
	}
	return want
}

// bareStructIdent extracts the bare struct identifier from an ObjC type string,
// stripping const/struct keywords, a single pointer, and nullability. It returns
// "" when the type is not a single-pointer-or-value plain identifier (it must not
// match blocks, function pointers, arrays, or double pointers). e.g.
// "const IOUSBDeviceDescriptor * _Nullable" -> "IOUSBDeviceDescriptor".
func bareStructIdent(objcType string) string {
	t := objcType
	for _, q := range []string{"_Nullable", "_Nonnull", "_Null_unspecified", "const", "volatile", "struct"} {
		t = strings.ReplaceAll(t, q, " ")
	}
	if strings.ContainsAny(t, "(^[<") {
		return "" // block, function pointer, array, or generic — not a plain struct
	}
	stars := strings.Count(t, "*")
	if stars > 1 {
		return "" // double pointer — not a value struct reference
	}
	t = strings.ReplaceAll(t, "*", " ")
	fields := strings.Fields(t)
	if len(fields) != 1 {
		return ""
	}
	ident := fields[0]
	// A plain C identifier only.
	for i := 0; i < len(ident); i++ {
		c := ident[i]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return ""
		}
	}
	return ident
}

// --- Functions ---

func scanFunction(node *ASTNode, framework *macosplatformmetadata.FrameworkMeta, f *frameworkFilter) {
	if node.IsImplicit {
		return
	}
	absFile, line := nodeFileLine(node, f.currentFile)
	fn := macosplatformmetadata.Function{
		Name:         node.Name,
		IsVariadic:   node.IsVariadic,
		Availability: scanAvailability(node, absFile, line),
		SDKFile:      sdkRelativePath(f.sdkPath, absFile),
		SDKLine:      line,
		IsWarnUnused: node.hasAttr("WarnUnusedResultAttr"),
		Doc:          docForNode(absFile, line),
	}
	switch {
	case node.ReturnType != nil:
		// ObjCMethodDecl nodes carry a distinct returnType field.
		fn.Return = macosplatformmetadata.ReturnType{ObjCType: node.ReturnType.QualType}
	case node.Type != nil:
		// Plain C FunctionDecl: Clang folds the full function signature into
		// type.qualType (e.g. "CFIndex (CFArrayRef)"). Extract the return type
		// as the portion before the first opening parenthesis.
		if ret := parseFuncReturnType(node.Type.QualType); ret != "" && ret != "void" {
			fn.Return = macosplatformmetadata.ReturnType{ObjCType: ret}
		}
	}
	for i := range node.Inner {
		child := &node.Inner[i]
		if child.Kind == "ParmVarDecl" {
			fn.Params = append(fn.Params, makeParam(child))
		}
	}
	framework.Functions = append(framework.Functions, fn)
}

// parseFuncReturnType extracts the return type from a Clang FunctionDecl's
// type.qualType string, which has the form "ReturnType (Param1, Param2, ...)".
// Returns empty string if the string cannot be parsed (no "(" found).
func parseFuncReturnType(qualType string) string {
	idx := strings.Index(qualType, "(")
	if idx <= 0 {
		return ""
	}
	return strings.TrimSpace(qualType[:idx])
}

// --- Externs ---

func scanExtern(node *ASTNode, framework *macosplatformmetadata.FrameworkMeta, f *frameworkFilter) {
	if node.StorageClass != "extern" {
		return
	}
	qt := ""
	if node.Type != nil {
		qt = node.Type.QualType
	}
	absFile, line := nodeFileLine(node, f.currentFile)
	framework.Externs = append(framework.Externs, macosplatformmetadata.Extern{
		Name:         node.Name,
		ObjCType:     qt,
		Availability: scanAvailability(node, absFile, line),
		SDKFile:      sdkRelativePath(f.sdkPath, absFile),
		SDKLine:      line,
		Doc:          docForNode(absFile, line),
	})
}

// --- Typedefs ---

func scanTypedef(node *ASTNode, framework *macosplatformmetadata.FrameworkMeta, f *frameworkFilter) {
	if node.Type == nil {
		return
	}
	// Framework-specific typedefs (those whose declaration lives in a .framework/
	// header) must be filtered: Clang includes transitively-imported headers in the
	// AST, so scanning Automator also sees CKSubscriptionID from CloudKit. Without
	// filtering, TypedefOwnerIndex attributes these to the wrong framework, creating
	// false import-cycle edges (e.g. cloudkit→automator).
	//
	// System-header typedefs (usr/include, SDK private headers, etc.) intentionally
	// bypass the filter — dispatch_queue_t, qos_class_t, etc. must be resolvable by
	// the typemap from any framework's scan.
	//
	// Use f.currentFile (updated by UpdateFile before each node visit) as the
	// authoritative file path — node.Loc.file is often absent when Clang's cursor
	// optimisation omits the path for nodes in the same file as the previous node.
	file := nodeFile(node, f.currentFile)
	if strings.Contains(file, ".framework/") && !f.Accept() {
		return
	}
	framework.Typedefs[node.Name] = node.Type.QualType
}

// --- Availability ---

// scanAvailability returns the macOS availability for a node.
// It tries AvailabilityAttr nodes from Clang JSON first (works with older
// Clang that emits platform/version fields), then falls back to parsing the
// raw SDK header source — necessary for Apple clang ≥ 21 where AvailabilityAttr
// JSON nodes carry only range information with no platform/version fields.
func scanAvailability(node *ASTNode, absFile string, lineNo int) macosplatformmetadata.Availability {
	var av macosplatformmetadata.Availability
	// Try the top-level Availability array (populated by some Clang versions).
	for i := range node.Availability {
		attr := &node.Availability[i]
		if attr.Kind == "AvailabilityAttr" && attr.Platform == "macos" {
			av.MacOSIntroduced = attr.Introduced
			av.MacOSDeprecated = attr.Deprecated
			av.MacOSObsoleted = attr.Obsoleted
			if attr.Message != "" {
				av.DeprecationMsg = attr.Message
				av.ReplacedBy = parseReplacedBy(attr.Message)
			}
		}
	}
	// Also try inner nodes (another Clang serialisation variant).
	// Apple Clang ≥ 21 emits AvailabilityAttr in inner[] WITHOUT platform/introduced
	// fields. When the platform is present use it directly; otherwise fall back to
	// the expansionLoc (the macro call-site in the SDK header) to parse via lineAvailability.
	for i := range node.Inner {
		child := &node.Inner[i]
		if child.Kind != "AvailabilityAttr" {
			continue
		}
		if child.Platform == "macos" {
			av.MacOSIntroduced = child.Introduced
			av.MacOSDeprecated = child.Deprecated
			av.MacOSObsoleted = child.Obsoleted
			if child.Message != "" {
				av.DeprecationMsg = child.Message
				av.ReplacedBy = parseReplacedBy(child.Message)
			}
		} else if child.Platform == "" && child.Range != nil {
			// Apple Clang ≥ 21: platform field absent; use the expansion location
			// (the actual SDK header line where API_AVAILABLE/API_UNAVAILABLE appears)
			// to parse the annotation from source.
			expFile := child.Range.Begin.expansionFile()
			expLine := child.Range.Begin.expansionLine()
			if expFile != "" && expLine > 0 {
				expAv := lineAvailability(expFile, expLine)
				if expAv.MacOSIntroduced != "" && av.MacOSIntroduced == "" {
					av.MacOSIntroduced = expAv.MacOSIntroduced
					av.MacOSDeprecated = expAv.MacOSDeprecated
				}
				if expAv.MacOSDeprecated != "" && av.MacOSDeprecated == "" {
					av.MacOSDeprecated = expAv.MacOSDeprecated
				}
				if expAv.IsUnavailable {
					av.IsUnavailable = true
				}
				if expAv.DeprecationMsg != "" && av.DeprecationMsg == "" {
					av.DeprecationMsg = expAv.DeprecationMsg
					av.ReplacedBy = expAv.ReplacedBy
				}
			}
		}
	}
	// UnavailableAttr: __attribute__((unavailable)) / NS_UNAVAILABLE
	if node.hasAttr("UnavailableAttr") {
		av.IsUnavailable = true
		return av
	}
	// An obsoleted API is fully removed from the SDK — compiling against it fails.
	// Treat MacOSObsoleted as unavailable so bridge functions are not emitted.
	if av.MacOSObsoleted != "" {
		av.IsUnavailable = true
	}
	// Raw-source fallback (covers Apple clang ≥ 21 which omits fields from JSON).
	if av.MacOSIntroduced == "" && av.MacOSDeprecated == "" && !av.IsUnavailable {
		lineAv := lineAvailability(absFile, lineNo)
		if lineAv.MacOSIntroduced != "" || lineAv.MacOSDeprecated != "" || lineAv.IsUnavailable {
			av = lineAv
		}
	}
	// Entitlement keys from doc comment block preceding this declaration.
	av.Entitlements = entitlementsForNode(absFile, lineNo)
	return av
}

// parseReplacedBy extracts "use X instead" patterns from deprecation messages.
func parseReplacedBy(msg string) string {
	msg = strings.ToLower(msg)
	for _, prefix := range []string{"use ", "prefer "} {
		if _, after, found := strings.Cut(msg, prefix); found {
			rest := after
			// trim trailing punctuation / "instead"
			rest = strings.TrimSuffix(rest, " instead")
			rest = strings.TrimSuffix(rest, ".")
			rest = strings.TrimSpace(rest)
			return rest
		}
	}
	return ""
}

// --- Helpers ---

// hasSignificantChildren returns true if any inner node is a method, property,
// or generic type parameter — indicating a real class definition rather than a
// forward declaration (@class Foo;).
func hasSignificantChildren(inner []ASTNode) bool {
	for i := range inner {
		switch inner[i].Kind {
		case "ObjCMethodDecl", "ObjCPropertyDecl", "ObjCTypeParamDecl":
			return true
		}
	}
	return false
}

func isNSErrorOut(qt string) bool {
	// Require the double-pointer pattern that identifies a genuine out-parameter.
	// NSError out-params appear as "NSError **", "NSError * _Nullable *", or
	// "NSError * __autoreleasing *". Detect by counting '*' after "NSError":
	// two or more stars = double pointer = out-param.
	// A single "NSError *" inside a block completion handler is already
	// excluded by the isBlockType guard at the call site.
	idx := strings.Index(qt, "NSError")
	if idx < 0 {
		return false
	}
	return strings.Count(qt[idx:], "*") >= 2
}

// isBlockType returns true for ObjC block types. Qualifiers after the caret
// (e.g. _Nonnull in "void (^ _Nonnull)(NSError *)") are tolerated by checking
// only for the opening "(^" rather than the exact "(^)" sequence.
func isBlockType(qt string) bool {
	return strings.Contains(qt, "(^")
}

// hasValueInitializer returns true when an EnumConstantDecl node contains an
// inner node that carries an explicit numeric initializer. Availability attrs
// and similar non-value nodes do not qualify — they must be skipped so that
// the enum auto-increment counter stays correct for implicit sequential members.
//
// Typed enums (NS_ENUM / NS_OPTIONS) wrap the initializer expression in an
// ImplicitCastExpr (e.g. an NSUInteger cast around a ConstantExpr), so that
// kind must also be recognised here.  The folded compile-time value lives on
// the ConstantExpr child reached by scanEnumValue's recursion.
func hasValueInitializer(node *ASTNode) bool {
	for _, inner := range node.Inner {
		switch inner.Kind {
		case "IntegerLiteral", "FloatingLiteral", "ConstantExpr", "ImplicitCastExpr",
			"UnaryOperator", "BinaryOperator", "ParenExpr", "CStyleCastExpr":
			return true
		}
	}
	return false
}

// objcIntTypeToGo converts an ObjC integer type name to its Go equivalent.
func objcIntTypeToGo(qt string) string {
	qt = strings.TrimSpace(qt)
	switch qt {
	case "int", "signed int":
		return "int32"
	case "unsigned int":
		return "uint32"
	case "long", "signed long":
		return "int64"
	case "unsigned long", "NSUInteger":
		return "uint64"
	case "long long", "signed long long", "NSInteger":
		return "int64"
	case "unsigned long long":
		return "uint64"
	case "short", "signed short":
		return "int16"
	case "unsigned short":
		return "uint16"
	case "char", "signed char":
		return "int8"
	case "unsigned char":
		return "uint8"
	}
	// If it contains "unsigned", guess uint64
	if strings.Contains(qt, "unsigned") {
		return "uint64"
	}
	return "int64"
}
