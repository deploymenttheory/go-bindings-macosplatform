package emit

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/view"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/shared/fileasm"
)

const (
	PureobjcImport = "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
	PureobjcPkg    = "purego"
	ObjcImport     = "github.com/ebitengine/purego/objc"
	ObjcPkg        = "objc"
)

// keep unexported aliases for use within this package
const (
	pureobjcImport = PureobjcImport
	pureobjcPkg    = PureobjcPkg
	objcImport     = ObjcImport
	objcPkg        = ObjcPkg
)

// EmitClasses writes one .go file per ObjC class in the framework to outDir.
func EmitClasses(
	outDir, packageName, framework string,
	m *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	reg *RegistrySnapshot,
) error {
	names := make([]string, 0, len(m.Classes))
	for name := range m.Classes {
		names = append(names, name)
	}
	sort.Strings(names)

	// macOS uses a case-insensitive filesystem by default; detect class names
	// that produce case-insensitive filename collisions and disambiguate.
	lowerCount := make(map[string]int, len(names))
	for _, name := range names {
		lowerCount[strings.ToLower(name)]++
	}
	lowerSeen := make(map[string]int, len(names))

	for _, name := range names {
		cls := m.Classes[name]
		if cls.Availability.IsUnavailable {
			continue
		}

		baseName := name
		lower := strings.ToLower(name)
		if lowerCount[lower] > 1 {
			lowerSeen[lower]++
			if lowerSeen[lower] > 1 {
				baseName = fmt.Sprintf("%s_%d", name, lowerSeen[lower])
			}
		}

		var buf bytes.Buffer
		if err := emitClass(&buf, name, cls, packageName, framework, m, mapper, reg); err != nil {
			return fmt.Errorf("emit class %s: %w", name, err)
		}
		fname := filepath.Join(outDir, baseName+".go")
		if err := WriteGoFile(fname, buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

// RegistrySnapshot carries the cross-framework lookup tables the class emitter needs.
type RegistrySnapshot struct {
	OwnerIndex         map[string]string
	GenericClasses     map[string]bool
	GenericParamIndex  map[string][]string
	ClassIndex         map[string]meta.Class
	BlockedImports     map[string]map[string]bool
	EnumGoTypeIndex    map[string]string // enum name → underlying Go type (e.g. "int64")
	UnavailableClasses map[string]bool   // classes marked IsUnavailable in metadata
	ModulePrefix       string            // Go module path prefix for framework packages
}

func emitClass(
	w io.Writer,
	className string,
	cls meta.Class,
	packageName, framework string,
	m *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	reg *RegistrySnapshot,
) error {
	isGeneric := reg.GenericClasses[className]
	genericParams := reg.GenericParamIndex[className]

	// Build context for type resolution.
	ctx := typemap.Context{
		ClassName:      className,
		Framework:      framework,
		GenericParams:  genericParams,
		ClassNameIndex: reg.OwnerIndex,
	}

	// Collect selectors used in this class.
	var selectors []selectorEntry
	selectorSeen := make(map[string]bool)

	for _, method := range cls.Methods {
		if !isMethodBridgeable(method) {
			continue
		}
		key := method.Selector + ":" + fmt.Sprint(method.IsClassMethod)
		if !selectorSeen[key] {
			selectorSeen[key] = true
			selectors = append(selectors, selectorEntry{method.Selector, method.IsClassMethod})
		}
	}

	// Write body into buffer first to discover imports.
	var body bytes.Buffer
	imports := make(typemap.ImportSet)

	// Type declaration.
	typeDeclOut, err := render.ClassTypeDecl(buildClassTypeDeclView(className, cls, isGeneric, genericParams, framework, reg, imports))
	if err != nil {
		return err
	}
	body.Write(typeDeclOut)

	// Package-level vars for class ref and selectors.
	classVarsOut, err := render.ClassVars(buildClassVarsView(className, selectors))
	if err != nil {
		return err
	}
	body.Write(classVarsOut)

	// Constructor (XFromID).
	fromIDOut, err := render.FromIDConstructor(buildFromIDConstructorView(className, isGeneric, genericParams))
	if err != nil {
		return err
	}
	body.Write(fromIDOut)

	// Methods.
	goNameCount := make(map[string]int)
	for _, method := range cls.Methods {
		if !isMethodBridgeable(method) {
			continue
		}
		goName := naming.MethodName(method.Selector)
		if method.IsClassMethod {
			goName = classMethodName(className+goName, m, reg)
		}
		goNameCount[goName]++
	}

	goNameSeen := make(map[string]int)
	selectorSeen2 := make(map[string]bool)

	for _, method := range cls.Methods {
		if !isMethodBridgeable(method) {
			continue
		}
		key := method.Selector + ":" + fmt.Sprint(method.IsClassMethod)
		if selectorSeen2[key] {
			continue
		}
		selectorSeen2[key] = true

		goName := naming.MethodName(method.Selector)
		// Prefix class methods with the class name to prevent package-level
		// symbol collisions when multiple ObjC classes in the same framework
		// share a class method name (e.g. NSDate and NSCalendarDate both have
		// +distantFuture). Instance methods have receivers so no prefix needed.
		if method.IsClassMethod {
			goName = classMethodName(className+goName, m, reg)
		}
		goNameSeen[goName]++
		if goNameCount[goName] > 1 && goNameSeen[goName] > 1 {
			goName = fmt.Sprintf("%s%d", goName, goNameSeen[goName])
		}

		methodOut, err := render.ClassMethod(buildRawMethodView(
			goName,
			className,
			method,
			ctx,
			mapper,
			imports,
			isGeneric,
			genericParams,
			reg,
		))
		if err != nil {
			return err
		}
		body.Write(methodOut)
	}

	// Collect imports: the cross-framework types discovered during resolution,
	// always objc and purego, plus unsafe/fmt when the rendered body uses them.
	bodyStr := body.String()
	allImports := make(typemap.ImportSet)
	for alias, path := range imports {
		allImports[alias] = path
	}
	allImports[objcPkg] = objcImport
	allImports[pureobjcPkg] = pureobjcImport
	if strings.Contains(bodyStr, "unsafe.") {
		allImports["unsafe"] = "unsafe"
	}
	if strings.Contains(bodyStr, "fmt.") {
		allImports["fmt"] = "fmt"
	}

	// generatedHeader carries both the DO-NOT-EDIT comment and the build tag with
	// no blank line between them, so it is the whole Header and BuildTag is empty.
	out := fileasm.Assemble(fileasm.File{
		Header:      strings.TrimRight(generatedHeader, "\n"),
		PkgName:     packageName,
		ImportLines: fileasm.ImportLinesStdlibExternalInternal(allImports),
		Body:        bodyStr,
	})
	_, err = w.Write(out)
	return err
}

// buildClassTypeDeclView resolves an ObjC class's Go struct declaration: the doc
// comment, the (generic) type header, the embedded field (raw ptr or superclass
// struct), and whether the promoted Ptr/InitPtr accessors are emitted. A
// cross-framework superclass import is accumulated into imports as a side
// effect, exactly as the original emitter did.
func buildClassTypeDeclView(
	className string,
	cls meta.Class,
	isGeneric bool,
	genericParams []string,
	framework string,
	reg *RegistrySnapshot,
	imports typemap.ImportSet,
) view.ClassTypeDecl {
	var comment strings.Builder
	if cls.Doc != "" {
		fmt.Fprintf(&comment, "// %s\n//\n", cls.Doc)
	}
	// Apple's class documentation URLs follow a deterministic lowercase scheme,
	// so the link is computable from metadata alone (method URLs are not).
	fmt.Fprintf(&comment, "// Apple documentation: https://developer.apple.com/documentation/%s/%s\n",
		strings.ToLower(framework), strings.ToLower(className))
	comment.WriteString(deprecatedComment(cls.Availability))

	typeHeader := className
	if isGeneric {
		constraints := make([]string, len(genericParams))
		for i, gp := range genericParams {
			// AnyObject accepts both raw objc.ID and typed wrapper structs,
			// so generic params like "id" (any object) work without unsafe.Pointer.
			constraints[i] = gp + " " + pureobjcPkg + ".AnyObject"
		}
		typeHeader = className + "[" + strings.Join(constraints, ", ") + "]"
	}

	superOwner := reg.OwnerIndex[cls.Super]
	superBlocked := cls.Super != "" && superOwner != "" && superOwner != framework &&
		reg.BlockedImports[framework] != nil && reg.BlockedImports[framework][superOwner]
	// Treat unavailable super-classes as unknown to avoid referencing a type that was
	// not emitted (EmitClasses skips IsUnavailable classes).
	superUnavailable := cls.Super != "" && reg.UnavailableClasses != nil &&
		reg.UnavailableClasses[cls.Super]

	isRoot := cls.Super == "" || reg.OwnerIndex[cls.Super] == "" || superBlocked || superUnavailable

	embedLine := "ptr objc.ID"
	if !isRoot {
		// Embed the superclass struct.
		superType := cls.Super
		if superOwner != framework {
			pkg := strings.ToLower(superOwner)
			imports[pkg] = reg.ModulePrefix + "/" + pkg
			superType = pkg + "." + cls.Super
		}
		if reg.GenericClasses[cls.Super] {
			superParams := reg.GenericParamIndex[cls.Super]
			if isGeneric && len(genericParams) > 0 {
				// Map child's generic params to parent's by position.
				args := make([]string, len(superParams))
				for i := range args {
					if i < len(genericParams) {
						args[i] = genericParams[i]
					} else {
						args[i] = "objc.ID"
					}
				}
				superType = superType + "[" + strings.Join(args, ", ") + "]"
			} else {
				// Non-generic child — instantiate parent's params with objc.ID.
				ids := make([]string, len(superParams))
				for i := range ids {
					ids[i] = "objc.ID"
				}
				superType = superType + "[" + strings.Join(ids, ", ") + "]"
			}
		}
		embedLine = superType
	}

	built := view.ClassTypeDecl{
		CommentBlock:   comment.String(),
		TypeHeader:     typeHeader,
		EmbedLine:      embedLine,
		EmitPtrMethods: isRoot,
	}
	if isRoot {
		// Ptr/InitPtr are promoted accessors: subclasses in other packages cannot
		// reach the unexported ptr field directly, so they read and write it
		// through these.
		typeSuffix := ""
		if isGeneric {
			typeSuffix = "[" + strings.Join(genericParams, ", ") + "]"
		}
		built.PtrReceiver = className + typeSuffix
	}
	return built
}

// writeClassVars writes the package-level selector and class var declarations.
// Class lookups go through _objcClass (emitted in the package runtime file),
// which force-loads the framework first: package-level var initializers run
// before the runtime file's init(), so a plain objc.GetClass here would
// resolve to a nil class for any framework not already loaded into the
// process.
func buildClassVarsView(className string, selectors []selectorEntry) view.ClassVars {
	built := view.ClassVars{
		ClassVarName: varClassName(className),
		ClassName:    className,
	}
	// Deduplicate selectors for the var block.
	seen := make(map[string]bool)
	for _, selector := range selectors {
		varName := varSelectorName(className, selector.selector)
		if seen[varName] {
			continue
		}
		seen[varName] = true
		built.Selectors = append(built.Selectors, view.ClassSelectorVar{
			VarName:  varName,
			Selector: selector.selector,
		})
	}
	return built
}

// buildFromIDConstructorView resolves a class's XFromID factory: its signature
// (generic when the class has type parameters) and the type literal allocated
// for the wrapper.
func buildFromIDConstructorView(className string, isGeneric bool, genericParams []string) view.FromIDConstructor {
	signature := fmt.Sprintf("func %sFromID(id objc.ID) *%s", className, className)
	if isGeneric {
		// Generic version — use AnyObject constraint (= any) to accept both
		// raw objc.ID and typed wrapper structs without unsafe.Pointer.
		params := make([]string, len(genericParams))
		constraints := make([]string, len(genericParams))
		for i, gp := range genericParams {
			params[i] = gp
			constraints[i] = gp + " " + pureobjcPkg + ".AnyObject"
		}
		signature = fmt.Sprintf(
			"func %sFromID[%s](id objc.ID) *%s[%s]",
			className,
			strings.Join(constraints, ", "),
			className,
			strings.Join(params, ", "),
		)
	}

	allocType := className
	if isGeneric && len(genericParams) > 0 {
		allocType = className + "[" + strings.Join(genericParams, ", ") + "]"
	}

	return view.FromIDConstructor{Signature: signature, AllocType: allocType}
}

// isMethodBridgeable reports whether an ObjC method can be bridged via purego.
func isMethodBridgeable(method meta.Method) bool {
	if method.Availability.IsUnavailable {
		return false
	}
	if method.IsVariadic && !isSelectorFormatVariadic(method.Selector) {
		return false
	}
	if method.IsClassMethod && (method.Selector == "initialize" || method.Selector == "load") {
		return false
	}
	return true
}

// isSelectorFormatVariadic returns true when the selector contains "format".
func isSelectorFormatVariadic(selector string) bool {
	for _, part := range strings.Split(selector, ":") {
		if strings.Contains(strings.ToLower(part), "format") {
			return true
		}
	}
	return false
}

// methodCallModel carries the assembled pieces of a generated method body.
type methodCallModel struct {
	goParams      []string
	sendArgs      []string
	blockAdapters []blockAdapterModel
	retGoType     string
	retIsVoid     bool
	retIsObject   bool
}

// buildRawMethodView resolves one Go method wrapping an ObjC method via
// objc.Send: its signature, the block adapters for block parameters, the
// objc.Send target and argument list, and the return-kind dispatch.
func buildRawMethodView(
	goName, className string,
	method meta.Method,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	imports typemap.ImportSet,
	isGeneric bool,
	genericParams []string,
	reg *RegistrySnapshot,
) view.RawMethod {
	selVarName := varSelectorName(className, method.Selector)

	call := buildMethodCallModel(
		className, method, ctx, mapper, imports, isGeneric, genericParams, reg,
	)

	// For class methods on generic classes, the generic type params (e.g.
	// ObjectType) are NOT in scope (no receiver). Substitute them with objc.ID
	// so the generated free function has a concrete, valid signature.
	if method.IsClassMethod && isGeneric && len(genericParams) > 0 {
		substituteCallModelGenericParams(&call, genericParams)
	}

	// Method receiver (or no receiver for class methods).
	receiver := ""
	if !method.IsClassMethod {
		typeSuffix := ""
		if isGeneric && len(genericParams) > 0 {
			typeSuffix = "[" + strings.Join(genericParams, ", ") + "]"
		}
		receiver = fmt.Sprintf("(o *%s%s) ", className, typeSuffix)
	}

	var returnSig string
	switch {
	case !call.retIsVoid && method.IsNSError:
		returnSig = " (" + call.retGoType + ", error)"
	case method.IsNSError:
		returnSig = " error"
	case !call.retIsVoid:
		returnSig = " " + call.retGoType
	}

	var comment strings.Builder
	if method.Doc != "" {
		fmt.Fprintf(&comment, "// %s\n", method.Doc)
	}
	comment.WriteString(deprecatedComment(method.Availability))

	target := "o.Ptr()"
	if method.IsClassMethod {
		target = fmt.Sprintf("objc.ID(%s)", varClassName(className))
	}

	sendArgStr := ""
	if len(call.sendArgs) > 0 {
		sendArgStr = ", " + strings.Join(call.sendArgs, ", ")
	}
	// NSError handling — append a *objc.ID slot.
	if method.IsNSError {
		sendArgStr += ", unsafe.Pointer(&_nsErr)"
	}

	built := view.RawMethod{
		CommentBlock:    comment.String(),
		Receiver:        receiver,
		GoName:          goName,
		ParamStr:        strings.Join(call.goParams, ", "),
		ReturnSig:       returnSig,
		HasNSError:      method.IsNSError,
		Target:          target,
		SelVar:          selVarName,
		SendArgStr:      sendArgStr,
		RetGoType:       call.retGoType,
		AlreadyRetained: method.Return.IsAlreadyRetained,
	}
	for _, adapter := range call.blockAdapters {
		built.Adapters = append(built.Adapters, blockAdapterRenderView(adapter))
	}
	switch {
	case call.retIsVoid:
		built.ReturnKind = 0
	case call.retIsObject:
		built.ReturnKind = 1
		built.WrapExpr = buildWrapExpr(call.retGoType, className, isGeneric, genericParams)
	case call.retGoType == "string":
		built.ReturnKind = 2
	case call.retGoType == "bool":
		built.ReturnKind = 3
	default:
		built.ReturnKind = 4
		if method.IsNSError {
			built.ZeroVal = zeroValueForReturn(call.retGoType, reg)
		}
	}
	return built
}

// buildMethodCallModel resolves the parameter list, send arguments, block
// adapters, and return type for one method.
func buildMethodCallModel(
	className string,
	method meta.Method,
	ctx typemap.Context,
	mapper *typemap.Mapper,
	imports typemap.ImportSet,
	isGeneric bool,
	genericParams []string,
	reg *RegistrySnapshot,
) methodCallModel {
	var call methodCallModel
	usedParamNames := make(map[string]int)

	for _, param := range method.Params {
		rawName := naming.ParamName(param.Name)
		usedParamNames[rawName]++
		paramName := rawName
		if usedParamNames[rawName] > 1 {
			paramName = fmt.Sprintf("%s%d", rawName, usedParamNames[rawName])
		}

		// Block params: literal "(^" types carry IsBlock; typedef-named blocks
		// are detected via the typedef chain.
		isBlockParam := param.IsBlock
		if !isBlockParam {
			_, isBlockParam = mapper.ResolveBlockSignature(param.ObjCType)
		}
		if isBlockParam {
			adapter := buildBlockAdapter(
				paramName,
				param.ObjCType,
				ctx,
				mapper,
				imports,
				reg.OwnerIndex,
			)
			call.goParams = append(call.goParams, paramName+" "+adapter.PublicGoType)
			if adapter.Degraded {
				mapper.AppendDiagnostic(
					"%s: %s.%s param %s → objc.Block (%s)",
					ctx.Framework, className, method.Selector, paramName, adapter.DegradeReason,
				)
				call.sendArgs = append(call.sendArgs, paramName)
			} else {
				call.blockAdapters = append(call.blockAdapters, adapter)
				call.sendArgs = append(call.sendArgs, "__block_"+paramName)
			}
			continue
		}

		goType := mapper.GoType(param.ObjCType, ctx, imports)
		if goType == "" {
			goType = "unsafe.Pointer"
		}
		call.goParams = append(call.goParams, paramName+" "+goType)

		// Convert the Go arg to what objc.Send expects.
		// Guard with isObjCClass to avoid calling .Ptr() on struct pointers
		// like *NSRange that are not ObjC objects.
		if isObjCObjectType(goType) && isObjCClass(goType, reg.OwnerIndex) {
			call.sendArgs = append(call.sendArgs, paramName+".Ptr()")
		} else {
			call.sendArgs = append(call.sendArgs, paramName)
		}
	}

	// Return type.
	retObjCType := method.Return.ObjCType
	call.retIsVoid = retObjCType == "void" || retObjCType == ""
	_, retIsBlock := mapper.ResolveBlockSignature(retObjCType)

	switch {
	case method.Return.IsInstancetype:
		if isGeneric && len(genericParams) > 0 {
			call.retGoType = "*" + className + "[" + strings.Join(genericParams, ", ") + "]"
		} else {
			call.retGoType = "*" + className
		}
		call.retIsObject = true
	case retIsBlock:
		// A returned block surfaces as the raw block object — objc.Send
		// instantiated with a Go func type would make purego treat the
		// returned pointer as a C function pointer and panic.
		call.retGoType = "objc.Block"
	case !call.retIsVoid:
		call.retGoType = mapper.GoReturnType(retObjCType, ctx, imports)
		if call.retGoType == "" {
			call.retGoType = "unsafe.Pointer"
		}
		call.retIsObject = isObjCObjectType(call.retGoType) &&
			isObjCClass(call.retGoType, reg.OwnerIndex)
	}

	return call
}

// substituteCallModelGenericParams rewrites generic type parameters to
// objc.ID throughout a class method's call model (signature, send args, and
// block adapters), since type params are not in scope without a receiver.
func substituteCallModelGenericParams(call *methodCallModel, genericParams []string) {
	substituteGenericParams := func(s string) string {
		for _, gp := range genericParams {
			// Replace "Foo[ObjectType]" → "Foo[objc.ID]", including multi-param
			// generic positions, then bare tokens with word boundaries.
			s = strings.ReplaceAll(s, "["+gp+"]", "[objc.ID]")
			s = strings.ReplaceAll(s, "["+gp+",", "[objc.ID,")
			s = strings.ReplaceAll(s, ", "+gp+"]", ", objc.ID]")
			s = replaceToken(s, gp, "objc.ID")
		}
		return s
	}
	call.retGoType = substituteGenericParams(call.retGoType)
	for i, p := range call.goParams {
		call.goParams[i] = substituteGenericParams(p)
	}
	for i, a := range call.sendArgs {
		call.sendArgs[i] = substituteGenericParams(a)
	}
	// Block adapters capture generic params in their callback signatures
	// and conversion expressions — substitute those too.
	for ai := range call.blockAdapters {
		adapter := &call.blockAdapters[ai]
		adapter.PublicGoType = substituteGenericParams(adapter.PublicGoType)
		adapter.RetGoType = substituteGenericParams(adapter.RetGoType)
		for pi := range adapter.Params {
			component := &adapter.Params[pi]
			component.PublicGoType = substituteGenericParams(component.PublicGoType)
			component.ABIType = substituteGenericParams(component.ABIType)
			component.ConvertFmt = substituteGenericParams(component.ConvertFmt)
		}
	}
	// Recalculate object-ness with the substituted type.
	call.retIsObject = isObjCObjectType(call.retGoType)
}

// buildWrapExpr returns the expression to convert an objc.ID to a typed Go pointer.
func buildWrapExpr(retGoType, className string, isGeneric bool, genericParams []string) string {
	// If instancetype → use XFromID with type args extracted from retGoType itself.
	// Do NOT use genericParams here — retGoType may have had generic params substituted
	// (e.g. ObjectType → objc.ID for class methods).
	//
	// Guard: after stripping "*ClassName" the remainder must be "" or "[..." —
	// otherwise a longer class name shares a prefix (e.g. *NSURLSession matching
	// against *NSURLSessionDataTask) and we must fall through to the generic path.
	prefix := "*" + className
	if strings.HasPrefix(retGoType, prefix) {
		rest := retGoType[len(prefix):]
		if rest == "" || strings.HasPrefix(rest, "[") {
			// retGoType is exactly *ClassName or *ClassName[T1,T2]
			_, typeArgs := splitGenericName(rest)
			actualArgs := typeArgs
			if actualArgs == "" && isGeneric && len(genericParams) > 0 {
				actualArgs = strings.Join(genericParams, ", ")
			}
			if actualArgs != "" {
				return fmt.Sprintf("%sFromID[%s](_ret)", className, actualArgs)
			}
			return fmt.Sprintf("%sFromID(_ret)", className)
		}
	}
	// Cross-package pointer: *pkg.Class or *pkg.Class[T] → pkg.ClassFromID[T](_ret)
	// Same-package generic: *Class[T] → ClassFromID[T](_ret)
	if strings.HasPrefix(retGoType, "*") {
		inner := retGoType[1:]

		// First split off generics so we don't confuse dots inside type params
		// (e.g. "[objc.ID]") with package separators.
		baseType, typeArgs := splitGenericName(inner) // "pkg.Class", "objc.ID"

		// Now check for a package prefix in the base type (no brackets here).
		if dotIdx := strings.Index(baseType, "."); dotIdx >= 0 {
			pkg := baseType[:dotIdx]
			typName := baseType[dotIdx+1:]
			if typeArgs != "" {
				return fmt.Sprintf("%s.%sFromID[%s](_ret)", pkg, typName, typeArgs)
			}
			return fmt.Sprintf("%s.%sFromID(_ret)", pkg, typName)
		}
		// Same package.
		if typeArgs != "" {
			return fmt.Sprintf("%sFromID[%s](_ret)", baseType, typeArgs)
		}
		return fmt.Sprintf("%sFromID(_ret)", baseType)
	}
	return "_ret"
}

// replaceToken replaces all word-boundary occurrences of old with new_ in s.
// Handles cases like "anObject ObjectType" → "anObject objc.ID" but not
// "SomeObjectType" (would not match "ObjectType" as it's not word-boundary).
func replaceToken(s, old, new_ string) string {
	if !strings.Contains(s, old) {
		return s
	}
	var result strings.Builder
	i := 0
	for i < len(s) {
		idx := strings.Index(s[i:], old)
		if idx < 0 {
			result.WriteString(s[i:])
			break
		}
		abs := i + idx
		// Check boundaries: preceding char must be non-identifier.
		prevOK := abs == 0 || !isIdentChar(s[abs-1])
		// Following char must be non-identifier.
		end := abs + len(old)
		nextOK := end >= len(s) || !isIdentChar(s[end])
		if prevOK && nextOK {
			result.WriteString(s[i:abs])
			result.WriteString(new_)
			i = end
		} else {
			result.WriteString(s[i : abs+1])
			i = abs + 1
		}
	}
	return result.String()
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// splitGenericName splits "NSArray[ObjectType]" into ("NSArray", "ObjectType").
// Returns ("TypeName", "") if no generic params present.
func splitGenericName(name string) (typName, typeArgs string) {
	brIdx := strings.Index(name, "[")
	if brIdx < 0 {
		return name, ""
	}
	typName = name[:brIdx]
	// Strip surrounding brackets.
	inner := name[brIdx+1:]
	inner = strings.TrimSuffix(inner, "]")
	return typName, inner
}

// zeroValue returns the Go zero value expression for a type.
func zeroValue(goType string) string {
	switch goType {
	case "bool":
		return "false"
	case "string":
		return `""`
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return "0"
	case "float32", "float64":
		return "0"
	// objc runtime types are uintptr aliases — use numeric zero.
	case "objc.ID", "objc.SEL", "objc.Class":
		return "0"
	case "unsafe.Pointer":
		return "nil"
	}
	if strings.HasPrefix(goType, "*") {
		return "nil"
	}
	return goType + "{}"
}

// zeroValueForReturn returns the zero value for a method return type, correctly
// handling Go interface types (ObjC protocol mappings) that cannot use TypeName{}.
// ownerIndex is used to distinguish ObjC class struct types from protocol interfaces.
// enumGoTypeIndex is used to return "0" for enum types rather than "nil".
func zeroValueForReturn(goType string, reg *RegistrySnapshot) string {
	if strings.HasPrefix(goType, "*") || isPrimitiveLike(goType) {
		return zeroValue(goType)
	}
	// Non-pointer, non-primitive: may be a protocol (Go interface), enum, or C struct.
	baseName := goType
	if dotIdx := strings.LastIndex(baseName, "."); dotIdx >= 0 {
		baseName = baseName[dotIdx+1:]
	}
	if brIdx := strings.Index(baseName, "["); brIdx >= 0 {
		baseName = baseName[:brIdx]
	}
	// Enum types have integer zero values.
	if reg != nil && reg.EnumGoTypeIndex != nil {
		if _, isEnum := reg.EnumGoTypeIndex[baseName]; isEnum {
			return "0"
		}
	}
	// Known ObjC class → struct type literal.
	if reg != nil && reg.OwnerIndex != nil {
		if _, isClass := reg.OwnerIndex[baseName]; isClass {
			return zeroValue(goType)
		}
	}
	// Not a known ObjC class or enum — assume interface/protocol → nil.
	return "nil"
}

// isPrimitiveLike reports whether a type string is a primitive or objc runtime type.
func isPrimitiveLike(t string) bool {
	switch t {
	case "bool", "string",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"objc.ID", "objc.SEL", "objc.Class", "unsafe.Pointer":
		return true
	}
	return false
}

type selectorEntry struct {
	selector      string
	isClassMethod bool
}

// classMethodName returns a safe Go package-level function name for an ObjC
// class method. If the candidate name collides with a type declaration in the
// same package (enum, struct, or class), a "Class" suffix is appended.
func classMethodName(candidate string, m *meta.FrameworkMeta, reg *RegistrySnapshot) string {
	if _, isEnum := m.Enums[candidate]; isEnum {
		return candidate + "Class"
	}
	if _, isStruct := m.Structs[candidate]; isStruct {
		return candidate + "Class"
	}
	if owner, ok := reg.OwnerIndex[candidate]; ok && owner == m.Framework {
		return candidate + "Class"
	}
	return candidate
}
