package raw

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// Bridge generates the C bridge header (.h) and ObjC implementation (.m)
// for a framework, writing both into outDir/bridge/, plus a thin shim .m file
// in outDir (the package root) that #includes the bridge implementation so that
// CGo auto-compiles it. (CGo only auto-compiles .m files in the package root,
// not in subdirectories.)
//
// Why we generate C bridge functions rather than calling ObjC methods directly:
//
// CGo only understands C calling conventions — it cannot express Objective-C
// message sends ([object method]) at all. The ObjC methods already exist on
// macOS, compiled into the framework dylibs, but the only way Go can invoke
// them is through a plain C function. Each generated bridge function:
//   1. Accepts CGo-compatible parameters (void* for ObjC objects, scalar types)
//   2. Casts them back to ObjC types and sends the message
//   3. Retains the result with [retain] before returning (see memory model below)
//   4. Returns a C-compatible value
//
// No method logic is reimplemented; the bridge is purely a calling-convention
// adapter. The alternative — using objc_msgSend directly — is fragile: the
// cast of the function pointer varies by return type and architecture. The
// generated wrapper approach is far more robust.
//
// Memory model for returned ObjC objects (non-ARC):
//
// All bridge .m files are compiled with -fno-objc-arc. In non-ARC code,
// __bridge_retained is a no-op (equivalent to __bridge). Returned ObjC
// objects would be autoreleased by the @autoreleasepool block and freed
// before the Go caller could use them. To match Go's finalizer model, every
// bridge function that returns an ObjC object calls [_result retain] before
// returning. The Go wrapper calls runtime.Track to register a CFRelease
// finalizer, which releases that +1 retain when the Go GC collects the
// wrapper.
//
// Why some methods are skipped (marked Unavailable in metadata):
//
// A small number of SDK classes mark their -init and +new methods with
// __attribute__((unavailable)) / NS_UNAVAILABLE. This is a hard Clang
// compile-time error, not a runtime version guard. These are factory-only
// classes (e.g. NSCollectionLayoutDimension, WKWebExtensionAction) where
// Apple's design explicitly forbids bare initialisation; callers must use
// the class factory methods. Generating call sites for unavailable methods
// would produce code that does not compile. Skipping them honours the SDK's
// explicit design intent.
func EmitBridge(outDir string, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) error {
	bridgeDir := filepath.Join(outDir, "bridge")
	if err := os.MkdirAll(bridgeDir, 0o755); err != nil {
		return err
	}

	stem := strings.ToLower(framework.Framework) + "_bridge"
	hPath := filepath.Join(bridgeDir, stem+".h")
	mPath := filepath.Join(bridgeDir, stem+".m")

	hFile, err := os.Create(hPath)
	if err != nil {
		return fmt.Errorf("create bridge header: %w", err)
	}
	defer hFile.Close()

	mFile, err := os.Create(mPath)
	if err != nil {
		return fmt.Errorf("create bridge impl: %w", err)
	}
	defer mFile.Close()

	if err := EmitBridgeHeader(hFile, framework, m, knownClasses); err != nil {
		return err
	}
	if err := EmitBridgeImpl(mFile, framework, m, knownClasses, stem+".h"); err != nil {
		return err
	}

	// Write a thin shim .m in the package root. CGo auto-compiles any .m file
	// found directly in the package directory — it does NOT recurse into bridge/.
	// This shim includes the real implementation so the symbols are available
	// when the test binary (or any binary) is linked.
	shimPath := filepath.Join(outDir, stem+"_impl.m")
	shimFile, err := os.Create(shimPath)
	if err != nil {
		return fmt.Errorf("create bridge shim: %w", err)
	}
	defer shimFile.Close()
	return executeTemplate(shimFile, "bridge_shim", bridgeShimModel{Stem: stem})
}

// BridgeHeader writes the C bridge header (.h) to w.
func EmitBridgeHeader(w io.Writer, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) error {
	return executeTemplate(w, "bridge_header_file", buildBridgeHeaderModel(framework, m, knownClasses))
}

