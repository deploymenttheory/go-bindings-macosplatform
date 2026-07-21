//go:build darwin

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// methodID returns the structured ID comment for an ObjC class method.
func methodID(framework, className, selector string) string {
	return "ID: objc-sym " + framework + "." + className + ".+[" + selector + "]"
}

// functionID returns the structured ID comment for a free C function.
func functionID(framework, funcName string) string {
	return "ID: objc-sym " + framework + "." + funcName
}

// loadCandidates reads all .gometa.json files from metaDirs and returns every
// auto-testable candidate record derived from the metadata.
// Only zero-arg class methods and free functions that will actually be emitted
// as exported Go symbols are included.
func loadCandidates(metaDirs []string, modulePrefix string) ([]FuncRecord, error) {
	files, err := collectMetaFiles(metaDirs)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "found %d metadata files\n", len(files))

	var candidates []FuncRecord
	for _, path := range files {
		framework, err := meta.Read(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", path, err)
			continue
		}
		recs := candidatesFromFramework(framework, modulePrefix)
		candidates = append(candidates, recs...)
	}
	return candidates, nil
}

// collectMetaFiles globs .gometa.json files from the given directory trees.
// Prefers arm64 files when multiple arch variants exist in one directory.
func collectMetaFiles(dirs []string) ([]string, error) {
	var all []string
	for _, dir := range dirs {
		top, err := filepath.Glob(filepath.Join(dir, "*.gometa.json"))
		if err != nil {
			return nil, err
		}
		sub, err := filepath.Glob(filepath.Join(dir, "*", "*.gometa.json"))
		if err != nil {
			return nil, err
		}
		all = append(all, selectBestArch(append(top, sub...))...)
	}
	return all, nil
}

// selectBestArch picks one file per directory, preferring arm64.
func selectBestArch(files []string) []string {
	byDir := map[string][]string{}
	for _, f := range files {
		byDir[filepath.Dir(f)] = append(byDir[filepath.Dir(f)], f)
	}
	var out []string
	for _, group := range byDir {
		best := group[0]
		for _, f := range group[1:] {
			base := filepath.Base(f)
			if len(base) > 5 && base[len(base)-5:] != ".json" {
				continue
			}
			// Prefer arm64 over any other arch variant.
			if containsSubstring(f, "arm64") && !containsSubstring(best, "arm64") {
				best = f
			}
		}
		out = append(out, best)
	}
	return out
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()
}

// candidatesFromFramework extracts FuncRecord candidates from one framework's metadata.
func candidatesFromFramework(framework *meta.FrameworkMeta, modulePrefix string) []FuncRecord {
	if framework.IsSwiftOnly {
		return nil
	}
	if len(framework.UmbrellaFor) > 0 {
		return nil
	}
	if skippedFrameworks[framework.Framework] {
		return nil
	}

	pkg := naming.PackageName(framework.Framework)
	isLib := framework.LinkLib != ""
	// Target the idiomatic layer — the public API developers consume — rather than
	// the raw bindings. The zero-arg class-method factory function names are shared
	// between the two layers (both use ClassMethodGoNameFromMeta), so only the
	// import path changes here; the generated assertions call-and-discard so they
	// compile against the idiomatic return types (obj.Object / value types).
	var importSubdir string
	if isLib {
		importSubdir = "bindings/libraries"
	} else {
		importSubdir = "bindings/frameworks"
	}
	importPath := modulePrefix + "/" + importSubdir + "/" + pkg
	needsMainThread := mainThreadFrameworks[framework.Framework]

	var out []FuncRecord

	// Class methods.
	for className, cls := range framework.Classes {
		if cls.Availability.IsUnavailable {
			continue
		}
		for _, method := range cls.Methods {
			if !method.IsClassMethod {
				continue
			}
			if !rawfw.MethodWillBeEmitted(method) {
				continue
			}
			if len(method.Params) > 0 {
				continue // only zero-arg methods are auto-testable
			}
			goName := rawfw.ClassMethodGoNameFromMeta(className, method.Selector, framework)
			id := methodID(framework.Framework, className, method.Selector)
			rec := classifyMetaRecord(id, framework.Framework, method.Selector, goName, pkg, importPath, needsMainThread, method.Return, method.Return.IsNullable, method.IsNSError)
			if rec != nil {
				out = append(out, *rec)
			}
		}
	}

	// Free functions.
	for _, fn := range framework.Functions {
		if !rawfw.FunctionWillBeEmitted(fn) {
			continue
		}
		if len(fn.Params) > 0 {
			continue
		}
		// Skip SDK-internal marker symbols (e.g. EventKit's DATE_COMPONENTS_DO_NOT_USE).
		if containsSubstring(fn.Name, "DO_NOT_USE") {
			continue
		}
		if isSkippedFunction(framework.Framework, fn.Name) {
			continue
		}
		id := functionID(framework.Framework, fn.Name)
		rec := classifyMetaRecord(id, framework.Framework, fn.Name, rawfw.FunctionGoName(fn), pkg, importPath, needsMainThread, fn.Return, false, false)
		if rec != nil {
			rec.IsCFunction = true
			out = append(out, *rec)
		}
	}

	// The idiomatic layer renames many symbols (deprefixed class methods such as
	// DCDeviceCurrentDevice → CurrentDevice), so a raw-derived name may not exist
	// there. Keep only candidates whose function is actually present in the emitted
	// idiomatic package, so the generated test file always compiles. When the
	// package's source cannot be read (e.g. a stub with no callable API), drop the
	// framework's candidates rather than emit unresolved calls.
	idiomaticFuncs := idiomaticExportedFuncs(filepath.Join(importSubdir, pkg))
	filtered := out[:0]
	for _, rec := range out {
		// C functions are excluded: the idiomatic wrappers bind lazily and panic on
		// a symbol absent from the running OS, and the idiomatic layer has no
		// SymbolAvailable guard to skip them (the raw layer did). Only ObjC
		// class-method factories/singletons — dispatched via objc_msgSend, always
		// safe — are sampled, and only when the idiomatic package actually declares
		// the (often deprefixed) function name.
		if rec.IsCFunction {
			continue
		}
		if idiomaticFuncs[rec.GoFuncName] {
			filtered = append(filtered, rec)
		}
	}
	return filtered
}

