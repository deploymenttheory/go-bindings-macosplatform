package pipeline

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	idiofw "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks"
	rawfw "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emitmanifest"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/typemap"
)

// BindingsConfig controls the bindings generation pass.
type BindingsConfig struct {
	Registry         *Registry
	FrameworksOutDir string // e.g. ./purego-frameworks
	LibrariesOutDir  string // e.g. ./purego-libraries
	Verbose          bool

	// DiagnosticsSink, when non-nil, receives every type-degradation
	// diagnostic recorded by the per-framework type mappers (unsafe.Pointer
	// fallbacks and cycle-forced objc.ID substitutions). The CLI uses this
	// to enforce a committed diagnostics baseline.
	DiagnosticsSink *[]string

	// Manifest, when non-nil, receives one parity entry per emitted construct
	// (keyed on its ObjC/C name), so the raw output can serve as the oracle the
	// idiomatic emitter's coverage is checked against. It never affects the
	// emitted bytes.
	Manifest *emitmanifest.Recorder
}

// GenerateBindings generates all framework and library packages from the registry.
func GenerateBindings(cfg BindingsConfig) error {
	// Clean output directories (only those that are configured).
	if cfg.FrameworksOutDir != "" {
		if err := cleanDir(cfg.FrameworksOutDir); err != nil {
			return fmt.Errorf("clean frameworks dir: %w", err)
		}
	}
	if cfg.LibrariesOutDir != "" {
		if err := cleanDir(cfg.LibrariesOutDir); err != nil {
			return fmt.Errorf("clean libraries dir: %w", err)
		}
	}

	ordered := SortByDependency(cfg.Registry)

	for _, m := range ordered {
		if m.IsSwiftOnly {
			if err := emitSwiftOnlyStub(m, cfg); err != nil {
				return fmt.Errorf("emit stub %s: %w", m.Framework, err)
			}
			continue
		}
		isLibrary := m.LinkLib != ""
		// Caller can opt out of one class by leaving the corresponding dir empty.
		if isLibrary && cfg.LibrariesOutDir == "" {
			continue
		}
		if !isLibrary && cfg.FrameworksOutDir == "" {
			continue
		}
		outDir := cfg.FrameworksOutDir
		if isLibrary {
			outDir = cfg.LibrariesOutDir
		}
		if err := generateFramework(m, outDir, cfg); err != nil {
			return fmt.Errorf("generate %s: %w", m.Framework, err)
		}
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "generated %s\n", m.Framework)
		}
	}
	return nil
}