// buildBridgeHeaderModel constructs the complete model for a _bridge.h file.
// All filtering and declaration building happens here; the template is a structural description.
func buildBridgeHeaderModel(framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) bridgeHeaderModel {
	ctx := m.BaseContext(framework.Framework, knownClasses)
	classNames := sortedStringKeys(framework.Classes)

	// ── Class method declarations ──────────────────────────────────────────────
	var methodDecls []bridgeDeclModel
	seenMethodDecl := map[string]bool{}
	for _, className := range classNames {
		cls := framework.Classes[className]
		if cls.Availability.IsUnavailable {
			continue
		}
		clsCtx := ctx
		clsCtx.ClassName = className
		clsCtx.GenericParams = cls.GenericParams
		bridgeNames := buildClassBridgeNames(framework.Framework, className, cls.Methods)
		for _, method := range cls.Methods {
			if shouldSkipBridgeMethod(method) {
				continue
			}
			if method.Availability.IsUnavailable {
				continue
			}
			cFunc := bridgeNames[methodKey(method.Selector, method.IsClassMethod)]
			if seenMethodDecl[cFunc] {
				continue
			}
			seenMethodDecl[cFunc] = true
			methodDecls = append(methodDecls, bridgeDeclModel{
				Entitlements: method.Availability.Entitlements,
				BridgeID:     naming.MethodBridgeID(framework.Framework, className, method.Selector, method.IsClassMethod),
				Decl:         bridgeFuncDecl(cFunc, method, clsCtx, m),
			})
		}
	}

	// ── Alloc helpers ──────────────────────────────────────────────────────────
	var allocHelpers []bridgeAllocModel
	for _, className := range classNames {
		cls := framework.Classes[className]
		if cls.Availability.IsUnavailable {
			continue
		}
		if hasDesignatedInitWithArgs(cls) {
			allocHelpers = append(allocHelpers, bridgeAllocModel{
				ClassName: className,
				FuncName:  allocBridgeName(framework.Framework, className),
			})
		}
	}

	// ── NSCoding declarations (pre-rendered; structure is fixed) ───────────────
	var codingBuf strings.Builder
	for _, className := range classNames {
		cls := framework.Classes[className]
		if cls.Availability.IsUnavailable || len(cls.GenericParams) > 0 || !classConformsToCoding(cls) {
			continue
		}
		writeCodingBridgeDecl(&codingBuf, framework.Framework, className)
	}

	// ── Free function declarations ─────────────────────────────────────────────
	var functionDecls []bridgeDeclModel
	seenFuncDecl := map[string]bool{}
	for _, fn := range framework.Functions {
		if fn.IsInline || fn.IsVariadic || fn.Availability.IsUnavailable ||
			strings.HasPrefix(fn.Name, "__builtin") || isUPPFunction(fn.Name) ||
			hasVAListArg(fn) || hasByValueUnknownType(fn) {
			continue
		}
		cFunc := strings.ToLower(framework.Framework) + "_fn_" + fn.Name
		if seenFuncDecl[cFunc] {
			continue
		}
		seenFuncDecl[cFunc] = true
		functionDecls = append(functionDecls, bridgeDeclModel{
			Entitlements: fn.Availability.Entitlements,
			BridgeID:     naming.FunctionBridgeID(framework.Framework, fn.Name),
			Decl:         bridgeFuncDeclFromFunction(cFunc, fn, ctx, m),
		})
	}

	// ── Foreign extension declarations ─────────────────────────────────────────
	var foreignDecls []bridgeDeclModel
	for _, className := range sortedStringKeys(framework.ForeignExtensions) {
		methods := framework.ForeignExtensions[className]
		clsCtx := ctx
		clsCtx.ClassName = className
		clsCtx.GenericParams = m.GenericParamIndex[className]
		bridgeNames := buildClassBridgeNames(framework.Framework, className, methods)
		for _, method := range methods {
			if method.Availability.IsUnavailable || shouldSkipBridgeMethod(method) {
				continue
			}
			cFunc := bridgeNames[methodKey(method.Selector, method.IsClassMethod)]
			foreignDecls = append(foreignDecls, bridgeDeclModel{
				Entitlements: method.Availability.Entitlements,
				BridgeID:     naming.MethodBridgeID(framework.Framework, className, method.Selector, method.IsClassMethod),
				Decl:         bridgeFuncDecl(cFunc, method, clsCtx, m),
			})
		}
	}

	// ── Protocol proxy declarations ────────────────────────────────────────────
	var protoDecls []bridgeProtoDeclModel
	for _, protoName := range sortedStringKeys(framework.Protocols) {
		proto := framework.Protocols[protoName]
		if proto.Availability.IsUnavailable {
			continue
		}
		goProtoName := naming.ProtocolGoTypeName(protoName, m.OwnerIndex)
		idTypeName := goProtoName + "IDProtocol"
		bridgeNames := buildClassBridgeNames(framework.Framework, idTypeName, proto.Methods)
		for _, method := range proto.Methods {
			if shouldSkipBridgeMethod(method) || method.Availability.IsUnavailable || method.IsClassMethod {
				continue
			}
			cFunc := bridgeNames[methodKey(method.Selector, false)]
			if cFunc == "" || seenMethodDecl[cFunc] {
				continue
			}
			seenMethodDecl[cFunc] = true
			protoDecls = append(protoDecls, bridgeProtoDeclModel{
				BridgeID: naming.MethodBridgeID(framework.Framework, idTypeName, method.Selector, false),
				RetType:  bridgeReturnCType(method, ctx, m),
				FuncName: cFunc,
				Params:   bridgeParamList(false, method.Params, method.IsNSError, ctx, m),
			})
		}
	}

	return bridgeHeaderModel{
		MethodDecls:   methodDecls,
		AllocHelpers:  allocHelpers,
		CodingDecls:   codingBuf.String(),
		FunctionDecls: functionDecls,
		ForeignDecls:  foreignDecls,
		ProtoDecls:    protoDecls,
	}
}

// isUPPFunction returns true for Carbon Universal Procedure Pointer functions
// (Invoke*UPP, New*UPP, Dispose*UPP). These use macros that dereference the
// UPP void* argument as a function pointer — which fails in a -fno-objc-arc
// bridge context where all pointers are void*. Skip them entirely.
func isUPPFunction(name string) bool {
	return (strings.HasPrefix(name, "Invoke") || strings.HasPrefix(name, "New") || strings.HasPrefix(name, "Dispose")) &&
		strings.HasSuffix(name, "UPP")
}

// hasVAListArg returns true if any argument in the function is a va_list type.
func hasVAListArg(fn meta.Function) bool {
	for _, arg := range fn.Params {
		if isVAList(arg.ObjCType) {
			return true
		}
	}
	return false
}

// BridgeImpl writes the ObjC bridge implementation (.m) to w.
func EmitBridgeImpl(w io.Writer, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, headerName string) error {
	return executeTemplate(w, "bridge_impl_file", buildBridgeImplModel(framework, m, knownClasses, headerName))
}

