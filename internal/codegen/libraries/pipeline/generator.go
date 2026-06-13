package pipeline

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/emit/library"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/emit/raw"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
	swiftemit "github.com/deploymenttheory/go-bindings-macosplatform/internal/swift/emit"
	swiftparser "github.com/deploymenttheory/go-bindings-macosplatform/internal/swift/parser"
)

// BindingsConfig controls raw Go binding generation: frameworks/ output plus
// the shared block and callback trampoline files in bindings/runtime/blocks and bindings/runtime/callbacks.
type BindingsConfig struct {
	// Registry is the combined metadata for all frameworks.
	Registry *Registry
	// FrameworksOutDir is the root output directory for ObjC framework packages.
	// Canonical value: <repo-root>/frameworks
	FrameworksOutDir string
	// LibrariesOutDir is the root output directory for C library packages.
	// When non-empty, frameworks with LinkLib set are written here instead of FrameworksOutDir.
	// Canonical value: <repo-root>/libraries
	LibrariesOutDir string
	// SDKPath is the macOS SDK root (e.g. from xcrun --show-sdk-path).
	// Used to locate .swiftinterface files for Swift-only frameworks.
	SDKPath string
	// BlocksDir overrides the output path for bindings/runtime/blocks generated files.
	// Defaults to defaultBlocksDir when empty. Tests use this to redirect output
	// to a temp directory without altering the canonical repo path.
	BlocksDir string
	// CallbacksDir overrides the output path for bindings/runtime/callbacks generated files.
	// Defaults to defaultCallbacksDir when empty. Tests use this to redirect output
	// to a temp directory without altering the canonical repo path.
	CallbacksDir string
	// Verbose enables diagnostic output for unsafe.Pointer type degradations.
	Verbose bool
	// Strict returns an error when any type degrades to unsafe.Pointer.
	Strict bool
	// IsNSStringOverloads enables Go-string convenience overloads for NSString * params.
	IsNSStringOverloads bool
	// DiagnosticsSink, when non-nil, receives every type-degradation diagnostic
	// recorded by the type mapper. The CLI uses this to enforce a committed
	// diagnostics baseline.
	DiagnosticsSink *[]string
}

// defaultBlocksDir and defaultCallbacksDir are the canonical paths (relative to the
// repo root) for the generated block and callback trampoline packages.
const (
	defaultBlocksDir    = "bindings/runtime/blocks"
	defaultCallbacksDir = "bindings/runtime/callbacks"
)

// CustomConfig controls generation of the opinionated/custom layer.
// Only *_generated.go files are written; hand-crafted files are never touched.
type CustomConfig struct {
	// Registry is the combined metadata for all frameworks.
	Registry *Registry
	// OutDir is the root output directory for the opinionated/custom layer.
	// Canonical value: <repo-root>/opinionated/custom
	OutDir string
	// Frameworks is an optional filter: when non-empty, only the named frameworks
	// are regenerated. Empty means regenerate all frameworks.
	Frameworks []string
	// Verbose enables diagnostic output.
	Verbose bool
}

// buildMapper constructs the shared type mapper from a loaded registry.
func buildMapper(reg *Registry, nsStringOverloads bool) *typemap.Mapper {
	m := typemap.New()
	m.GenericClasses = reg.GenericClasses
	m.GenericParamIndex = reg.GenericParamIndex
	m.OwnerIndex = reg.OwnerIndex
	m.EnumIndex = reg.EnumIndex
	m.EnumGoTypeIndex = reg.EnumGoTypeIndex
	m.TypedefIndex = reg.TypedefIndex
	m.StructIndex = reg.StructIndex
	m.CFTypeIndex = reg.CFTypeIndex
	m.ProtocolIndex = reg.ProtocolIndex
	m.ProtocolProxyIndex = reg.ProtocolProxyIndex
	m.ModulePrefix = reg.ModulePrefix
	m.BlockedImports = resolveBlockedImports(reg)
	m.IsNSStringOverloads = nsStringOverloads
	return m
}