func generateFramework(m *meta.FrameworkMeta, outBase string, cfg BindingsConfig) error {
	reg := cfg.Registry
	packageName := naming.PackageName(m.Framework)
	outDir := filepath.Join(outBase, packageName)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	mapper := &typemap.Mapper{
		OwnerIndex:               reg.OwnerIndex,
		GenericClasses:           reg.GenericClasses,
		GenericParamIndex:        reg.GenericParamIndex,
		EnumIndex:                reg.EnumIndex,
		EnumGoTypeIndex:          reg.EnumGoTypeIndex,
		TypedefIndex:             reg.TypedefIndex,
		StructIndex:              reg.StructIndex,
		ProtocolIndex:            reg.ProtocolIndex,
		CFTypeIndex:              reg.CFTypeIndex,
		BlockedImports:           reg.BlockedImports,
		UnavailableClasses:       reg.UnavailableClasses,
		UnavailableEnumBaseTypes: reg.UnavailableEnumBaseTypes,
		ModulePrefix:             reg.ModulePrefix,
		LibraryModulePrefix:      reg.LibraryModulePrefix,
	}

	snap := &rawfw.RegistrySnapshot{
		OwnerIndex:         reg.OwnerIndex,
		GenericClasses:     reg.GenericClasses,
		GenericParamIndex:  reg.GenericParamIndex,
		ClassIndex:         reg.ClassIndex,
		BlockedImports:     reg.BlockedImports,
		EnumGoTypeIndex:    reg.EnumGoTypeIndex,
		UnavailableClasses: reg.UnavailableClasses,
		ModulePrefix:       reg.ModulePrefix,
	}

	isLibrary := m.LinkLib != ""

	// doc.go
	if err := writeDocGo(outDir, packageName, m); err != nil {
		return err
	}

	// Collect C function registration lines before writing runtime.go so they
	// can be embedded inside the init() after Dlopen — preventing the init-order
	// bug that arises when registrations live in a separate file's init().
	fnRegLines, err := writeFunctionsFile(
		outDir,
		packageName,
		m,
		mapper,
		reg.OwnerIndex,
		cfg.Manifest,
	)
	if err != nil {
		return err
	}

	// runtime.go — framework loading + C function registrations inside init().
	if err := writeRuntimeGo(outDir, packageName, m, isLibrary, fnRegLines); err != nil {
		return err
	}

	if err := writeEnumsFile(outDir, packageName, m, cfg.Manifest); err != nil {
		return err
	}
	if err := writeStructsFile(outDir, packageName, m, mapper, cfg.Manifest); err != nil {
		return err
	}
	if !isLibrary {
		if err := writeProtocolsFile(
			outDir,
			packageName,
			m,
			mapper,
			reg.OwnerIndex,
			cfg.Manifest,
		); err != nil {
			return err
		}
	}
	if err := writeExternsFile(outDir, packageName, m, mapper, cfg.Manifest); err != nil {
		return err
	}

	// Classes (ObjC frameworks only).
	if !isLibrary && len(m.Classes) > 0 {
		if err := rawfw.EmitClasses(
			outDir,
			packageName,
			m.Framework,
			m,
			mapper,
			snap,
			cfg.Manifest,
		); err != nil {
			return fmt.Errorf("emit classes for %s: %w", m.Framework, err)
		}
	}

	if cfg.DiagnosticsSink != nil {
		*cfg.DiagnosticsSink = append(*cfg.DiagnosticsSink, mapper.Diagnostics...)
	}

	return nil
}

// writeFunctionsFile emits <packageName>_functions.go and returns the registration
// lines runtime.go must run after a successful Dlopen.
func writeFunctionsFile(
	outDir, packageName string,
	m *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	ownerIndex map[string]string,
	rec *emitmanifest.Recorder,
) ([]rawfw.FunctionRegistration, error) {
	if len(m.Functions) == 0 {
		return nil, nil
	}
	var body bytes.Buffer
	dylibVar := dylibVarName(m)
	fnImports, regLines, err := rawfw.EmitFunctions(&body, m, mapper, dylibVar, ownerIndex, rec)
	if err != nil {
		return nil, fmt.Errorf("emit functions for %s: %w", m.Framework, err)
	}
	bodyStr := body.String()
	if bodyStr == "" {
		return regLines, nil
	}
	var buf bytes.Buffer
	if strings.Contains(bodyStr, "unsafe.") {
		fnImports["unsafe"] = "unsafe"
	}
	if strings.Contains(bodyStr, "objc.") {
		fnImports[rawfw.ObjcPkg] = rawfw.ObjcImport
	}
	if strings.Contains(bodyStr, "purego.") {
		fnImports["purego"] = rawfw.PureobjcImport
	}
	// purego import only needed in the functions file for the var types;
	// RegisterLibFunc calls move into runtime.go which already imports purego.
	pruneUnusedImports(fnImports, bodyStr)
	writeFileHeaderRaw(&buf, packageName, fnImports)
	buf.Write(body.Bytes())
	if err := writeGoFile(outDir, packageName+"_functions.go", buf.Bytes()); err != nil {
		return nil, err
	}
	return regLines, nil
}