// buildBridgeImplModel constructs the complete model for a _bridge.m file.
// For each method, buildBridgeTryBody pre-renders the @try block content so the
// template shows only the @try/@catch wrapper structure.
func buildBridgeImplModel(framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, headerName string) bridgeImplModel {
	ctx := m.BaseContext(framework.Framework, knownClasses)
	classNames := sortedStringKeys(framework.Classes)

	// ── Class method implementations ───────────────────────────────────────────
	var methods []bridgeImplMethodModel
	seenMethodImpl := map[string]bool{}
	for _, className := range classNames {
		cls := framework.Classes[className]
		if cls.Availability.IsUnavailable {
			continue
		}
		clsCtx := ctx
		clsCtx.ClassName = className
		clsCtx.GenericParams = cls.GenericParams
		bridgeNames := buildClassBridgeNames(framework.Framework, className, cls.Methods)
		for _, method := range cls.Methods {
			if shouldSkipBridgeMethod(method) || method.Availability.IsUnavailable {
				continue
			}
			cFunc := bridgeNames[methodKey(method.Selector, method.IsClassMethod)]
			if seenMethodImpl[cFunc] {
				continue
			}
			seenMethodImpl[cFunc] = true

			var target string
			if method.IsClassMethod {
				target = className
			} else {
				target = fmt.Sprintf("(__bridge %s *)self", className)
			}
			methods = append(methods, buildBridgeImplMethod(
				framework.Framework, className, method.Selector, method.IsClassMethod,
				cFunc, target, method, clsCtx, m,
			))
		}
	}

	// ── Alloc helpers ──────────────────────────────────────────────────────────
	var allocImpls []bridgeAllocImplModel
	for _, className := range classNames {
		cls := framework.Classes[className]
		if cls.Availability.IsUnavailable {
			continue
		}
		if hasDesignatedInitWithArgs(cls) {
			allocImpls = append(allocImpls, bridgeAllocImplModel{
				ClassName: className,
				FuncName:  allocBridgeName(framework.Framework, className),
			})
		}
	}

	// ── NSCoding implementations (pre-rendered; body is always the same) ───────
	var codingBuf strings.Builder
	for _, className := range classNames {
		cls := framework.Classes[className]
		if cls.Availability.IsUnavailable || len(cls.GenericParams) > 0 || !classConformsToCoding(cls) {
			continue
		}
		writeCodingBridgeImpl(&codingBuf, framework.Framework, className)
	}

	// ── Free function implementations ─────────────────────────────────────────
	var functions []bridgeImplMethodModel
	seenFuncImpl := map[string]bool{}
	for _, fn := range framework.Functions {
		if fn.IsInline || fn.IsVariadic || fn.Availability.IsUnavailable ||
			strings.HasPrefix(fn.Name, "__builtin") || isUPPFunction(fn.Name) ||
			hasVAListArg(fn) || hasByValueUnknownType(fn) {
			continue
		}
		cFunc := strings.ToLower(framework.Framework) + "_fn_" + fn.Name
		if seenFuncImpl[cFunc] {
			continue
		}
		seenFuncImpl[cFunc] = true
		functions = append(functions, buildBridgeFuncImplMethod(fn, cFunc, ctx, m))
	}

	// ── Foreign extension implementations ──────────────────────────────────────
	var foreignMethods []bridgeImplMethodModel
	for _, className := range sortedStringKeys(framework.ForeignExtensions) {
		methods := framework.ForeignExtensions[className]
		clsCtx := ctx
		clsCtx.ClassName = className
		clsCtx.GenericParams = m.GenericParamIndex[className]
		bridgeNames := buildClassBridgeNames(framework.Framework, className, methods)
		for _, method := range methods {
			if method.Availability.IsUnavailable || shouldSkipBridgeMethod(method) {
				continue
			}
			cFunc := bridgeNames[methodKey(method.Selector, method.IsClassMethod)]
			var target string
			if method.IsClassMethod {
				target = className
			} else {
				target = fmt.Sprintf("(__bridge %s *)self", className)
			}
			foreignMethods = append(foreignMethods, buildBridgeImplMethod(
				framework.Framework, className, method.Selector, method.IsClassMethod,
				cFunc, target, method, clsCtx, m,
			))
		}
	}

	// ── Protocol proxy implementations ─────────────────────────────────────────
	var protoMethods []bridgeImplMethodModel
	seenProtoImpl := map[string]bool{}
	for _, protoName := range sortedStringKeys(framework.Protocols) {
		proto := framework.Protocols[protoName]
		if proto.Availability.IsUnavailable {
			continue
		}
		goProtoName := naming.ProtocolGoTypeName(protoName, m.OwnerIndex)
		idTypeName := goProtoName + "IDProtocol"
		bridgeNames := buildClassBridgeNames(framework.Framework, idTypeName, proto.Methods)
		for _, method := range proto.Methods {
			if shouldSkipBridgeMethod(method) || method.Availability.IsUnavailable || method.IsClassMethod {
				continue
			}
			cFunc := bridgeNames[methodKey(method.Selector, false)]
			if cFunc == "" || seenProtoImpl[cFunc] {
				continue
			}
			seenProtoImpl[cFunc] = true
			target := fmt.Sprintf("(id<%s>)self", protoName)
			protoMethods = append(protoMethods, buildBridgeImplMethod(
				framework.Framework, idTypeName, method.Selector, false,
				cFunc, target, method, ctx, m,
			))
		}
	}

	return bridgeImplModel{
		Framework:       framework.Framework,
		ParentFramework: framework.ParentFramework,
		UmbrellaHeader:  framework.Header,
		HeaderName:      headerName,
		Methods:         methods,
		AllocImpls:      allocImpls,
		CodingImpls:     codingBuf.String(),
		Functions:       functions,
		ForeignMethods:  foreignMethods,
		ProtoMethods:    protoMethods,
	}
}

// buildBridgeImplMethod builds the model for one ObjC method bridge implementation.
// target is the ObjC message receiver expression (class name or `(__bridge X *)self`).
func buildBridgeImplMethod(framework, className, selector string, isClassMethod bool, cFunc, target string, method meta.Method, ctx typemap.Context, m *typemap.Mapper) bridgeImplMethodModel {
	retC := bridgeReturnCType(method, ctx, m)
	objcCall := buildObjCCall(target, method, ctx, m)
	// Multi-keyword format-variadic methods (e.g. initWithFormat:options:locale:) cannot
	// use the @"%@" trick because variadic args must come last in ObjC message syntax.
	// Instead, the template wraps the @try body with -Wformat-security pragmas.
	// The @"%@" trick IS used when the format keyword is the LAST selector part
	// (e.g. initWithFormat:, raise:format:) — handled in buildObjCCall.
	needsFormatPragma := false
	if isFormatStringVariadic(method) {
		selParts := strings.Split(selector, ":")
		if len(selParts) > 0 && selParts[len(selParts)-1] == "" {
			selParts = selParts[:len(selParts)-1]
		}
		if len(selParts) > 0 {
			lastPart := selParts[len(selParts)-1]
			needsFormatPragma = !strings.Contains(strings.ToLower(lastPart), "format")
		}
	}
	return bridgeImplMethodModel{
		Entitlements:      method.Availability.Entitlements,
		BridgeID:          naming.MethodBridgeID(framework, className, selector, isClassMethod),
		RetType:           retC,
		FuncName:          cFunc,
		Params:            bridgeParamList(isClassMethod, method.Params, method.IsNSError, ctx, m),
		IsNSError:        method.IsNSError,
		NeedsFormatPragma: needsFormatPragma,
		TryBody:           buildBridgeTryBody(objcCall, method.Return, method.IsNSError, retC, ctx, m),
		CatchReturn:       bridgeCatchReturn(retC),
	}
}

