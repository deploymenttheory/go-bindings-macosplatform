package raw

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// EmitSubclassFactories writes per-class subclass factory files for all classes
// in framework that appear in superIndex (i.e. classes that Apple itself subclasses).
//
// For each eligible class NSFoo it writes:
//   - outDir/NSFoo_subclass.go  — NSFooOverrides struct + NewNSFooSubclass factory
//   - outDir/bridge/NSFoo_subclass.h — C declarations
//   - outDir/bridge/NSFoo_subclass.m — ObjC alloc+class_addMethod+BindCallback
func EmitSubclassFactories(outDir string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, superIndex map[string]bool, allClasses map[string]macosplatformmetadata.Class) error {
	packageName := strings.ToLower(framework.Framework)
	bridgeDir := filepath.Join(outDir, "bridge")

	// Sort class names for deterministic output.
	names := make([]string, 0, len(framework.Classes))
	for name := range framework.Classes {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, className := range names {
		cls := framework.Classes[className]
		if !superIndex[className] {
			continue
		}
		if m.GenericClasses[className] {
			continue
		}
		if cls.Availability.IsUnavailable {
			continue
		}
		methods := collectOverridableMethods(cls, m, allClasses)
		if len(methods) == 0 {
			continue
		}

		// Go factory file.
		var goBuf bytes.Buffer
		if err := emitSubclassGo(&goBuf, packageName, framework.Framework, className, methods, m); err != nil {
			return fmt.Errorf("subclass factory go %s: %w", className, err)
		}
		if goBuf.Len() > 0 {
			goPath := filepath.Join(outDir, className+"_subclass.go")
			if err := os.WriteFile(goPath, goBuf.Bytes(), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", goPath, err)
			}
		}

		// Bridge header.
		var hBuf bytes.Buffer
		emitSubclassHeader(&hBuf, className, methods)
		hPath := filepath.Join(bridgeDir, className+"_subclass.h")
		if err := os.WriteFile(hPath, hBuf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", hPath, err)
		}

		// Bridge implementation.
		var mBuf bytes.Buffer
		emitSubclassImpl(&mBuf, framework.Framework, className, methods)
		mPath := filepath.Join(bridgeDir, className+"_subclass.m")
		if err := os.WriteFile(mPath, mBuf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", mPath, err)
		}
	}
	return nil
}

// overrideMethod is a pruned view of macosplatformmetadata.Method for the subclass factory generator.
type overrideMethod struct {
	GoName   string // exported Go field name: "MouseUp"
	Selector string // ObjC selector: "mouseUp:"
	Sig      MethodSigModel
}

// collectOverridableMethods returns all IMP-safe instance methods reachable on
// cls and its framework ancestors, excluding root classes (NSObject, NSProxy, etc.).
// Walking up to but not including the root ensures that a generated NSView override
// struct includes NSResponder methods (mouseEntered:, mouseExited:, etc.) while
// never including universal NSObject infrastructure methods (dealloc, retain,
// allowsWeakReference, etc.) that are meaningless override targets for callers.
// If the resulting set is empty no subclass factory should be generated.
func collectOverridableMethods(cls macosplatformmetadata.Class, m *typemap.Mapper, allClasses map[string]macosplatformmetadata.Class) []overrideMethod {
	seen := make(map[string]bool)
	var result []overrideMethod

	// Walk the superclass chain: start with cls, then cls.Super, and so on.
	// Guard against cycles with a depth limit.
	current := cls
	for depth := 0; depth < 64; depth++ {
		for _, method := range current.Methods {
			if !isMethodIMPSafe(method, m) {
				continue
			}
			sig, ok := methodSigFromMethod(method, m)
			if !ok {
				continue
			}
			goName := naming.MethodName(method.Selector)
			if seen[goName] {
				continue
			}
			seen[goName] = true
			result = append(result, overrideMethod{
				GoName:   goName,
				Selector: method.Selector,
				Sig:      sig,
			})
		}
		if current.Super == "" {
			break
		}
		parent, ok := allClasses[current.Super]
		if !ok {
			break
		}
		// Stop before entering a root class. Root classes (NSObject, NSProxy)
		// have Super == "" and their methods are universal infrastructure, not
		// meaningful framework override points.
		if parent.Super == "" {
			break
		}
		current = parent
	}

	sort.Slice(result, func(i, j int) bool { return result[i].GoName < result[j].GoName })
	return result
}

// emitSubclassGo writes the Go factory file for NSFoo → goBridgeSub_NSFoo.
func emitSubclassGo(w *bytes.Buffer, packageName, framework, className string, methods []overrideMethod, m *typemap.Mapper) error {
	return executeTemplate(w, "subclass_go_file", buildSubclassGoFileModel(packageName, framework, className, methods))
}

// emitSubclassHeader writes the .h file declaring the C bridge functions.
func emitSubclassHeader(w *bytes.Buffer, className string, methods []overrideMethod) {
	goClassName := "goBridge_Sub_" + className
	_ = executeTemplate(w, "subclass_h_file", subclassHFileModel{
		GoClassName: goClassName,
		IMPSigs:     uniqueIMPSigsFor(methods),
	})
}

// emitSubclassImpl writes the .m file that creates the per-instance subclass.
func emitSubclassImpl(w *bytes.Buffer, framework, className string, methods []overrideMethod) {
	goClassName := "goBridge_Sub_" + className
	_ = executeTemplate(w, "subclass_m_file", subclassMFileModel{
		Framework:   framework,
		ClassName:   className,
		GoClassName: goClassName,
		HeaderFile:  className + "_subclass.h",
		IMPSigs:     uniqueIMPSigsFor(methods),
	})
}

func buildSubclassGoFileModel(packageName, framework, className string, methods []overrideMethod) subclassGoFileModel {
	goClassName := "goBridge_Sub_" + className
	impSigs := uniqueIMPSigsFor(methods)

	// Pre-render the Overrides struct fields.
	var overridesFields strings.Builder
	for _, method := range methods {
		funcType := method.Sig.goCallbackFuncType()
		fmt.Fprintf(&overridesFields, "\t// %s overrides -[%s %s].\n", method.GoName, className, method.Selector)
		fmt.Fprintf(&overridesFields, "\t%s %s\n", method.GoName, funcType)
	}

	// Pre-render the add_method if-blocks (inside "if overrides != nil").
	var addMethodBody strings.Builder
	for _, method := range methods {
		enc := method.Sig.ObjCEnc
		getterFn := impGetterName(method.Sig.Name)
		fmt.Fprintf(&addMethodBody, "\t\tif overrides.%s != nil {\n", method.GoName)
		fmt.Fprintf(&addMethodBody, "\t\t\t_sel%s := C.CString(%q)\n", method.GoName, method.Selector)
		fmt.Fprintf(&addMethodBody, "\t\t\t_enc%s := C.CString(%q)\n", method.GoName, enc)
		fmt.Fprintf(&addMethodBody, "\t\t\tC.%s_addMethod(cls, _sel%s, C.%s(), _enc%s)\n",
			goClassName, method.GoName, getterFn, method.GoName)
		fmt.Fprintf(&addMethodBody, "\t\t\tC.free(unsafe.Pointer(_sel%s))\n", method.GoName)
		fmt.Fprintf(&addMethodBody, "\t\t\tC.free(unsafe.Pointer(_enc%s))\n", method.GoName)
		fmt.Fprintf(&addMethodBody, "\t\t}\n")
	}

	// Pre-render the BindMethod if-blocks (inside "if overrides != nil").
	var bindMethodBody strings.Builder
	for _, method := range methods {
		fmt.Fprintf(&bindMethodBody, "\t\tif overrides.%s != nil {\n", method.GoName)
		fmt.Fprintf(&bindMethodBody, "\t\t\tcallbacks.BindMethod(obj, %q, overrides.%s)\n", method.Selector, method.GoName)
		fmt.Fprintf(&bindMethodBody, "\t\t}\n")
	}

	return subclassGoFileModel{
		ClassName:       className,
		GoClassName:     goClassName,
		PackageName:     packageName,
		IMPSigs:         impSigs,
		OverridesFields: overridesFields.String(),
		AddMethodBody:   addMethodBody.String(),
		BindMethodBody:  bindMethodBody.String(),
	}
}

// EmitGeneratedBridgesImpl writes {packageName}_impl.m at outDir — a
// CGo-compiled aggregate that #includes all *_subclass.m and
// *_impl.m files from bridge/. Without this file the ObjC symbols
// defined in those bridge files would not be linked. If no generated bridge
// files exist the file is not written.
func EmitGeneratedBridgesImpl(outDir, packageName string) error {
	bridgeDir := filepath.Join(outDir, "bridge")
	entries, err := os.ReadDir(bridgeDir)
	if err != nil {
		return nil
	}

	var files []string
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && (strings.HasSuffix(n, "_subclass.m") || strings.HasSuffix(n, "_protocol_callback.m")) {
			files = append(files, n)
		}
	}
	if len(files) == 0 {
		return nil
	}
	sort.Strings(files)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n")
	fmt.Fprintf(&buf, "#pragma clang diagnostic push\n")
	fmt.Fprintf(&buf, "#pragma clang diagnostic ignored \"-Wdeprecated-declarations\"\n")
	fmt.Fprintf(&buf, "#pragma clang diagnostic ignored \"-Wunguarded-availability\"\n")
	fmt.Fprintf(&buf, "#pragma clang diagnostic ignored \"-Wunguarded-availability-new\"\n")
	for _, f := range files {
		fmt.Fprintf(&buf, "#include \"bridge/%s\"\n", f)
	}
	fmt.Fprintf(&buf, "#pragma clang diagnostic pop\n")

	path := filepath.Join(outDir, packageName+"_impl.m")
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// uniqueIMPSigsFor returns the sorted deduplicated set of IMP sig names needed
// for the given set of methods.
func uniqueIMPSigsFor(methods []overrideMethod) []string {
	seen := make(map[string]bool)
	for _, m := range methods {
		seen[m.Sig.Name] = true
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// impGetterName returns the C function name that returns the IMP pointer for sigName.
// e.g. "void_ptr" → "goBridge_Trampoline_MethodFn_void_ptr"
func impGetterName(sigName string) string {
	return "goBridge_Trampoline_MethodFn_" + sigName
}