// writeEnumsFile emits <packageName>_enums.go.
func writeEnumsFile(
	outDir, packageName string,
	m *meta.FrameworkMeta,
	rec *emitmanifest.Recorder,
) error {
	if len(m.Enums) == 0 {
		return nil
	}
	var buf bytes.Buffer
	needStrings := false
	needFmt := false
	// Scan to see which imports enums need.
	for _, e := range m.Enums {
		if e.IsBitmask {
			needStrings = true
		} else {
			needFmt = true
		}
	}
	imports := typemap.ImportSet{}
	if needStrings {
		imports["strings"] = "strings"
	}
	if needFmt {
		imports["fmt"] = "fmt"
	}
	writeFileHeaderRaw(&buf, packageName, imports)
	if err := rawfw.EmitEnums(&buf, m, rec); err != nil {
		return fmt.Errorf("emit enums for %s: %w", m.Framework, err)
	}
	return writeGoFile(outDir, packageName+"_enums.go", buf.Bytes())
}

// writeStructsFile emits <packageName>_structs.go.
func writeStructsFile(
	outDir, packageName string,
	m *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	rec *emitmanifest.Recorder,
) error {
	if len(m.Structs) == 0 {
		return nil
	}
	var body bytes.Buffer
	structImports, err := rawfw.EmitStructs(&body, m, mapper, rec)
	if err != nil {
		return fmt.Errorf("emit structs for %s: %w", m.Framework, err)
	}
	bodyStr := body.String()
	if strings.Contains(bodyStr, "unsafe.") {
		structImports["unsafe"] = "unsafe"
	}
	if strings.Contains(bodyStr, "objc.") {
		structImports[rawfw.ObjcPkg] = rawfw.ObjcImport
	}
	pruneUnusedImports(structImports, bodyStr)
	var buf bytes.Buffer
	writeFileHeaderRaw(&buf, packageName, structImports)
	buf.Write(body.Bytes())
	return writeGoFile(outDir, packageName+"_structs.go", buf.Bytes())
}

// writeProtocolsFile emits <packageName>_protocols.go (ObjC frameworks only).
func writeProtocolsFile(
	outDir, packageName string,
	m *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	ownerIndex map[string]string,
	rec *emitmanifest.Recorder,
) error {
	if len(m.Protocols) == 0 {
		return nil
	}
	var body bytes.Buffer
	protoImports, err := rawfw.EmitProtocols(&body, m, mapper, ownerIndex, rec)
	if err != nil {
		return fmt.Errorf("emit protocols for %s: %w", m.Framework, err)
	}
	bodyStr := body.String()
	if bodyStr == "" {
		return nil
	}
	if strings.Contains(bodyStr, "unsafe.") {
		protoImports["unsafe"] = "unsafe"
	}
	if strings.Contains(bodyStr, "objc.") {
		protoImports[rawfw.ObjcPkg] = rawfw.ObjcImport
	}
	pruneUnusedImports(protoImports, bodyStr)
	var buf bytes.Buffer
	writeFileHeaderRaw(&buf, packageName, protoImports)
	buf.Write(body.Bytes())
	return writeGoFile(outDir, packageName+"_protocols.go", buf.Bytes())
}

// writeExternsFile emits <packageName>_externs.go.
func writeExternsFile(
	outDir, packageName string,
	m *meta.FrameworkMeta,
	mapper *typemap.Mapper,
	rec *emitmanifest.Recorder,
) error {
	if len(m.Externs) == 0 {
		return nil
	}
	var body bytes.Buffer
	extImports, err := rawfw.EmitExterns(&body, m, mapper, dylibVarName(m), rec)
	if err != nil {
		return fmt.Errorf("emit externs for %s: %w", m.Framework, err)
	}
	bodyStr := body.String()
	if bodyStr == "" {
		return nil
	}
	if strings.Contains(bodyStr, "unsafe.") {
		extImports["unsafe"] = "unsafe"
	}
	if strings.Contains(bodyStr, "purego.") {
		extImports["purego"] = "github.com/ebitengine/purego"
	}
	if strings.Contains(bodyStr, "objc.") {
		extImports[rawfw.ObjcPkg] = rawfw.ObjcImport
	}
	pruneUnusedImports(extImports, bodyStr)
	var buf bytes.Buffer
	writeFileHeaderRaw(&buf, packageName, extImports)
	buf.Write(body.Bytes())
	return writeGoFile(outDir, packageName+"_externs.go", buf.Bytes())
}