// buildBridgeFuncImplMethod builds the model for one C free-function bridge implementation.
func buildBridgeFuncImplMethod(fn meta.Function, cFunc string, ctx typemap.Context, m *typemap.Mapper) bridgeImplMethodModel {
	retC := m.CType(fn.Return.ObjCType, ctx, nil)
	if fn.Return.ObjCType == "" || fn.Return.ObjCType == "void" {
		retC = "void"
	}
	objcCall := buildFreeFuncCall(fn, ctx, m)
	// Free functions never have NSError — the exception mechanism is used instead.
	tryBody := buildBridgeTryBody(objcCall, fn.Return, false, retC, ctx, m)
	return bridgeImplMethodModel{
		Entitlements: fn.Availability.Entitlements,
		BridgeID:     naming.FunctionBridgeID(ctx.Framework, fn.Name),
		RetType:      retC,
		FuncName:     cFunc,
		Params:       freeFuncParamList(fn.Params, ctx, m),
		IsNSError:   false,
		TryBody:      tryBody,
		CatchReturn:  bridgeCatchReturn(retC),
	}
}

// buildBridgeTryBody pre-renders the @try block body for one bridge function.
// The multi-path dispatch (void / object / struct-by-value / primitive + NSError variants)
// lives here so the template shows only the @try/@catch wrapper structure.
func buildBridgeTryBody(objcCall string, ret meta.ReturnType, hasNSError bool, retC string, ctx typemap.Context, m *typemap.Mapper) string {
	var sb strings.Builder
	isVoid := ret.ObjCType == "" || typemap.IsVoid(ret.ObjCType)
	isInstancetype := ret.IsInstancetype || typemap.IsInstancetype(ret.ObjCType)

	switch {
	case isVoid && !hasNSError:
		fmt.Fprintf(&sb, "%s;", objcCall)

	case isVoid && hasNSError:
		fmt.Fprintf(&sb, "%s;", objcCall)
		fmt.Fprintf(&sb, "\n            if (outError) *outError = (__bridge void *)[_err retain];")

	case isInstancetype || isObjectObjCType(ret.ObjCType, ctx.ClassNameIndex):
		fmt.Fprintf(&sb, "id _result = %s;", objcCall)
		if hasNSError {
			fmt.Fprintf(&sb, "\n            if (outError) *outError = (__bridge void *)[_err retain];")
		}
		if ret.IsAlreadyRetained {
			fmt.Fprintf(&sb, "\n            return (__bridge id)_result;")
		} else {
			fmt.Fprintf(&sb, "\n            return (__bridge id)[_result retain];")
		}

	case typemap.IsSEL(typemap.Normalise(ret.ObjCType)):
		// SEL is an opaque type; casting it directly to const char* is deprecated.
		// sel_getName converts it safely to the selector's UTF-8 name.
		fmt.Fprintf(&sb, "SEL _selResult = %s;", objcCall)
		fmt.Fprintf(&sb, "\n            const char * _result = sel_getName(_selResult);")
		if hasNSError {
			fmt.Fprintf(&sb, "\n            if (outError) *outError = (__bridge void *)[_err retain];")
		}
		fmt.Fprintf(&sb, "\n            return _result;")

	default:
		structTypeName := typemap.Normalise(ret.ObjCType)
		if isStructByValueType(structTypeName, ctx, m) {
			fmt.Fprintf(&sb, "%s *_result = (%s *)malloc(sizeof(%s));", structTypeName, structTypeName, structTypeName)
			fmt.Fprintf(&sb, "\n            *_result = %s;", objcCall)
			if hasNSError {
				fmt.Fprintf(&sb, "\n            if (outError) *outError = (__bridge void *)[_err retain];")
			}
			fmt.Fprintf(&sb, "\n            return (void *)_result;")
		} else {
			fmt.Fprintf(&sb, "%s _result = (%s)%s;", retC, retC, objcCall)
			if hasNSError {
				fmt.Fprintf(&sb, "\n            if (outError) *outError = (__bridge void *)[_err retain];")
			}
			fmt.Fprintf(&sb, "\n            return _result;")
		}
	}
	return sb.String()
}

// hasDesignatedInitWithArgs reports whether the class has at least one designated
// initializer with non-zero arguments (bare -[init] needs no alloc bridge helper).
func hasDesignatedInitWithArgs(cls meta.Class) bool {
	for _, m := range cls.Methods {
		if m.IsDesignatedInit && m.IsInit && len(m.Params) > 0 && !m.Availability.IsUnavailable {
			return true
		}
	}
	return false
}

// allocBridgeName returns the C function name for the per-class alloc helper.
func allocBridgeName(framework, className string) string {
	return strings.ToLower(framework) + "_" + className + "_alloc"
}

// writeCodingBridgeDecl writes the C header declarations for the NSCoding
// convenience bridge functions of a single class.
func writeCodingBridgeDecl(w io.Writer, framework, className string) {
	fw := strings.ToLower(framework)
	fmt.Fprintf(w, "// NSCoding convenience: %s.SerializeToArchive\n", className)
	fmt.Fprintf(w, "void* %s_%s_serializeToArchive(void *self, size_t *outLen, void **outError, void **outException);\n", fw, className)
	fmt.Fprintf(w, "// NSCoding convenience: New%sFromArchive\n", className)
	fmt.Fprintf(w, "id %s_%s_newFromArchive(void *data, size_t dataLen, void **outError, void **outException);\n\n", fw, className)
}