// idiomaticExportedFuncs returns the exported, receiver-less, zero-parameter
// function names the emitted idiomatic package at dir declares — the calls a
// generated `pkg.Func()` test can resolve. An unreadable directory yields an
// empty set.
func idiomaticExportedFuncs(dir string) map[string]bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := map[string]bool{}
	fset := token.NewFileSet()
	for _, ent := range entries {
		name := ent.Name()
		if ent.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, perr := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if perr != nil {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() {
				continue
			}
			if fn.Type.Params.NumFields() == 0 {
				out[fn.Name.Name] = true
			}
		}
	}
	return out
}

// classifyMetaRecord builds a FuncRecord from metadata fields. Returns nil if
// the symbol should be skipped.
func classifyMetaRecord(
	id, framework, selector, goFuncName, pkg, importPath string,
	needsMainThread bool,
	retType meta.ReturnType,
	isNullable bool,
	isNSError bool,
) *FuncRecord {
	if isNSError {
		return nil // method adds implicit NSError** → Go emits (T, error); not auto-testable
	}
	retKind := classifyReturnFromMeta(retType)
	if retKind == RetTuple {
		return nil // (T, error) returns require unwrapping the template can't produce
	}

	category := categorise(selector, retKind, isNullable)
	if category == CatSkip {
		return nil
	}

	return &FuncRecord{
		ID:              id,
		Framework:       framework,
		Selector:        selector,
		GoFuncName:      goFuncName,
		ZeroArgs:        true,
		RetKind:         retKind,
		GoPackage:       pkg,
		GoImportPath:    importPath,
		NeedsMainThread: needsMainThread,
		TestName:        testName(id),
		Category:        category,
	}
}

// classifyReturnFromMeta maps metadata return type information to a RetKind.
//
// Only instancetype / generic returns are classified as RetPointer (nil-checkable)
// because those always produce a typed Go pointer in the generated bindings.
// Concrete ObjC class pointers (e.g. "NSFoo *") may degrade to objc.ID due to
// import-cycle breaks in the emitter, and C pointer types like "const char *"
// map to Go string — neither can be compared to nil. Treating them as RetScalar
// (emit "_ = result") avoids compile errors and is safe for smoke testing.
func classifyReturnFromMeta(ret meta.ReturnType) RetKind {
	if rawfw.ReturnIsVoid(ret) {
		return RetVoid
	}
	// instancetype / generic always produce a real typed pointer in purego.
	if ret.IsInstancetype || ret.IsGeneric {
		return RetPointer
	}
	objcType := ret.ObjCType
	// NSError** out-param — the emitter adds an error return value.
	if len(objcType) > 3 && objcType[len(objcType)-3:] == " **" {
		return RetTuple
	}
	// Protocol return (id<Proto>) — maps to a Go interface in generated code.
	// purego panics at runtime on interface returns ("unsupported return kind:
	// interface"), so skip these methods.
	if containsSubstring(objcType, "id<") {
		return RetTuple // sentinel for "skip"
	}
	// Everything else — bare id (→ objc.ID), const char * (→ string), concrete
	// class pointers that may degrade to objc.ID, and plain scalar types — are
	// all treated as scalar so the test body uses "_ = result" with no nil-check.
	return RetScalar
}

// categorise classifies a symbol into a testable category.
func categorise(selector string, retKind RetKind, isNullable bool) Category {
	switch retKind {
	case RetPointer:
		if isSingleton(selector) || isNullable {
			return CatSingleton
		}
		return CatZeroArgFactory
	case RetScalar:
		return CatZeroArgScalar
	case RetVoid:
		return CatZeroArgScalar // void functions are "scalar" in the test template
	case RetTuple:
		return CatSkip
	}
	return CatSkip
}