func writeDocGo(outDir, packageName string, m *meta.FrameworkMeta) error {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated by go-bindings-purecg. DO NOT EDIT.\n")
	fmt.Fprintf(&buf, "//go:build darwin\n\n")
	fmt.Fprintf(
		&buf,
		"// Package %s provides purego-based Go bindings for the macOS %s framework.\n",
		packageName,
		m.Framework,
	)
	if m.LinkLib != "" {
		fmt.Fprintf(
			&buf,
			"// This is a C library package — all APIs are pure C, no Objective-C runtime.\n",
		)
	}
	if m.IsSwiftOnly {
		fmt.Fprintf(&buf, "// This framework is Swift-only and has no callable bindings.\n")
	}
	if m.LinkLib == "" {
		fmt.Fprintf(
			&buf,
			"//\n// Apple documentation: https://developer.apple.com/documentation/%s\n",
			strings.ToLower(m.Framework),
		)
	}
	fmt.Fprintf(&buf, "package %s\n", packageName)
	return writeGoFile(outDir, "doc.go", buf.Bytes())
}

// writeRuntimeGo writes the runtime.go file for a framework package.
// regLines, if non-empty, are purego.RegisterLibFunc calls for C functions
// in this framework. They are emitted inside _loadLibrary() after a
// successful Dlopen so they always have access to the library handle.
//
// Loading goes through a sync.Once so that package-level class var
// initializers — which Go runs BEFORE this package's init() — can force the
// framework load via _objcClass. Without this, objc.GetClass would resolve
// to nil classes for any framework not already loaded into the process
// (Foundation happens to be pre-loaded, masking the bug; Virtualization and
// AppKit are not).
func writeRuntimeGo(
	outDir, packageName string,
	m *meta.FrameworkMeta,
	_ bool,
	regLines []rawfw.FunctionRegistration,
) error {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated by go-bindings-purecg. DO NOT EDIT.\n")
	fmt.Fprintf(&buf, "//go:build darwin\n\n")
	fmt.Fprintf(&buf, "package %s\n\n", packageName)
	if len(regLines) > 0 {
		fmt.Fprintf(&buf, "import (\n")
		fmt.Fprintf(&buf, "\t\"fmt\"\n")
		fmt.Fprintf(&buf, "\t\"os\"\n")
		fmt.Fprintf(&buf, "\t\"sync\"\n\n")
		fmt.Fprintf(&buf, "\t\"github.com/ebitengine/purego\"\n")
		fmt.Fprintf(&buf, "\t\"github.com/ebitengine/purego/objc\"\n")
		fmt.Fprintf(&buf, ")\n\n")
	} else {
		fmt.Fprintf(&buf, "import (\n")
		fmt.Fprintf(&buf, "\t\"fmt\"\n")
		fmt.Fprintf(&buf, "\t\"sync\"\n\n")
		fmt.Fprintf(&buf, "\t\"github.com/ebitengine/purego\"\n")
		fmt.Fprintf(&buf, "\t\"github.com/ebitengine/purego/objc\"\n")
		fmt.Fprintf(&buf, ")\n\n")
	}

	dylibVar := dylibVarName(m)
	dylibPath := frameworkDylibPath(m)

	fmt.Fprintf(&buf, "var (\n")
	fmt.Fprintf(&buf, "\t%s uintptr\n", dylibVar)
	fmt.Fprintf(&buf, "\t_loadOnce sync.Once\n")
	if len(regLines) > 0 {
		fmt.Fprintf(&buf, "\t_failedSymbols = make(map[string]bool)\n")
	}
	fmt.Fprintf(&buf, ")\n\n")

	if len(regLines) > 0 {
		fmt.Fprintf(&buf, "// _register binds one C symbol, recording the symbol on failure so\n")
		fmt.Fprintf(&buf, "// SymbolAvailable can report it. A symbol that doesn't exist (e.g. a\n")
		fmt.Fprintf(
			&buf,
			"// header-inline function) or exceeds purego's argument limits must not\n",
		)
		fmt.Fprintf(&buf, "// prevent the remaining functions from being registered.\n")
		fmt.Fprintf(&buf, "func _register(symbol string, register func()) {\n")
		fmt.Fprintf(&buf, "\tdefer func() {\n")
		fmt.Fprintf(&buf, "\t\tif recover() != nil {\n")
		fmt.Fprintf(&buf, "\t\t\t_failedSymbols[symbol] = true\n")
		fmt.Fprintf(&buf, "\t\t}\n")
		fmt.Fprintf(&buf, "\t}()\n")
		fmt.Fprintf(&buf, "\tregister()\n")
		fmt.Fprintf(&buf, "}\n\n")

		fmt.Fprintf(
			&buf,
			"// SymbolAvailable reports whether the named C symbol was bound when the\n",
		)
		fmt.Fprintf(
			&buf,
			"// library loaded. Calling a generated wrapper whose symbol is unavailable\n",
		)
		fmt.Fprintf(&buf, "// dereferences a nil function variable and panics.\n")
		fmt.Fprintf(&buf, "func SymbolAvailable(symbol string) bool {\n")
		fmt.Fprintf(&buf, "\t_loadOnce.Do(_loadLibrary)\n")
		fmt.Fprintf(&buf, "\treturn %s != 0 && !_failedSymbols[symbol]\n", dylibVar)
		fmt.Fprintf(&buf, "}\n\n")
	}

	fmt.Fprintf(&buf, "func _loadLibrary() {\n")
	fmt.Fprintf(&buf, "\tvar err error\n")
	fmt.Fprintf(
		&buf,
		"\t%s, err = purego.Dlopen(%q, purego.RTLD_GLOBAL|purego.RTLD_LAZY)\n",
		dylibVar,
		dylibPath,
	)
	if len(regLines) > 0 {
		// Frameworks that expose plain C functions (regLines non-empty) may be
		// types-only (no loadable dylib, e.g. CoreAudioTypes), or reorganised
		// or removed in newer SDK releases. Treat a missing dylib as
		// non-fatal and stay quiet unless PUREGO_BINDINGS_DEBUG is set,
		// leaving all function vars as no-ops.
		fmt.Fprintf(&buf, "\tif err != nil {\n")
		fmt.Fprintf(&buf, "\t\tif os.Getenv(\"PUREGO_BINDINGS_DEBUG\") != \"\" {\n")
		fmt.Fprintf(
			&buf,
			"\t\t\tfmt.Fprintf(os.Stderr, \"purego-bindings: %%s unavailable (%%v) — C functions will be no-ops\\n\", %q, err)\n",
			m.Framework,
		)
		fmt.Fprintf(&buf, "\t\t}\n")
		fmt.Fprintf(&buf, "\t\treturn\n")
		fmt.Fprintf(&buf, "\t}\n")
		// Each registration runs through _register so a failing symbol is
		// recorded and doesn't prevent the remaining registrations.
		for _, reg := range regLines {
			fmt.Fprintf(&buf, "\t_register(%q, func() { %s })\n", reg.Symbol, reg.Line)
		}
	} else {
		// Pure ObjC framework — must be present; panic on failure.
		fmt.Fprintf(&buf, "\tif err != nil {\n")
		fmt.Fprintf(
			&buf,
			"\t\tpanic(fmt.Sprintf(\"purego-bindings: failed to load %s: %%v\", err))\n",
			m.Framework,
		)
		fmt.Fprintf(&buf, "\t}\n")
	}
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "func init() {\n")
	fmt.Fprintf(&buf, "\t_loadOnce.Do(_loadLibrary)\n")
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "// _objcClass resolves an ObjC class after ensuring the framework is\n")
	fmt.Fprintf(&buf, "// loaded. It is called from package-level class var initializers, which\n")
	fmt.Fprintf(&buf, "// run before this package's init().\n")
	fmt.Fprintf(&buf, "func _objcClass(name string) objc.Class {\n")
	fmt.Fprintf(&buf, "\t_loadOnce.Do(_loadLibrary)\n")
	fmt.Fprintf(&buf, "\treturn objc.GetClass(name)\n")
	fmt.Fprintf(&buf, "}\n")

	return writeGoFile(outDir, packageName+"_runtime.go", buf.Bytes())
}