// writeCodingBridgeImpl writes the ObjC implementations for the NSCoding
// convenience bridge functions of a single class.
func writeCodingBridgeImpl(w io.Writer, framework, className string) {
	fw := strings.ToLower(framework)
	serializeFn := fw + "_" + className + "_serializeToArchive"
	deserializeFn := fw + "_" + className + "_newFromArchive"

	fmt.Fprintf(w, "// NSCoding convenience: %s.SerializeToArchive\n", className)
	fmt.Fprintf(w, "void* %s(void *self, size_t *outLen, void **outError, void **outException) {\n", serializeFn)
	fmt.Fprintf(w, "    @autoreleasepool {\n")
	fmt.Fprintf(w, "        @try {\n")
	fmt.Fprintf(w, "            NSError *_err = nil;\n")
	fmt.Fprintf(w, "            NSData *_data = [NSKeyedArchiver archivedDataWithRootObject:(__bridge id)self requiringSecureCoding:YES error:&_err];\n")
	fmt.Fprintf(w, "            if (outError && _err) *outError = (__bridge void *)[_err retain];\n")
	fmt.Fprintf(w, "            if (!_data) { *outLen = 0; return NULL; }\n")
	fmt.Fprintf(w, "            NSUInteger len = _data.length;\n")
	fmt.Fprintf(w, "            void *buf = malloc(len);\n")
	fmt.Fprintf(w, "            memcpy(buf, _data.bytes, len);\n")
	fmt.Fprintf(w, "            *outLen = len;\n")
	fmt.Fprintf(w, "            return buf;\n")
	fmt.Fprintf(w, "        } @catch (NSException *_ex) {\n")
	fmt.Fprintf(w, "            if (outException) *outException = (__bridge void *)[_ex retain];\n")
	fmt.Fprintf(w, "            *outLen = 0;\n")
	fmt.Fprintf(w, "            return NULL;\n")
	fmt.Fprintf(w, "        }\n")
	fmt.Fprintf(w, "    }\n")
	fmt.Fprintf(w, "}\n\n")

	fmt.Fprintf(w, "// NSCoding convenience: New%sFromArchive\n", className)
	fmt.Fprintf(w, "id %s(void *data, size_t dataLen, void **outError, void **outException) {\n", deserializeFn)
	fmt.Fprintf(w, "    @autoreleasepool {\n")
	fmt.Fprintf(w, "        @try {\n")
	fmt.Fprintf(w, "            NSError *_err = nil;\n")
	fmt.Fprintf(w, "            NSData *_nsData = [NSData dataWithBytes:data length:dataLen];\n")
	fmt.Fprintf(w, "            id _result = [NSKeyedUnarchiver unarchivedObjectOfClass:[%s class] fromData:_nsData error:&_err];\n", className)
	fmt.Fprintf(w, "            if (outError && _err) *outError = (__bridge void *)[_err retain];\n")
	fmt.Fprintf(w, "            if (!_result) return NULL;\n")
	fmt.Fprintf(w, "            return (__bridge id)[_result retain];\n")
	fmt.Fprintf(w, "        } @catch (NSException *_ex) {\n")
	fmt.Fprintf(w, "            if (outException) *outException = (__bridge void *)[_ex retain];\n")
	fmt.Fprintf(w, "            return NULL;\n")
	fmt.Fprintf(w, "        }\n")
	fmt.Fprintf(w, "    }\n")
	fmt.Fprintf(w, "}\n\n")
}

// buildClassBridgeNames returns a map from methodKey → final C function name
// for every non-variadic method in the class, with selector collisions resolved.
//
// A collision occurs when two selectors have the same base name but different
// arities (e.g. "open" and "open:") — they map to the same raw bridge name.
// When that happens the argument count is appended ("_0" / "_1") to both names.
//
// The map key is methodKey(selector, isClassMethod) — a unique string per ObjC
// method. Using raw bridge names as keys would fail because two colliding selectors
// share the same raw name and would overwrite each other in the map.
func buildClassBridgeNames(framework, className string, methods []meta.Method) map[string]string {
	// First pass: raw name → count of methods that map to it.
	rawCount := make(map[string]int)
	for _, method := range methods {
		if shouldSkipBridgeMethod(method) {
			continue
		}
		raw := naming.BridgeFuncName(framework, className, method.Selector, method.IsClassMethod)
		rawCount[raw]++
	}

	// Second pass: assign final names, keyed by unique (selector, isClass) pair.
	result := make(map[string]string, len(methods))
	for _, method := range methods {
		if shouldSkipBridgeMethod(method) {
			continue
		}
		raw := naming.BridgeFuncName(framework, className, method.Selector, method.IsClassMethod)
		final := raw
		if rawCount[raw] > 1 {
			nArgs := strings.Count(method.Selector, ":")
			final = fmt.Sprintf("%s_%d", raw, nArgs)
		}
		result[methodKey(method.Selector, method.IsClassMethod)] = final
	}
	return result
}

// methodKey returns a string that is unique for each (selector, isClassMethod) pair.
// ObjC guarantees at most one instance method and at most one class method per selector per class.
func methodKey(selector string, isClassMethod bool) string {
	if isClassMethod {
		return "c:" + selector
	}
	return "i:" + selector
}

// bridgeFuncDecl builds the C function declaration for a method.
func bridgeFuncDecl(cFunc string, method meta.Method, ctx typemap.Context, m *typemap.Mapper) string {
	retC := bridgeReturnCType(method, ctx, m)
	params := bridgeParamList(method.IsClassMethod, method.Params, method.IsNSError, ctx, m)
	return fmt.Sprintf("%s %s(%s)", retC, cFunc, params)
}

// bridgeFuncDeclFromFunction builds the C function declaration for a plain C function.
func bridgeFuncDeclFromFunction(cFunc string, fn meta.Function, ctx typemap.Context, m *typemap.Mapper) string {
	retC := m.CType(fn.Return.ObjCType, ctx, nil)
	if fn.Return.ObjCType == "" || fn.Return.ObjCType == "void" {
		retC = "void"
	}
	// Free functions: no self arg, no NSError out-param.
	params := freeFuncParamList(fn.Params, ctx, m)
	return fmt.Sprintf("%s %s(%s)", retC, cFunc, params)
}

// freeFuncParamList builds a C parameter list for a plain C function (no self).
// Like bridgeParamList, a trailing void **outException is always appended.
func freeFuncParamList(args []meta.Param, ctx typemap.Context, m *typemap.Mapper) string {
	resolved := buildParamNames(args)
	var parts []string
	for i, arg := range args {
		// Skip va_list arguments — cannot be expressed in the Go bridge.
		if isVAList(arg.ObjCType) {
			continue
		}
		cType := m.CType(arg.ObjCType, ctx, nil)
		parts = append(parts, fmt.Sprintf("%s %s", cType, resolved[i]))
	}
	parts = append(parts, "void **outException")
	return strings.Join(parts, ", ")
}