// GenerateBindings writes frameworks/ output plus block and callback trampoline files.
// Frameworks are emitted in topological dependency order.
// The frameworks and libraries output directories are wiped before generation so
// renamed or removed frameworks do not leave stale packages behind.
func GenerateBindings(cfg BindingsConfig) error {
	reg := cfg.Registry
	m := buildMapper(reg, cfg.IsNSStringOverloads)

	// Clean-slate: remove the entire output trees so stale packages from renamed
	// or removed frameworks cannot persist across runs.
	if cfg.FrameworksOutDir != "" {
		if err := os.RemoveAll(cfg.FrameworksOutDir); err != nil {
			return fmt.Errorf("clean frameworks dir: %w", err)
		}
	}
	if cfg.LibrariesOutDir != "" {
		if err := os.RemoveAll(cfg.LibrariesOutDir); err != nil {
			return fmt.Errorf("clean libraries dir: %w", err)
		}
		// The bsd support package backs the typemap's POSIX/BSD struct
		// resolution (bsd.Timespec, bsd.EtherAddr, …). The libraries tree is
		// wiped above, so it is re-emitted on every run.
		bsdDir := filepath.Join(cfg.LibrariesOutDir, "bsd")
		if err := os.MkdirAll(bsdDir, 0o755); err != nil {
			return fmt.Errorf("mkdir bsd package dir: %w", err)
		}
		if err := writeFile(filepath.Join(bsdDir, "bsd.go"), func(buf *bytes.Buffer) error {
			return raw.EmitBSDPackage(buf)
		}); err != nil {
			return fmt.Errorf("generate bsd package: %w", err)
		}
	}

	blocksDir := cfg.BlocksDir
	if blocksDir == "" {
		blocksDir = defaultBlocksDir
	}
	callbacksDir := cfg.CallbacksDir
	if callbacksDir == "" {
		callbacksDir = defaultCallbacksDir
	}

	// Block trampolines.
	{
		frameworks := make([]*macosplatformmetadata.FrameworkMeta, 0, len(reg.Frameworks))
		frameworks = append(frameworks, reg.Frameworks...)
		sigs := raw.CollectBlockSignaturesFromFrameworks(frameworks, m)

		if err := writeFile(
			filepath.Join(blocksDir, "blocks_generated.go"),
			func(buf *bytes.Buffer) error {
				return raw.EmitRuntimeBlocksGo(buf, sigs, "blocks")
			},
		); err != nil {
			return fmt.Errorf("generate blocks_generated.go: %w", err)
		}
		if err := writeFile(
			filepath.Join(blocksDir, "block_trampolines_generated.h"),
			func(buf *bytes.Buffer) error {
				return raw.EmitRuntimeBlocksTrampolineHeader(buf, sigs)
			},
		); err != nil {
			return fmt.Errorf("generate block_trampolines_generated.h: %w", err)
		}
		if err := writeFile(
			filepath.Join(blocksDir, "block_trampolines_generated.m"),
			func(buf *bytes.Buffer) error {
				return raw.EmitRuntimeBlocksTrampolineImpl(buf, sigs)
			},
		); err != nil {
			return fmt.Errorf("generate block_trampolines_generated.m: %w", err)
		}
	}

	// Callback trampolines.
	{
		frameworks := make([]*macosplatformmetadata.FrameworkMeta, 0, len(reg.Frameworks))
		frameworks = append(frameworks, reg.Frameworks...)
		sigs := raw.CollectMethodSigsFromFrameworks(frameworks, m)

		if err := writeFile(
			filepath.Join(callbacksDir, "callbacks_generated.go"),
			func(buf *bytes.Buffer) error {
				return raw.EmitRuntimeCallbacksGo(buf, sigs, "callbacks")
			},
		); err != nil {
			return fmt.Errorf("generate callbacks_generated.go: %w", err)
		}
		if err := writeFile(
			filepath.Join(callbacksDir, "method_trampolines_generated.h"),
			func(buf *bytes.Buffer) error {
				return raw.EmitRuntimeCallbacksTrampolineHeader(buf, sigs)
			},
		); err != nil {
			return fmt.Errorf("generate method_trampolines_generated.h: %w", err)
		}
		if err := writeFile(
			filepath.Join(callbacksDir, "method_trampolines_generated.m"),
			func(buf *bytes.Buffer) error {
				return raw.EmitRuntimeCallbacksTrampolineImpl(buf, sigs)
			},
		); err != nil {
			return fmt.Errorf("generate method_trampolines_generated.m: %w", err)
		}
	}

	if err := forEachFramework(reg, nil, func(framework *macosplatformmetadata.FrameworkMeta) error {
		if err := emitFramework(cfg, framework, m, reg); err != nil {
			return fmt.Errorf("generate %s: %w", framework.Framework, err)
		}
		return nil
	}); err != nil {
		return err
	}

	if cfg.DiagnosticsSink != nil {
		*cfg.DiagnosticsSink = append(*cfg.DiagnosticsSink, m.Diagnostics...)
	}

	if (cfg.Verbose || cfg.Strict) && len(m.Diagnostics) > 0 {
		fmt.Fprintf(
			os.Stderr,
			"\n[codegen] %d type degradation(s) to unsafe.Pointer:\n",
			len(m.Diagnostics),
		)
		seen := make(map[string]bool, len(m.Diagnostics))
		unique := m.Diagnostics[:0]
		for _, d := range m.Diagnostics {
			if !seen[d] {
				seen[d] = true
				unique = append(unique, d)
			}
		}
		sort.Strings(unique)
		for _, d := range unique {
			fmt.Fprintf(os.Stderr, "  %s\n", d)
		}
	}

	if cfg.Strict && len(m.Diagnostics) > 0 {
		seen := make(map[string]bool, len(m.Diagnostics))
		n := 0
		for _, d := range m.Diagnostics {
			if !seen[d] {
				seen[d] = true
				n++
			}
		}
		return fmt.Errorf(
			"strict mode: %d unique type degradation(s) to unsafe.Pointer — run with -v to list them",
			n,
		)
	}

	return nil
}

// GenerateCustom writes *_generated.go files for the opinionated/custom layer.
// Hand-crafted files in the output directories are never deleted.
func GenerateCustom(cfg CustomConfig) error {
	reg := cfg.Registry
	m := buildMapper(reg, false)

	return forEachFramework(reg, cfg.Frameworks, func(framework *macosplatformmetadata.FrameworkMeta) error {
		if err := emitOpinionated(cfg, framework, m, reg); err != nil {
			return fmt.Errorf("opinionated %s: %w", framework.Framework, err)
		}
		return nil
	})
}

// forEachFramework calls fn for each framework in topological dependency order.
// When filter is non-empty, only frameworks whose lowercase name appears in the
// filter set are processed; an empty filter means all frameworks.
func forEachFramework(reg *Registry, filter []string, fn func(*macosplatformmetadata.FrameworkMeta) error) error {
	filterSet := make(map[string]bool, len(filter))
	for _, fw := range filter {
		filterSet[strings.ToLower(fw)] = true
	}
	for _, framework := range sortFrameworksByDependency(reg) {
		if len(filterSet) > 0 && !filterSet[strings.ToLower(framework.Framework)] {
			continue
		}
		if err := fn(framework); err != nil {
			return err
		}
	}
	return nil
}