func emitSwiftOnlyStub(m *meta.FrameworkMeta, cfg BindingsConfig) error {
	packageName := naming.PackageName(m.Framework)
	outDir := filepath.Join(cfg.FrameworksOutDir, packageName)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	return writeDocGo(outDir, packageName, m)
}

// pruneUnusedImports removes entries from imports whose alias does not appear
// in bodyStr. This catches cases where the type mapper added a cross-framework
// import that the emitter later replaced with unsafe.Pointer or objc.ID.
func pruneUnusedImports(imports typemap.ImportSet, bodyStr string) {
	for alias, path := range imports {
		// stdlib packages (no dot in path) use the last path segment as alias
		segs := strings.Split(path, "/")
		defaultAlias := segs[len(segs)-1]
		usedAlias := alias
		if alias == defaultAlias {
			usedAlias = defaultAlias
		}
		if !strings.Contains(bodyStr, usedAlias+".") {
			delete(imports, alias)
		}
	}
}

func writeFileHeaderRaw(buf *bytes.Buffer, packageName string, imports typemap.ImportSet) {
	fmt.Fprintf(buf, "// Code generated by go-bindings-purecg. DO NOT EDIT.\n")
	fmt.Fprintf(buf, "//go:build darwin\n\n")
	fmt.Fprintf(buf, "package %s\n\n", packageName)

	if len(imports) == 0 {
		return
	}

	var stdlib, external, internal_ []string
	for _, path := range imports {
		switch {
		case !strings.Contains(path, "."):
			stdlib = append(stdlib, path)
		case strings.HasPrefix(path, "github.com/deploymenttheory"):
			internal_ = append(internal_, path)
		default:
			external = append(external, path)
		}
	}
	sort.Strings(stdlib)
	sort.Strings(external)
	sort.Strings(internal_)

	// alias lookup
	pathAlias := make(map[string]string)
	for alias, path := range imports {
		segs := strings.Split(path, "/")
		if alias != segs[len(segs)-1] {
			pathAlias[path] = alias
		}
	}

	fmt.Fprint(buf, "import (\n")
	for _, p := range stdlib {
		if a, ok := pathAlias[p]; ok {
			fmt.Fprintf(buf, "\t%s %q\n", a, p)
		} else {
			fmt.Fprintf(buf, "\t%q\n", p)
		}
	}
	if len(stdlib) > 0 && len(external)+len(internal_) > 0 {
		fmt.Fprint(buf, "\n")
	}
	for _, p := range external {
		if a, ok := pathAlias[p]; ok {
			fmt.Fprintf(buf, "\t%s %q\n", a, p)
		} else {
			fmt.Fprintf(buf, "\t%q\n", p)
		}
	}
	if len(external) > 0 && len(internal_) > 0 {
		fmt.Fprint(buf, "\n")
	}
	for _, p := range internal_ {
		if a, ok := pathAlias[p]; ok {
			fmt.Fprintf(buf, "\t%s %q\n", a, p)
		} else {
			fmt.Fprintf(buf, "\t%q\n", p)
		}
	}
	fmt.Fprint(buf, ")\n\n")
}