// isVAList returns true for va_list ObjC argument types.
func isVAList(objcType string) bool {
	n := typemap.Normalise(objcType)
	return strings.Contains(n, "va_list") || strings.Contains(n, "__va_list")
}

// hasByValueUnknownType returns true when any argument or return type is a
// C value type (no '*') that the typemap cannot represent as void* — specifically
// SIMD vector types (vFloat, vDouble) and structs passed by value (DenseMatrix_Float).
// The compiler cannot cast a pointer to these types, producing build errors.
//
// This check is intentionally narrow: it allows through pointer typedefs (recognised
// by _Nonnull / _Nullable nullability annotations — only valid on pointer types),
// enums, and any named type that the mapper would handle via its enum or CF tables.
// Entitlement-gated APIs (vmnet, NetworkExtension, etc.) use pointer typedefs and
// integer enums — they must NOT be filtered here.
func hasByValueUnknownType(fn meta.Function) bool {
	check := func(objcType string) bool {
		if objcType == "" || objcType == "void" {
			return false
		}
		// Pointer types (explicit * or nullability-annotated pointer typedefs).
		if strings.Contains(objcType, "*") {
			return false
		}
		// Nullability annotations (_Nonnull, _Nullable, _Null_unspecified) are only
		// valid on pointer types — a bare typedef with one of these is a pointer.
		if strings.Contains(objcType, "_Nonnull") ||
			strings.Contains(objcType, "_Nullable") ||
			strings.Contains(objcType, "_Null_unspecified") {
			return false
		}
		n := typemap.Normalise(objcType)
		// Known scalar / ObjC meta types.
		if typemap.IsBOOL(n) || typemap.IsID(n) || typemap.IsSEL(n) || typemap.IsClass(n) {
			return false
		}
		if isVAList(n) || typemap.IsBlock(n) {
			return false
		}
		// Standard C scalar types.
		switch n {
		case "void", "bool", "_Bool",
			"char", "signed char", "unsigned char",
			"short", "unsigned short",
			"int", "unsigned int",
			"long", "unsigned long",
			"long long", "unsigned long long",
			"float", "double", "long double",
			"int8_t", "int16_t", "int32_t", "int64_t",
			"uint8_t", "uint16_t", "uint32_t", "uint64_t",
			"size_t", "ssize_t", "ptrdiff_t", "intptr_t", "uintptr_t",
			"NSInteger", "NSUInteger", "CGFloat":
			return false
		}
		// Named types without a nullability annotation and without * fall into two
		// categories:
		//   (a) Enum / integer typedefs — safe, CType() maps them to the right int type.
		//   (b) SIMD vector / struct-by-value types — unsafe.
		// We conservatively allow any type whose normalised name contains "Ref" or
		// ends in "_t" (common C typedef conventions for pointer/integer types), and
		// any type that looks like an enum (e.g. vmnet_return_t, FFTDirection).
		// Types known to be SIMD/vector (e.g. vFloat, vDouble, DenseMatrix_Float,
		// DenseVector_Double) have neither characteristic.
		if strings.HasSuffix(n, "_t") || strings.Contains(n, "Ref") {
			return false
		}
		// Enum-like names: contain no spaces and are not a known struct prefix.
		// Struct-by-value types from vecLib/BNNS look like: DenseMatrix_Float,
		// DenseVector_Double, BNNSNearestNeighbors, vFloat, vDouble, etc.
		// These all lack "_t"/"Ref" and represent value types.
		return true
	}
	if check(fn.Return.ObjCType) {
		return true
	}
	for _, arg := range fn.Params {
		if check(arg.ObjCType) {
			return true
		}
	}
	return false
}

// bridgeReturnCType computes the C return type for a bridge function.
// ObjC object pointer returns are declared as id (the semantically correct ObjC type)
// rather than void *. Primitive, enum, and struct-by-value returns use their concrete
// C types. Genuinely opaque returns (blocks, malloc'd structs) remain void *.
func bridgeReturnCType(method meta.Method, ctx typemap.Context, m *typemap.Mapper) string {
	ret := method.Return
	if ret.IsInstancetype || typemap.IsInstancetype(ret.ObjCType) {
		return "id"
	}
	if ret.ObjCType == "" || typemap.IsVoid(ret.ObjCType) {
		return "void"
	}
	cType := m.CType(ret.ObjCType, ctx, nil)
	if cType == "void *" && isObjectObjCType(ret.ObjCType, ctx.ClassNameIndex) {
		return "id"
	}
	return cType
}

// bridgeParamList builds the C parameter list for a bridge function.
// Every generated bridge function receives a trailing void **outException parameter
// so that ObjC exceptions caught in the @try/@catch wrapper can be returned to Go.
// Block args are passed as void* — the Go side creates a GoBlock via MakeBlock_*.
func bridgeParamList(isClassMethod bool, args []meta.Param, hasNSError bool, ctx typemap.Context, m *typemap.Mapper) string {
	resolved := buildParamNames(args)
	var parts []string
	if !isClassMethod {
		parts = append(parts, "void *self")
	}

	for i, arg := range args {
		if arg.IsBlock {
			parts = append(parts, fmt.Sprintf("void *%s", resolved[i]))
			continue
		}
		cType := m.CType(arg.ObjCType, ctx, nil)
		parts = append(parts, fmt.Sprintf("%s %s", cType, resolved[i]))
	}
	if hasNSError {
		parts = append(parts, "void **outError")
	}
	parts = append(parts, "void **outException")
	return strings.Join(parts, ", ")
}

// bridgeCatchReturn returns the zero-value return statement for the @catch block,
// or an empty string for void-returning functions (no return needed).
func bridgeCatchReturn(retC string) string {
	switch retC {
	case "void", "":
		return ""
	case "void *", "id":
		return "return nil;"
	case "bool":
		return "return false;"
	default:
		return "return 0;"
	}
}