// sortFrameworksByDependency returns frameworks ordered so that dependencies are generated
// before the frameworks that depend on them.
// A framework A depends on framework B if any class in A has a superclass that
// belongs to B (per OwnerIndex).
func sortFrameworksByDependency(reg *Registry) []*macosplatformmetadata.FrameworkMeta {
	// Build dependency edges: framework → set of frameworks it depends on.
	deps := make(map[string]map[string]bool)
	byName := make(map[string]*macosplatformmetadata.FrameworkMeta)
	for _, framework := range reg.Frameworks {
		byName[framework.Framework] = framework
		deps[framework.Framework] = make(map[string]bool)
	}
	for _, framework := range reg.Frameworks {
		for _, cls := range framework.Classes {
			if cls.Super == "" {
				continue
			}
			owner := reg.OwnerIndex[cls.Super]
			if owner != "" && owner != framework.Framework {
				deps[framework.Framework][owner] = true
			}
		}
	}

	// Kahn's algorithm.
	inDegree := make(map[string]int)
	for fw := range deps {
		if _, ok := inDegree[fw]; !ok {
			inDegree[fw] = 0
		}
		for dep := range deps[fw] {
			inDegree[dep] = inDegree[dep] // ensure present
			_ = dep
		}
	}
	// recalculate: inDegree[A] = number of frameworks that A depends on
	for fw := range deps {
		inDegree[fw] = len(deps[fw])
	}

	// Start with frameworks that have no dependencies.
	var queue []string
	for fw := range inDegree {
		if inDegree[fw] == 0 {
			queue = append(queue, fw)
		}
	}
	// Sort for determinism.
	sort.Strings(queue)

	var sorted []*macosplatformmetadata.FrameworkMeta
	for len(queue) > 0 {
		fw := queue[0]
		queue = queue[1:]
		if framework, ok := byName[fw]; ok {
			sorted = append(sorted, framework)
		}
		// Find frameworks that depended on fw.
		for other, otherDeps := range deps {
			if otherDeps[fw] {
				delete(otherDeps, fw)
				inDegree[other]--
				if inDegree[other] == 0 {
					queue = append(queue, other)
					sort.Strings(queue)
				}
			}
		}
	}

	// Append any remaining frameworks (cycles or unknown deps) in original order.
	inSorted := make(map[string]bool)
	for _, framework := range sorted {
		inSorted[framework.Framework] = true
	}
	for _, framework := range reg.Frameworks {
		if !inSorted[framework.Framework] {
			sorted = append(sorted, framework)
		}
	}
	return sorted
}

// unsupportedBridgeFrameworks is the set of frameworks whose headers are
// incompatible with the Objective-C bridge compilation model and therefore
// receive only an empty stub package (no CGo, no bridge files).
//
// DriverKit: C++ kernel-extension framework; its headers require C++ mode
//
//	which breaks CGo's own generated prolog code.
//
// Ruby, Tcl, Tk: language runtime headers; Tcl/Tk require X11 (not installed
//
//	by default), Ruby headers have ObjC-incompatible syntax.
//
// PCSC: smart-card framework; its wintypes.h typedef conflicts with stdint.h.
var unsupportedBridgeFrameworks = map[string]bool{
	"DriverKit": true,
	"Ruby":      true,
	"Tcl":       true,
	"Tk":        true,
	"PCSC":      true,
}