func writeGoFile(outDir, filename string, content []byte) error {
	return rawfw.WriteGoFile(filepath.Join(outDir, filename), content)
}

func cleanDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clean %s: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return nil
}

// IdiomaticConfig controls generation of the opinionated idiomatic layer.
type IdiomaticConfig struct {
	// Registry is the combined metadata for all frameworks.
	Registry *Registry
	// OutDir is the root output directory. Each framework writes to OutDir/<pkgname>/.
	// Canonical value: <repo-root>/bindings
	OutDir string
	// Frameworks is an optional filter; empty means all frameworks.
	Frameworks []string
	// Verbose enables diagnostic output.
	Verbose bool

	// Manifest, when non-nil, receives one parity entry per emitted construct so
	// the idiomatic layer's coverage can be checked against the raw oracle. It
	// never affects the emitted bytes.
	Manifest *emitmanifest.Recorder
}

// GenerateIdiomatic writes the opinionated idiomatic layer to cfg.OutDir.
// For each targeted framework it emits one *_generated.go wrapper file per ObjC
// class, bundling raw alloc+init calls into Go constructors and improving async,
// collection, and error-handling ergonomics.
func GenerateIdiomatic(cfg IdiomaticConfig) error {
	reg := cfg.Registry
	mapper := buildMapper(reg)
	// The set of value structs the idiomatic layer emits (computed once over
	// every framework) gates both struct emission and cross-framework struct
	// references so the two always agree.
	mapper.EmittableStructs = idiofw.ComputeEmittableStructs(reg.Frameworks, mapper)
	// The broader set (every struct physically emitted, incl. opaque/degraded) so
	// a cross-framework typedef alias can target a struct that exists even when it
	// is not in the all-clean-fields EmittableStructs subset.
	mapper.AllEmittedStructs = idiofw.ComputeAllEmittedStructNames(reg.Frameworks)
	// The idiomatic package names that are C libraries (they live under
	// bindings/libraries/, not bindings/frameworks/), so a cross-framework typedef
	// alias whose canonical owner is one of these imports it through the library
	// prefix. A LinkLib marks a plain C library rather than an ObjC framework.
	mapper.LibraryPkgs = make(map[string]bool)
	for _, fw := range reg.Frameworks {
		if fw.LinkLib != "" {
			mapper.LibraryPkgs[naming.PackageName(fw.Framework)] = true
		}
	}

	// The class index spans every framework the idiomatic layer emits — not
	// just the ones this invocation regenerates — so a partial regen resolves
	// cross-framework wrapper references identically to a full one.
	var emittable []*meta.FrameworkMeta
	for _, fw := range reg.Frameworks {
		if fw.IsSwiftOnly || len(fw.UmbrellaFor) > 0 || fw.LinkLib != "" {
			continue
		}
		emittable = append(emittable, fw)
	}
	mapper.IdiomaticClassIndex = idiofw.ComputeIdiomaticClassIndex(emittable, reg.OwnerIndex)

	// On a full regen (no framework filter) wipe the output tree so packages for
	// frameworks that left the metadata — or the raw packages this public tree
	// replaced during the switch — do not linger. The support packages and the
	// raw internal tree are siblings of OutDir (bindings/{internal,runtime}), so
	// they are untouched.
	if len(cfg.Frameworks) == 0 {
		if err := cleanDir(cfg.OutDir); err != nil {
			return fmt.Errorf("clean idiomatic frameworks dir: %w", err)
		}
	}

	// Emit the layer's support packages first so the whole idiomatic tree is
	// regenerable from scratch on every run. They are framework-independent and
	// live at the bindings root (the parent of the per-framework out dir): the
	// private ones under bindings/internal, the public runtime helpers under
	// bindings/runtime (see idiofw.supportFiles).
	if err := idiofw.EmitSupportPackages(filepath.Dir(cfg.OutDir)); err != nil {
		return fmt.Errorf("idiomatic support packages: %w", err)
	}

	filterSet := make(map[string]bool, len(cfg.Frameworks))
	for _, fw := range cfg.Frameworks {
		filterSet[strings.ToLower(fw)] = true
	}

	for _, fw := range reg.Frameworks {
		if fw.LinkLib != "" {
			continue // skip C libraries — idiomatic layer is for ObjC frameworks
		}
		if len(filterSet) > 0 && !filterSet[strings.ToLower(fw.Framework)] {
			continue
		}

		pkgName := naming.PackageName(fw.Framework)
		outDir := filepath.Join(cfg.OutDir, pkgName)

		// Swift-only frameworks (no ObjC surface) and umbrella frameworks
		// (re-exporting sub-frameworks) have no bridgeable API, but they keep a
		// doc-only package so their public import path resolves — matching the
		// raw layer, which emits the same stubs.
		if fw.IsSwiftOnly || len(fw.UmbrellaFor) > 0 {
			if err := emitIdiomaticStub(outDir, pkgName, fw); err != nil {
				return fmt.Errorf("idiomatic stub %s: %w", fw.Framework, err)
			}
			continue
		}

		rawPkgPath := reg.ModulePrefix + "/" + pkgName

		if err := idiofw.EmitFrameworkWrappers(
			outDir,
			pkgName,
			"raw",
			rawPkgPath,
			fw,
			mapper,
			reg.IdiomaticConfigIndex[fw.Framework],
			cfg.Manifest,
		); err != nil {
			return fmt.Errorf("idiomatic %s: %w", fw.Framework, err)
		}
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "idiomatic: emitted %s → %s\n", fw.Framework, outDir)
		}
	}

	// The per-framework bindings/frameworks/<framework> packages are themselves the
	// fluent entry points: callers import only the frameworks they use. No
	// all-frameworks aggregator is emitted — one would transitively import (and,
	// via each raw package's init, eagerly dlopen) every framework, defeating
	// minimal imports.
	return nil
}