// writeMethodImpl writes the ObjC bridge function implementation for a method.
// The body is wrapped in @try/@catch so ObjC exceptions are caught and returned
// through the trailing void **outException out-parameter rather than unwinding
// through CGo into the Go runtime (which would crash).
// writeBridgeEntitlements emits "// Requires entitlement: <key>" lines to w.
// Called before the genbind comment and declaration/implementation in .h and .m.
func writeBridgeEntitlements(w io.Writer, keys []string) {
	for _, k := range keys {
		fmt.Fprintf(w, "// Requires entitlement: %s\n", k)
	}
}

// writeFunctionImpl is retained as an adapter for tests that call it directly.
// It delegates to the model-based approach so tests continue to exercise the same code path.
func writeFunctionImpl(w io.Writer, cFunc string, fn meta.Function, ctx typemap.Context, m *typemap.Mapper) {
	model := buildBridgeFuncImplMethod(fn, cFunc, ctx, m)
	_ = executeTemplate(w, "bridge_impl_method", model)
}

// buildFreeFuncCall builds the C function call expression for a plain C function.
// Block args arrive as void* from CGo and are cast back to their concrete block type.
func buildFreeFuncCall(fn meta.Function, ctx typemap.Context, m *typemap.Mapper) string {
	resolved := buildParamNames(fn.Params)
	var argParts []string
	for i, arg := range fn.Params {
		if isVAList(arg.ObjCType) {
			continue
		}
		n := typemap.Normalise(arg.ObjCType)
		if isStructByValueType(n, ctx, m) {
			cls := typemap.ClassName(n)
			if cls == "" {
				cls = n
			}
			argParts = append(argParts, fmt.Sprintf("*(%s*)%s", cls, resolved[i]))
		} else if typemap.IsBlock(n) {
			cleanType := typemap.NormaliseBlock(arg.ObjCType)
			argParts = append(argParts, fmt.Sprintf("(%s)%s", cleanType, resolved[i]))
		} else if target, ok := m.TypedefIndex[n]; ok && typemap.IsBlock(target) {
			argParts = append(argParts, fmt.Sprintf("(%s)%s", n, resolved[i]))
		} else {
			argParts = append(argParts, resolved[i])
		}
	}
	return fmt.Sprintf("%s(%s)", fn.Name, strings.Join(argParts, ", "))
}

// buildObjCCall constructs the Objective-C message expression for a method.
func buildObjCCall(target string, method meta.Method, ctx typemap.Context, m *typemap.Mapper) string {
	selector := method.Selector
	parts := strings.Split(selector, ":")
	// Remove trailing empty element (selector ends with ':').
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}

	if len(parts) == 0 {
		return fmt.Sprintf("[%s %s]", target, selector)
	}

	if len(method.Params) == 0 && !strings.Contains(selector, ":") {
		// No-arg selector: [target selector]
		return fmt.Sprintf("[%s %s]", target, selector)
	}

	// Build keyword message: [target key0:arg0 key1:arg1 ...]
	// For NSError**, add the _err local variable.
	resolved := buildParamNames(method.Params)
	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(target)
	sb.WriteString(" ")

	argIdx := 0
	for i, part := range parts {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(part)
		sb.WriteString(":")

		if method.IsNSError && argIdx == len(method.Params) {
			// This is the elided NSError** arg — inject _err.
			sb.WriteString("&_err")
		} else if argIdx < len(method.Params) {
			arg := method.Params[argIdx]
			if arg.IsBlock {
				// Cast void* trampoline to the ObjC block type for the message send.
				// Use NormaliseBlock (strips nullability but keeps const) so that block
				// pointer types with const-qualified parameters remain compatible.
				// Replace any ObjC generic type params (e.g. ObjectType) with id —
				// the C compiler does not know these type parameter names.
				// Then strip <id> so constrained generic types (e.g.
				// NSFetchRequest<id<NSFetchRequestResult>>) do not fail bounds checking.
				cleanType := typemap.NormaliseBlock(arg.ObjCType)
				for _, gp := range ctx.GenericParams {
					cleanType = strings.ReplaceAll(cleanType, gp, "id")
				}
				cleanType = stripGenericID(cleanType)
				fmt.Fprintf(&sb, "(__bridge %s)%s", cleanType, resolved[argIdx])
			} else {
				castExpr := objcArgCast(resolved[argIdx], arg.ObjCType, ctx, m)
				// Format-variadic methods (e.g. initWithFormat:, stringWithFormat:) declare
				// their format argument with NS_FORMAT_FUNCTION. When the format keyword is
				// the LAST keyword in the selector, use @"%@" as the literal format string
				// and pass the Go-provided string as the sole variadic argument. This
				// prevents ObjC from interpreting the pre-formatted Go string as a format
				// template. Callers must pre-format using fmt.Sprintf on the Go side.
				// When the format keyword is not last (e.g. initWithFormat:options:locale:),
				// the variadic args cannot be injected mid-message; format-security
				// is suppressed via pragma on the outer call in those cases.
				if isFormatStringVariadic(method) && strings.Contains(strings.ToLower(part), "format") && i == len(parts)-1 {
					sb.WriteString(`@"%@", `)
				}
				sb.WriteString(castExpr)
			}
		}
		argIdx++
	}

	sb.WriteString("]")
	return sb.String()
}