// emitFramework writes all files for a single framework into outDir/<packageName>/.
// C libraries (framework.LinkLib != "") are written to cfg.LibrariesOutDir instead.
func emitFramework(
	cfg BindingsConfig,
	framework *macosplatformmetadata.FrameworkMeta,
	m *typemap.Mapper,
	reg *Registry,
) error {
	isLibrary := framework.LinkLib != ""
	// Caller can opt out of one class by leaving the corresponding dir empty.
	if isLibrary && cfg.LibrariesOutDir == "" {
		return nil
	}
	if !isLibrary && cfg.FrameworksOutDir == "" {
		return nil
	}
	packageName := strings.ToLower(framework.Framework)
	rootDir := cfg.FrameworksOutDir
	if isLibrary && cfg.LibrariesOutDir != "" {
		rootDir = cfg.LibrariesOutDir
	}
	outDir := filepath.Join(rootDir, packageName)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	// Frameworks whose headers are incompatible with ObjC bridge compilation
	// get a minimal stub — just a package declaration, no CGo.
	if unsupportedBridgeFrameworks[framework.Framework] {
		return writeFile(filepath.Join(outDir, "doc.go"), func(buf *bytes.Buffer) error {
			fmt.Fprintf(buf, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n")
			fmt.Fprintf(buf, "//go:build darwin\n\n")
			fmt.Fprintf(
				buf,
				"// Package %s is a stub — the %s framework headers are\n",
				packageName,
				framework.Framework,
			)
			fmt.Fprintf(
				buf,
				"// incompatible with the Objective-C CGo bridge and are not generated.\n",
			)
			fmt.Fprintf(buf, "package %s\n", packageName)
			return nil
		})
	}

	pkgName := packageName

	// Swift-only frameworks: parse the .swiftinterface and emit pure-Go types.
	// Falls back to a documentation-only stub when no swiftinterface is found
	// or when no content was extracted (e.g. all types are macOS-unavailable).
	if framework.IsSwiftOnly {
		if cfg.SDKPath != "" {
			if swiftContent, _ := locateSwiftInterface(
				framework.Framework,
				cfg.SDKPath,
			); swiftContent != "" {
				swiftFM, err := swiftparser.ParseContent(framework.Framework, swiftContent)
				if err == nil && swiftFM.HasContent() {
					if emitErr := swiftemit.Emit(swiftFM, outDir, pkgName); emitErr != nil {
						return fmt.Errorf("swift emit %s: %w", framework.Framework, emitErr)
					}
					return writeFile(
						filepath.Join(outDir, "doc.go"),
						func(buf *bytes.Buffer) error {
							fmt.Fprintf(
								buf,
								"// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n",
							)
							fmt.Fprintf(
								buf,
								"// Package %s provides Go bindings for the macOS %s Swift framework.\n",
								pkgName,
								framework.Framework,
							)
							fmt.Fprintf(buf, "package %s\n", pkgName)
							return nil
						},
					)
				}
			}
		}
		return writeFile(filepath.Join(outDir, "doc.go"), func(buf *bytes.Buffer) error {
			fmt.Fprintf(buf, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n")
			fmt.Fprintf(
				buf,
				"// Package %s would provide Go bindings for the macOS %s framework,\n",
				pkgName,
				framework.Framework,
			)
			fmt.Fprintf(
				buf,
				"// but %s is implemented entirely in Swift and exposes no Objective-C\n",
				framework.Framework,
			)
			fmt.Fprintf(
				buf,
				"// surface that can be bridged via CGo. Swift-to-Go interop requires\n",
			)
			fmt.Fprintf(
				buf,
				"// the Swift/C interoperability layer which is outside the scope of this\n",
			)
			fmt.Fprintf(buf, "// code generator.\n")
			fmt.Fprintf(buf, "package %s\n", pkgName)
			return nil
		})
	}
	if len(framework.UmbrellaFor) > 0 {
		return writeFile(filepath.Join(outDir, "doc.go"), func(buf *bytes.Buffer) error {
			fmt.Fprintf(buf, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n")
			fmt.Fprintf(
				buf,
				"// Package %s corresponds to the macOS %s umbrella framework,\n",
				pkgName,
				framework.Framework,
			)
			fmt.Fprintf(buf, "// which re-exports the following constituent frameworks:\n")
			for _, sub := range framework.UmbrellaFor {
				fmt.Fprintf(buf, "//   - %s\n", sub)
			}
			fmt.Fprintf(buf, "//\n")
			fmt.Fprintf(
				buf,
				"// Import the constituent framework packages directly for their APIs.\n",
			)
			fmt.Fprintf(buf, "package %s\n", pkgName)
			return nil
		})
	}

	// doc.go
	if err := writeFile(filepath.Join(outDir, "doc.go"), func(buf *bytes.Buffer) error {
		fmt.Fprintf(buf, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n")
		fmt.Fprintf(
			buf,
			"// Package %s provides Go bindings for the macOS %s framework.\n",
			pkgName,
			framework.Framework,
		)
		fmt.Fprintf(buf, "package %s\n", pkgName)
		return nil
	}); err != nil {
		return err
	}

	// cgo.go — link flags for this framework only.
	if err := writeFile(filepath.Join(outDir, "cgo.go"), func(buf *bytes.Buffer) error {
		fmt.Fprintf(buf, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n")
		fmt.Fprintf(buf, "//go:build darwin\n\n")
		fmt.Fprintf(buf, "package %s\n\n", pkgName)
		fmt.Fprintf(
			buf,
			"// #cgo CFLAGS: -fno-objc-arc -x objective-c -Werror -Wno-deprecated-declarations\n",
		)
		if framework.LinkLib != "" {
			// C libraries (e.g. EndpointSecurity) link with -l rather than -framework.
			fmt.Fprintf(buf, "// #cgo LDFLAGS: -l%s\n", framework.LinkLib)
		} else {
			// Sub-frameworks must link via their parent (e.g. -framework Carbon for HIToolbox).
			linkFW := framework.Framework
			if framework.ParentFramework != "" {
				linkFW = framework.ParentFramework
			}
			fmt.Fprintf(buf, "// #cgo LDFLAGS: -framework %s\n", linkFW)
		}
		if err := os.MkdirAll(filepath.Join(outDir, "bridge"), 0o755); err != nil {
			return err
		}
		fmt.Fprintf(buf, "// #include %q\n", "bridge/"+packageName+"_bridge.h")
		fmt.Fprintf(buf, "import \"C\"\n")
		return nil
	}); err != nil {
		return err
	}

	// {fw}_enums.go
	if len(framework.Enums) > 0 {
		if err := writeFile(
			filepath.Join(outDir, packageName+"_enums.go"),
			func(buf *bytes.Buffer) error {
				needsFmt, needsStrings := raw.EnumsNeedImports(framework)
				writeGoHeaderEnums(buf, pkgName, needsFmt, needsStrings)
				return raw.EmitEnums(buf, framework)
			},
		); err != nil {
			return err
		}
	}

	// {fw}_structs.go — two-phase: generate body first to detect import needs.
	if len(framework.Structs) > 0 || len(framework.Typedefs) > 0 {
		if err := writeFile(
			filepath.Join(outDir, packageName+"_structs.go"),
			func(buf *bytes.Buffer) error {
				var body bytes.Buffer
				crossImports, err := raw.EmitStructs(&body, framework, m, reg.ClassNameIndex)
				if err != nil {
					return err
				}
				needsUnsafe := bytes.Contains(body.Bytes(), []byte("unsafe.Pointer"))
				needsObjc := bytes.Contains(body.Bytes(), []byte("cgo.Track"))
				writeGoHeaderStructsFull(buf, pkgName, needsUnsafe, needsObjc, crossImports)
				_, err = buf.Write(body.Bytes())
				return err
			},
		); err != nil {
			return err
		}
	}

	// {fw}_externs.go
	if len(framework.Externs) > 0 {
		if err := writeFile(
			filepath.Join(outDir, packageName+"_externs.go"),
			func(buf *bytes.Buffer) error {
				return raw.EmitExterns(buf, pkgName, framework, m, reg.ClassNameIndex)
			},
		); err != nil {
			return err
		}
	}

	// {fw}_protocols.go
	if len(framework.Protocols) > 0 {
		if err := writeFile(
			filepath.Join(outDir, packageName+"_protocols.go"),
			func(buf *bytes.Buffer) error {
				return raw.EmitProtocols(
					buf,
					pkgName,
					framework,
					m,
					reg.ClassNameIndex,
					reg.ProtocolIndex,
					reg.ClassIndex,
				)
			},
		); err != nil {
			return err
		}
	}

	// {fw}_proxies.go — id<Protocol> wrapper types for return-position protocols.
	if len(framework.Protocols) > 0 {
		if err := writeFile(
			filepath.Join(outDir, packageName+"_proxies.go"),
			func(buf *bytes.Buffer) error {
				return raw.EmitProtocolProxies(
					buf,
					pkgName,
					packageName,
					framework,
					m,
					reg.ClassNameIndex,
					reg.ClassIndex,
				)
			},
		); err != nil {
			return err
		}
	}

	// {fw}_class_interfaces.go — [ClassName]able Go interface per concrete ObjC class.
	if len(framework.Classes) > 0 {
		if err := writeFile(
			filepath.Join(outDir, packageName+"_class_interfaces.go"),
			func(buf *bytes.Buffer) error {
				return raw.EmitClassInterfaces(
					buf,
					pkgName,
					framework,
					m,
					reg.ClassNameIndex,
					reg.ClassIndex,
				)
			},
		); err != nil {
			return err
		}
	}

	// {fw}_block_types.go
	if len(framework.BlockTypes) > 0 {
		if err := writeFile(
			filepath.Join(outDir, packageName+"_block_types.go"),
			func(buf *bytes.Buffer) error {
				writeGoHeader(buf, pkgName, false)
				return raw.EmitBlocks(buf, framework, m, reg.ClassNameIndex)
			},
		); err != nil {
			return err
		}
	}

	// {fw}_functions.go
	if hasBridgeableFunctions(framework) {
		if err := writeFile(
			filepath.Join(outDir, packageName+"_functions.go"),
			func(buf *bytes.Buffer) error {
				return raw.EmitFunctions(
					buf,
					pkgName,
					packageName,
					framework,
					m,
					reg.ClassNameIndex,
				)
			},
		); err != nil {
			return err
		}
	}

	// bridge/{fw}_bridge.h and bridge/{fw}_bridge.m
	if err := raw.EmitBridge(outDir, framework, m, reg.ClassNameIndex); err != nil {
		return fmt.Errorf("bridge %s: %w", framework.Framework, err)
	}

	// One .go file per class.
	if err := raw.EmitClasses(
		outDir,
		framework,
		m,
		reg.ClassNameIndex,
		reg.ClassIndex,
		pkgName,
	); err != nil {
		return fmt.Errorf("classes %s: %w", framework.Framework, err)
	}

	// foundation_variadic.go — static variadic collection helpers, only for Foundation.
	if strings.EqualFold(framework.Framework, "Foundation") {
		if err := writeFile(
			filepath.Join(outDir, "foundation_variadic.go"),
			func(buf *bytes.Buffer) error {
				return raw.EmitFoundationVariadicWrappers(buf, pkgName)
			},
		); err != nil {
			return err
		}
	}

	superIndex := reg.SuperclassIndex()
	if err := raw.EmitSubclassFactories(
		outDir,
		framework,
		m,
		superIndex,
		reg.ClassIndex,
	); err != nil {
		return fmt.Errorf("subclass factories %s: %w", framework.Framework, err)
	}
	if err := raw.EmitProtocolImpls(outDir, framework, m); err != nil {
		return fmt.Errorf("protocol impls %s: %w", framework.Framework, err)
	}
	if err := raw.EmitGeneratedBridgesImpl(outDir, packageName); err != nil {
		return fmt.Errorf("generated bridges impl %s: %w", framework.Framework, err)
	}

	return nil
}

// emitOpinionated writes *_generated.go files for the opinionated layer for
// one framework. Only files matching *_generated.go are removed on each run;
// hand-crafted files in the same directory are never touched.
// Swift-only, umbrella, and unsupported-bridge frameworks are skipped silently.
func emitOpinionated(
	cfg CustomConfig,
	framework *macosplatformmetadata.FrameworkMeta,
	m *typemap.Mapper,
	reg *Registry,
) error {
	if framework.IsSwiftOnly || len(framework.UmbrellaFor) > 0 ||
		unsupportedBridgeFrameworks[framework.Framework] {
		return nil
	}

	packageName := strings.ToLower(framework.Framework)
	opDir := filepath.Join(cfg.OutDir, packageName)

	// Remove stale *_generated.go files; leave all other files untouched.
	if entries, err := os.ReadDir(opDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), "_generated.go") {
				_ = os.Remove(filepath.Join(opDir, e.Name()))
			}
		}
	}

	if err := os.MkdirAll(opDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", opDir, err)
	}

	rawImportPath := m.ModulePrefix + "/" + packageName

	if err := writeFile(
		filepath.Join(opDir, packageName+"_async_generated.go"),
		func(buf *bytes.Buffer) error {
			return library.EmitAsync(
				buf,
				packageName,
				rawImportPath,
				framework,
				m,
				reg.ClassNameIndex,
			)
		},
	); err != nil {
		return err
	}
	if err := writeFile(
		filepath.Join(opDir, packageName+"_slices_generated.go"),
		func(buf *bytes.Buffer) error {
			return library.EmitSlices(
				buf,
				packageName,
				rawImportPath,
				framework,
				m,
				reg.ClassNameIndex,
			)
		},
	); err != nil {
		return err
	}
	if err := writeFile(
		filepath.Join(opDir, packageName+"_specs_generated.go"),
		func(buf *bytes.Buffer) error {
			return library.EmitSpecs(
				buf,
				packageName,
				rawImportPath,
				framework,
				m,
				reg.ClassNameIndex,
			)
		},
	); err != nil {
		return err
	}
	return nil
}

// writeFile creates a file, invokes fn with a buffer, then writes the buffer.
func writeFile(path string, fn func(*bytes.Buffer) error) error {
	var buf bytes.Buffer
	if err := fn(&buf); err != nil {
		return fmt.Errorf("generate %s: %w", filepath.Base(path), err)
	}
	if buf.Len() == 0 {
		// Emitter produced nothing (e.g. all classes skipped); skip the file so
		// we don't leave an empty .go file that the compiler rejects.
		return nil
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// writeGoHeader writes the standard generated-file header for a plain Go file.
func writeGoHeader(buf *bytes.Buffer, pkgName string, needsUnsafe bool) {
	fmt.Fprintf(buf, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n")
	fmt.Fprintf(buf, "//go:build darwin\n\n")
	fmt.Fprintf(buf, "package %s\n\n", pkgName)
	if needsUnsafe {
		fmt.Fprintf(buf, "import \"unsafe\"\n\n")
	}
}

// writeGoHeaderEnums writes the header for an enum file, importing "fmt" when
// there are non-bitmask named enums (needed for String() default clause) and
// "strings" when there are bitmask enums (Join/Split).
func writeGoHeaderEnums(buf *bytes.Buffer, pkgName string, needsFmt, needsStrings bool) {
	fmt.Fprintf(buf, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n")
	fmt.Fprintf(buf, "//go:build darwin\n\n")
	fmt.Fprintf(buf, "package %s\n\n", pkgName)
	var imports []string
	if needsFmt {
		imports = append(imports, `"fmt"`)
	}
	if needsStrings {
		imports = append(imports, `"strings"`)
	}
	switch len(imports) {
	case 0:
	case 1:
		fmt.Fprintf(buf, "import %s\n\n", imports[0])
	default:
		fmt.Fprintf(buf, "import (\n")
		for _, imp := range imports {
			fmt.Fprintf(buf, "\t%s\n", imp)
		}
		fmt.Fprintf(buf, ")\n\n")
	}
}

// writeGoHeaderStructsFull writes the header for a structs file, adding whatever
// imports are needed: unsafe, objc, and any cross-framework packages.
func writeGoHeaderStructsFull(
	buf *bytes.Buffer,
	pkgName string,
	needsUnsafe, needsObjc bool,
	crossImports typemap.ImportSet,
) {
	fmt.Fprintf(buf, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n")
	fmt.Fprintf(buf, "//go:build darwin\n\n")
	fmt.Fprintf(buf, "package %s\n\n", pkgName)

	// Collect all imports needed.
	var allImports []string
	if needsUnsafe || needsObjc {
		allImports = append(allImports, "\"unsafe\"")
	}
	if needsObjc {
		allImports = append(
			allImports,
			"\"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/cgo\"",
		)
	}
	for _, path := range crossImports {
		allImports = append(allImports, "\""+path+"\"")
	}
	if len(allImports) == 0 {
		return
	}
	sort.Strings(allImports)
	// Deduplicate.
	seen := make(map[string]bool)
	var deduped []string
	for _, imp := range allImports {
		if !seen[imp] {
			seen[imp] = true
			deduped = append(deduped, imp)
		}
	}
	if len(deduped) == 1 {
		fmt.Fprintf(buf, "import %s\n\n", deduped[0])
		return
	}
	fmt.Fprintf(buf, "import (\n")
	for _, imp := range deduped {
		fmt.Fprintf(buf, "\t%s\n", imp)
	}
	fmt.Fprintf(buf, ")\n\n")
}

// writeGoHeaderRuntime writes a file header importing unsafe and the objc package.
func writeGoHeaderRuntime(buf *bytes.Buffer, pkgName string) {
	fmt.Fprintf(buf, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n")
	fmt.Fprintf(buf, "//go:build darwin\n\n")
	fmt.Fprintf(buf, "package %s\n\n", pkgName)
	fmt.Fprintf(buf, "import (\n")
	fmt.Fprintf(buf, "\t\"unsafe\"\n")
	fmt.Fprintf(buf, "\t\"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/cgo\"\n")
	fmt.Fprintf(buf, ")\n\n")
	fmt.Fprintf(buf, "var _ unsafe.Pointer   // suppress unused import\n")
	fmt.Fprintf(buf, "var _ cgo.Object = nil // suppress unused import\n\n")
}

// writeGoHeaderCGO writes the header for a file that makes CGo calls.
func writeGoHeaderCGO(buf *bytes.Buffer, pkgName, packageName string) {
	fmt.Fprintf(buf, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n")
	fmt.Fprintf(buf, "//go:build darwin\n\n")
	fmt.Fprintf(buf, "package %s\n\n", pkgName)
	fmt.Fprintf(buf, "// #include %q\n", "bridge/"+packageName+"_bridge.h")
	fmt.Fprintf(buf, "import \"C\"\n\n")
	fmt.Fprintf(
		buf,
		"import (\n\t\"unsafe\"\n\t\"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/cgo\"\n)\n\n",
	)
}

// reCls extracts bare ObjC class names from a type string for cycle detection.
var reCls = regexp.MustCompile(`\b([A-Z][A-Za-z0-9]+)\b`)

// resolveBlockedImports analyses all ObjC type strings in reg, builds a
// would-import graph, detects import cycles of any length, and returns a map of
// (sourceFramework → set of targetFrameworks) that should fall back to
// unsafe.Pointer to break those cycles.
//
// Edge selection uses two independent weight classes:
//   - methodWeight[A][B]: class/method/enum type references from A to B (high cost to suppress)
//   - protoWeight[A][B]:  protocol Implements references from A to B (low cost to suppress)
//
// When breaking a cycle, edges that are protocol-embed-only (methodWeight == 0) are
// preferred over method-type edges. Within the same class, the lower-total-weight edge
// is broken first; ties resolve by preferring the lexicographically larger source
// framework name. This is deterministic: adjacency lists are sorted, DFS starts from
// sorted keys, and the selection criterion is a stable total order.
func resolveBlockedImports(reg *Registry) map[string]map[string]bool {
	// methodWeight[A][B]: class/method/enum type references in A pointing to types owned by B.
	// protoWeight[A][B]:  protocol Implements references in A that name protocols owned by B.
	methodWeight := make(map[string]map[string]int)
	protoWeight := make(map[string]map[string]int)

	for _, framework := range reg.Frameworks {
		fw := framework.Framework
		if methodWeight[fw] == nil {
			methodWeight[fw] = make(map[string]int)
		}
		if protoWeight[fw] == nil {
			protoWeight[fw] = make(map[string]int)
		}

		// addMethodWeight counts every class or enum type reference in an ObjC type string.
		// It also resolves single-token lowercase typedef names (e.g. dispatch_queue_t →
		// NSObject<OS_dispatch_queue> *) so that typedef-derived cross-framework edges are
		// captured in the cycle graph, preventing import cycles from typedef resolution.
		typedefSeen := make(map[string]bool)
		var addMethodWeight func(objcType string)
		addMethodWeight = func(objcType string) {
			for _, tok := range reCls.FindAllString(objcType, -1) {
				owner := reg.OwnerIndex[tok]
				if owner == "" {
					owner = reg.EnumIndex[tok] // enums create real import edges too
				}
				if owner == "" {
					// Typedef ownership: CF opaque-pointer typedefs (CFRunLoopRef,
					// CFStringRef …) are emitted as *corefoundation.CFXxxRef by the
					// type mapper. Treat that as an import edge so cycle detection
					// breaks cycles that involve toll-free-bridged classes.
					owner = reg.TypedefOwnerIndex[tok]
				}
				if owner == "" {
					// Protocol-typed parameters (id<SomeProtocol>) create import edges
					// to the framework that defines the protocol.
					owner = reg.ProtocolIndex[tok]
				}
				// StructIndex takes precedence over TypedefOwnerIndex: many frameworks
				// transitively include headers that define struct typedefs (FSRef, CGSize),
				// so "first write wins" in TypedefOwnerIndex attributes them to the wrong
				// framework. The mapper resolves structs via StructIndex, so cycle
				// detection must use the same source for accurate import-graph edges.
				if structOwner := reg.StructIndex[tok]; structOwner != "" {
					owner = structOwner
				}
				if owner != "" && owner != fw {
					methodWeight[fw][owner]++
				}
			}
			// Resolve single-token typedef names to arbitrary depth.
			// Normalise first to strip _Null_unspecified etc. so "dispatch_queue_t _Null_unspecified"
			// → "dispatch_queue_t" before the simple-identifier guard.
			// Handles both lowercase (dispatch_queue_t → NSObject<OS_dispatch_queue> *) and
			// uppercase (VZMemorySize → NSInteger) typedefs, including multi-hop chains where
			// each expansion is recursively processed. typedefSeen guards against cycles.
			n := typemap.Normalise(objcType)
			if !strings.ContainsAny(n, " *<>^()") && len(n) > 0 {
				if target, ok := reg.TypedefIndex[n]; ok && target != n && !typedefSeen[n] {
					typedefSeen[n] = true
					addMethodWeight(target)
				}
			}
		}

		for _, cls := range framework.Classes {
			for _, method := range cls.Methods {
				addMethodWeight(method.Return.ObjCType)
				for _, arg := range method.Params {
					addMethodWeight(arg.ObjCType)
				}
			}
		}
		for _, proto := range framework.Protocols {
			for _, method := range proto.Methods {
				addMethodWeight(method.Return.ObjCType)
				for _, arg := range method.Params {
					addMethodWeight(arg.ObjCType)
				}
			}
			// Protocol InheritedProtocols relationships are protocol-embed edges, tracked separately.
			for _, parentName := range proto.InheritedProtocols {
				owner := reg.ProtocolIndex[parentName]
				if owner != "" && owner != fw {
					protoWeight[fw][owner]++
				}
			}
		}
		for _, fn := range framework.Functions {
			addMethodWeight(fn.Return.ObjCType)
			for _, arg := range fn.Params {
				addMethodWeight(arg.ObjCType)
			}
		}
		for _, e := range framework.Externs {
			addMethodWeight(e.ObjCType)
		}
		// Foreign extensions: the receiver class creates a method-weight dependency.
		for className := range framework.ForeignExtensions {
			owner := reg.OwnerIndex[className]
			if owner != "" && owner != fw {
				methodWeight[fw][owner]++
			}
		}
	}

	// Merge into a total-weight map for cycle edge selection, but retain the
	// per-class weights for tie-breaking.
	totalWeight := make(map[string]map[string]int)
	allFWs := make(map[string]bool)
	for fw := range methodWeight {
		allFWs[fw] = true
	}
	for fw := range protoWeight {
		allFWs[fw] = true
	}
	for fw := range allFWs {
		totalWeight[fw] = make(map[string]int)
		for tgt, w := range methodWeight[fw] {
			totalWeight[fw][tgt] += w
		}
		for tgt, w := range protoWeight[fw] {
			totalWeight[fw][tgt] += w
		}
	}

	// Build directed would-import adjacency list (edges with total weight > 0).
	// Sort each adjacency list for deterministic DFS traversal.
	adj := make(map[string][]string)
	for fw, targets := range totalWeight {
		var edges []string
		for tgt, w := range targets {
			if w > 0 && tgt != fw {
				edges = append(edges, tgt)
			}
		}
		sort.Strings(edges)
		adj[fw] = edges
	}

	blocked := make(map[string]map[string]bool)
	ensureBlocked := func(src, tgt string) {
		if blocked[src] == nil {
			blocked[src] = make(map[string]bool)
		}
		blocked[src][tgt] = true
	}

	// edgeScore returns a comparable score for an edge (src→dst). Lower scores are
	// broken first. Priority order:
	//  1. Edges with no method-weight (protocol-embed-only) beat edges with method-weight.
	//  2. Among same priority, lower total weight is broken first.
	//  3. Ties resolve by preferring the lexicographically larger source name.
	edgeScore := func(src, dst string) (prioClass int, total int, srcName string) {
		if methodWeight[src][dst] == 0 {
			prioClass = 0 // protocol-embed-only: preferred to break
		} else {
			prioClass = 1 // has method references: avoid breaking if possible
		}
		return prioClass, totalWeight[src][dst], src
	}

	// Repeatedly find and break cycles until the graph is acyclic.
	for {
		cycle := detectImportCycle(adj)
		if cycle == nil {
			break
		}
		// Select the edge in the cycle with the lowest score.
		var bestSrc, bestDst string
		bestPrio, bestTotal, bestSrcName := -1, -1, ""
		for i, fw := range cycle {
			dst := cycle[(i+1)%len(cycle)]
			p, t, s := edgeScore(fw, dst)
			if bestPrio < 0 ||
				p < bestPrio ||
				(p == bestPrio && t < bestTotal) ||
				(p == bestPrio && t == bestTotal && s > bestSrcName) {
				bestPrio, bestTotal, bestSrcName = p, t, s
				bestSrc, bestDst = fw, dst
			}
		}
		ensureBlocked(bestSrc, bestDst)
		// Remove the blocked edge from adj so the next iteration sees a smaller graph.
		updated := adj[bestSrc][:0]
		for _, tgt := range adj[bestSrc] {
			if tgt != bestDst {
				updated = append(updated, tgt)
			}
		}
		adj[bestSrc] = updated
	}

	return blocked
}

// detectImportCycle finds a cycle in the directed adjacency graph using DFS.
// It returns the cycle as a node slice in traversal order, or nil if the graph
// is acyclic. Node order within the adjacency lists must be deterministic to
// produce reproducible cycle selections across runs.
func detectImportCycle(adj map[string][]string) []string {
	const (
		unvisited = 0
		inStack   = 1
		done      = 2
	)
	state := make(map[string]int)
	var stack []string

	var dfs func(fw string) []string
	dfs = func(fw string) []string {
		state[fw] = inStack
		stack = append(stack, fw)
		for _, tgt := range adj[fw] {
			switch state[tgt] {
			case inStack:
				// Back edge — extract the cycle from the stack.
				for start, n := range stack {
					if n == tgt {
						cycle := make([]string, len(stack)-start)
						copy(cycle, stack[start:])
						return cycle
					}
				}
			case unvisited:
				if cycle := dfs(tgt); cycle != nil {
					return cycle
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[fw] = done
		return nil
	}

	// Iterate in sorted order for determinism across runs.
	fws := make([]string, 0, len(adj))
	for fw := range adj {
		fws = append(fws, fw)
	}
	sort.Strings(fws)

	for _, fw := range fws {
		if state[fw] == unvisited {
			if cycle := dfs(fw); cycle != nil {
				return cycle
			}
		}
	}
	return nil
}

// locateSwiftInterface locates and reads the .swiftinterface file for a
// framework under the given SDK path. Returns the file contents and path, or
// empty strings if no .swiftinterface is found.
func locateSwiftInterface(framework, sdkPath string) (content, path string) {
	base := filepath.Join(sdkPath, "System", "Library", "Frameworks",
		framework+".framework", "Modules", framework+".swiftmodule")

	for _, arch := range []string{"arm64e-apple-macos", "arm64-apple-macos", "x86_64-apple-macos"} {
		p := filepath.Join(base, arch+".swiftinterface")
		if data, err := os.ReadFile(p); err == nil {
			return string(data), p
		}
	}

	matches, _ := filepath.Glob(filepath.Join(base, "*.swiftinterface"))
	for _, m := range matches {
		if data, err := os.ReadFile(m); err == nil {
			return string(data), m
		}
	}
	return "", ""
}

// hasBridgeableFunctions returns true if the framework has at least one free
// function that can be bridged (not inline, not va_list-only).
func hasBridgeableFunctions(framework *macosplatformmetadata.FrameworkMeta) bool {
	for _, fn := range framework.Functions {
		if fn.IsInline {
			continue
		}
		hasVA := false
		for _, arg := range fn.Params {
			n := strings.ToLower(arg.ObjCType)
			if strings.Contains(n, "va_list") || strings.Contains(n, "__va_list") {
				hasVA = true
				break
			}
		}
		if !hasVA {
			return true
		}
	}
	return false
}