// emitIdiomaticStub writes a doc-only package for a Swift-only or umbrella
// framework: it has no ObjC surface to bridge, but a package must exist so the
// import path resolves. removeGeneratedFiles first clears any stale output.
func emitIdiomaticStub(outDir, pkgName string, fw *meta.FrameworkMeta) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	reason := "is a Swift-only framework with no Objective-C surface to bridge"
	if len(fw.UmbrellaFor) > 0 {
		reason = "is an umbrella framework that only re-exports its sub-frameworks"
	}
	var buf bytes.Buffer
	fmt.Fprint(&buf, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n")
	fmt.Fprint(&buf, "//go:build darwin\n\n")
	fmt.Fprintf(&buf, "// Package %s %s, so it exposes no\n", pkgName, reason)
	fmt.Fprint(
		&buf,
		"// generated API. This doc-only package exists so the import path resolves.\n",
	)
	fmt.Fprintf(&buf, "package %s\n", pkgName)
	return rawfw.WriteGoFile(filepath.Join(outDir, "doc.go"), buf.Bytes())
}

// buildMapper constructs a type mapper from the registry.
func buildMapper(reg *Registry) *typemap.Mapper {
	return &typemap.Mapper{
		OwnerIndex:               reg.OwnerIndex,
		GenericClasses:           reg.GenericClasses,
		GenericParamIndex:        reg.GenericParamIndex,
		EnumIndex:                reg.EnumIndex,
		EnumGoTypeIndex:          reg.EnumGoTypeIndex,
		TypedefIndex:             reg.TypedefIndex,
		StructIndex:              reg.StructIndex,
		ProtocolIndex:            reg.ProtocolIndex,
		CFTypeIndex:              reg.CFTypeIndex,
		BlockedImports:           reg.BlockedImports,
		UnavailableClasses:       reg.UnavailableClasses,
		UnavailableEnumBaseTypes: reg.UnavailableEnumBaseTypes,
		ModulePrefix:             reg.ModulePrefix,
		LibraryModulePrefix:      reg.LibraryModulePrefix,
	}
}

// dylibVarName returns the package-level var name for the framework's dylib handle.
func dylibVarName(m *meta.FrameworkMeta) string {
	return "_" + strings.ToLower(m.Framework) + "Lib"
}

// frameworkDylibPath returns the macOS dylib path for loading via Dlopen.
func frameworkDylibPath(m *meta.FrameworkMeta) string {
	if m.LinkLib != "" {
		return "/usr/lib/lib" + m.LinkLib + ".dylib"
	}
	parent := m.Framework
	if m.ParentFramework != "" {
		parent = m.ParentFramework
	}
	return fmt.Sprintf(
		"/System/Library/Frameworks/%s.framework/%s",
		parent, parent,
	)
}