// isStructByValueType reports whether the given ObjC type name names a C
// struct that is passed/returned by value. Three sources contribute:
//
//   - The hardcoded structCTypes set (CGRect, NSRange, …) — covers stable
//     SDK structs that exist regardless of which framework's metadata loaded.
//   - The StructIndex registry — covers any struct discovered via metadata
//     (MTLViewport, NSOperatingSystemVersion, …).
//   - TypedefIndex whose target starts with "struct " — covers the common
//     `typedef struct {…} Name;` pattern (MKMapRect, ether_addr_t, …) where
//     Clang emits the struct anonymously and the field info reaches the
//     metadata only via the typedef.
func isStructByValueType(n string, ctx typemap.Context, m *typemap.Mapper) bool {
	if typemap.IsStructCType(n) {
		return true
	}
	// "struct foo" explicit form — strip the keyword and check the bare name.
	// Clang sometimes records ObjC types with the "struct" prefix (e.g. "struct timespec").
	// Reject pointer-to-struct ("struct foo *") — those are pointer params, not by-value.
	if strings.HasPrefix(n, "struct ") {
		bare := strings.TrimSpace(n[7:])
		if strings.HasSuffix(bare, "*") {
			return false
		}
		if typemap.IsStructCType(bare) || m.StructIndex[bare] != "" {
			return true
		}
	}
	// Pointer-to-struct parameters (X *) are NOT passed by value — only bare
	// struct types (X) are. The StructIndex and TypedefIndex checks below
	// strip the * via ClassName, which would incorrectly match pointer params
	// like CFAllocatorContext * against their struct registry entries.
	if typemap.IsPointer(n) {
		return false
	}
	// "id" is an ObjC intrinsic (typedef struct objc_object *id) — it is always
	// an ObjC object pointer, never a by-value struct.
	if typemap.IsID(n) {
		return false
	}
	cls := typemap.ClassName(n)
	if cls == "" {
		cls = n
	}
	if m.StructIndex[cls] != "" {
		return true
	}
	// Walk the typedef chain (up to 4 levels) to handle cases like
	// SCNQuaternion → SCNVector4 → struct SCNVector4.
	// A typedef that resolves to a pointer-to-struct ("struct X *") is a pointer
	// type, not a by-value struct — skip those entries.
	seen := map[string]bool{}
	cur := cls
	for i := 0; i < 4; i++ {
		if seen[cur] {
			break
		}
		seen[cur] = true
		target, ok := m.TypedefIndex[cur]
		if !ok {
			break
		}
		// "struct X *" or "union X *" — pointer-to-struct, not by-value.
		if (strings.HasPrefix(target, "struct ") || strings.HasPrefix(target, "union ")) && strings.HasSuffix(strings.TrimSpace(target), "*") {
			return false
		}
		if strings.HasPrefix(target, "struct ") || strings.HasPrefix(target, "union ") {
			return true
		}
		if typemap.IsStructCType(target) {
			return true
		}
		// Strip "struct "/"union " prefix for next iteration, or continue with bare name.
		bare := target
		if idx := strings.Index(bare, " "); idx >= 0 {
			bare = bare[idx+1:]
		}
		if m.StructIndex[bare] != "" {
			return true
		}
		cur = bare
	}
	return false
}

// objcArgCast wraps a C parameter in the appropriate ObjC cast for the message send.
func objcArgCast(argName, objcType string, ctx typemap.Context, m *typemap.Mapper) string {
	n := typemap.Normalise(objcType)
	if typemap.IsBlock(n) {
		// Block args arrive as void*; cast to the concrete ObjC block type.
		// Use NormaliseBlock (preserves const) for compatibility with const-typed params.
		return "(__bridge " + typemap.NormaliseBlock(objcType) + ")" + argName
	}
	// SEL: bridge receives const char*; convert to a properly interned selector.
	if typemap.IsSEL(n) {
		return fmt.Sprintf("sel_getUid(%s)", argName)
	}
	// ObjC object pointer: cast void* back to object.
	// bare "id" is an ObjC pointer without a "*" in the type string — handle it first.
	if typemap.IsID(n) {
		return fmt.Sprintf("(__bridge id)%s", argName)
	}
	if typemap.IsPointer(n) && !typemap.IsDoublePointer(n) {
		cls := typemap.ClassName(n)
		if cls != "" {
			return fmt.Sprintf("(__bridge id)%s", argName)
		}
	}
	// NSError** → the bridge receives a void*; cast it to NSError** for the call.
	// (When HasNSError is set, the last arg is elided and &_err is injected by
	// buildObjCCall instead; this path handles mid-selector NSError** args.)
	if typemap.IsDoublePointer(n) && strings.Contains(n, "NSError") {
		return "(NSError **)" + argName
	}
	// Known C struct (hardcoded set, metadata-registered, or typedef-to-struct):
	// the bridge receives void* pointing to the struct value; dereference it
	// for the ObjC message send. Uses the shared isStructByValueType helper
	// so the same set of types drive both arg-cast and return-malloc paths.
	if isStructByValueType(n, ctx, m) {
		cls := typemap.ClassName(n)
		if cls == "" {
			cls = n
		}
		return fmt.Sprintf("*(%s*)%s", cls, argName)
	}
	// Primitive: no cast needed.
	return argName
}

// isObjectObjCType returns true if the ObjC qualType represents an ObjC object.
// knownClasses is consulted first; the capital-letter heuristic is a fallback for
// types not present in the registry (e.g. forward-declared classes).
// Underscore-prefixed names (e.g. _NSConcreteStackBlock) are also checked via
// knownClasses because typemap.ClassName rejects non-uppercase-starting names.
func isObjectObjCType(qt string, knownClasses map[string]bool) bool {
	if qt == "" {
		return false
	}
	n := typemap.Normalise(qt)
	if typemap.IsID(n) || typemap.IsInstancetype(n) {
		return true
	}
	// id<Protocol> is an ObjC object conforming to a protocol; retain on return.
	if len(typemap.IDProtocols(n)) > 0 {
		return true
	}
	if typemap.IsPointer(n) && !typemap.IsDoublePointer(n) {
		// typemap.ClassName enforces an uppercase-start requirement.
		cls := typemap.ClassName(n)
		if cls != "" {
			if knownClasses[cls] {
				return true
			}
			// Capital-letter heuristic: ObjC class names start uppercase.
			return cls[0] >= 'A' && cls[0] <= 'Z'
		}
		// ClassName returned "" (non-uppercase-starting name, e.g. _NSFoo).
		// Fall back to a bare-name lookup in knownClasses.
		bare := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(n), "*"))
		if idx := strings.Index(bare, "<"); idx > 0 {
			bare = bare[:idx]
		}
		bare = strings.TrimSpace(bare)
		return bare != "" && knownClasses[bare]
	}
	return false
}

// stripGenericID removes `<id>` type arguments from ObjC type strings.
// When a generic type parameter (e.g. ResultType) is replaced with `id`,
// constrained generic classes like `NSFetchRequest<id<NSFetchRequestResult>>`
// fail the type-bound check. Stripping `<id>` gives the raw class pointer
// which the ObjC runtime accepts without constraint checking.
func stripGenericID(s string) string {
	return strings.ReplaceAll(s, "<id>", "")
}
