//go:build darwin

// generate scans macOS SDK framework headers and emits Go bindings.
//
// Sub-commands:
//
//	scan             — Clang → .gometa.json (metadata cache)
//	bindings         — .gometa.json → bindings/frameworks/ (purego ObjC) + bindings/libraries/ (CGo C)
//	class-hierarchy  — .gometa.json → metadata/objcclasshierarchy/objc_class_hierarchy_generated.go
//	all              — scan (optional) + bindings in sequence
//	list             — list all frameworks in the installed Xcode SDK
//	idiomatic        — emit opinionated idiomatic layer (Go-friendly wrappers)
//
// Run any sub-command with -help for available flags.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	purepipeline "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/pipeline"
	cgopipeline "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/pipeline"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/diagnostics"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/metadiff"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/scanner"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/validate"
)

const (
	// The raw (purego/CGo) bindings live under bindings/internal/raw so Go's
	// internal-package rule makes them unreachable to external consumers. The
	// idiomatic layer at bindings/{frameworks,libraries} is the only public API.
	defaultFrameworksModulePrefix = "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/frameworks"
	defaultLibrariesModulePrefix  = "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries"

	// Filesystem output roots matching the module prefixes above.
	defaultRawFrameworksOutDir = "./bindings/internal/raw/frameworks"
	defaultRawLibrariesOutDir  = "./bindings/internal/raw/libraries"

	// The public idiomatic layer's output roots.
	defaultIdiomaticFrameworksOutDir = "./bindings/frameworks"
	defaultIdiomaticLibrariesOutDir  = "./bindings/libraries"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "scan":
		runScan(os.Args[2:])
	case "bindings":
		runBindings(os.Args[2:])
	case "all":
		runAll(os.Args[2:])
	case "class-hierarchy":
		runClassHierarchy(os.Args[2:])
	case "list":
		runList(os.Args[2:])
	case "validate":
		runValidate(os.Args[2:])
	case "diff":
		runDiff(os.Args[2:])
	case "idiomatic":
		runIdiomatic(os.Args[2:])
	case "parity":
		runParity(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown sub-command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprint(os.Stderr, `usage: generate <sub-command> [flags]

Sub-commands:
  scan             Scan SDK headers → .gometa.json metadata cache
  bindings         Emit bindings: purego ObjC frameworks + CGo C libraries
  class-hierarchy  Derive ObjC class hierarchy → metadata/objcclasshierarchy/
  all              Run scan (optional) + bindings in sequence
  list             List all frameworks available in the installed Xcode SDK
  validate         Run structural integrity checks over committed metadata
  diff             Semantic API diff between two metadata trees
  idiomatic        Emit opinionated idiomatic layer (Go-friendly wrappers)
  parity           Report raw constructs the idiomatic emitter does not yet emit

Run 'generate <sub-command> -help' for flags.
`)
}

// ---------------------------------------------------------------------------
// idiomatic
// ---------------------------------------------------------------------------

func runIdiomatic(args []string) {
	fs := flag.NewFlagSet("idiomatic", flag.ExitOnError)
	framework := fs.String("framework", "", `Framework/library(s) to emit: name, comma-separated list, or "all" (default: all)`)
	metaDir := fs.String("metadata-dir", "./metadata", "Directory containing .gometa.json files")
	out := fs.String("out", defaultIdiomaticFrameworksOutDir, "Output directory for idiomatic ObjC-framework packages")
	librariesOut := fs.String("libraries-out", defaultIdiomaticLibrariesOutDir, "Output directory for idiomatic CGo C-library packages")
	clibraries := fs.String("clibraries", "./metadata/clibraries.json", "C library registry JSON (falls back to built-in defaults when absent)")
	verbose := fs.Bool("v", false, "Verbose output")
	_ = fs.Parse(args)

	// The registry's per-library "backend" key changes the raw Go spelling the
	// idiomatic wrappers must reference (rawlib.FunctionGoName).
	loadCLibraryRegistry(*clibraries, *verbose)

	var names []string
	if *framework != "" && !strings.EqualFold(*framework, "all") {
		names = splitTrimmed(*framework, ",")
	}

	// ObjC frameworks → bindings/frameworks (purego pipeline).
	reg := loadPureRegistry(*metaDir, defaultFrameworksModulePrefix, defaultLibrariesModulePrefix)
	if err := purepipeline.GenerateIdiomatic(purepipeline.IdiomaticConfig{
		Registry:   reg,
		OutDir:     *out,
		Frameworks: names,
		Verbose:    *verbose,
	}); err != nil {
		log.Fatalf("idiomatic (frameworks): %v", err)
	}
	log.Printf("idiomatic frameworks: done → %s", *out)

	// CGo C libraries → bindings/libraries (CGo pipeline).
	cgoReg := loadCGORegistry(*metaDir, defaultLibrariesModulePrefix)
	if err := cgopipeline.GenerateIdiomaticLibraries(cgopipeline.IdiomaticConfig{
		Registry:      cgoReg,
		OutDir:        *librariesOut,
		Libraries:     names,
		Verbose:       *verbose,
		RawSourceRoot: defaultRawLibrariesOutDir, // read raw source from bindings/internal/raw/libraries
	}); err != nil {
		log.Fatalf("idiomatic (libraries): %v", err)
	}
	log.Printf("idiomatic libraries: done → %s", *librariesOut)
}

// ---------------------------------------------------------------------------
// scan
// ---------------------------------------------------------------------------

func runScan(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	framework := fs.String("framework", "", `Framework(s) to scan: name, comma-separated list, or "all" (required)`)
	metaDir := fs.String("metadata-dir", "./metadata", "Directory for .gometa.json files (committed metadata)")
	sdkPath := fs.String("sdk", "", "macOS SDK path (default: auto-detected via xcrun)")
	sdkVersion := fs.String("sdk-version", "", "SDK version string (default: auto-detected)")
	arch := fs.String("arch", "arm64", "Target architecture: arm64, x86_64, or comma-separated")
	parallel := fs.Int("parallel", runtime.NumCPU(), "Number of frameworks to scan concurrently")
	clibraries := fs.String("clibraries", "./metadata/clibraries.json", "C library registry JSON (falls back to built-in defaults when absent)")
	verbose := fs.Bool("v", false, "Verbose output")
	_ = fs.Parse(args)

	if *framework == "" {
		fmt.Fprintln(os.Stderr, "scan: --framework is required")
		fs.Usage()
		os.Exit(1)
	}

	loadCLibraryRegistry(*clibraries, *verbose)
	sdk := detectSDK(*sdkPath, *verbose)
	ver := detectSDKVersion(*sdkVersion)
	doScan(*framework, sdk, ver, *metaDir, *arch, *parallel, *verbose)
}

// loadCLibraryRegistry swaps the scanner's built-in C library defaults for the
// committed registry file when present, and likewise the per-framework scan
// config that lives beside it.
func loadCLibraryRegistry(path string, verbose bool) {
	loaded, err := scanner.LoadCLibrariesFile(path)
	if err != nil {
		log.Fatalf("loading C library registry: %v", err)
	}
	if loaded && verbose {
		log.Printf("C library registry: %s", path)
	}
	scanConfigPath := filepath.Join(filepath.Dir(path), "scanconfig.json")
	loaded, err = scanner.LoadScanConfigFile(scanConfigPath)
	if err != nil {
		log.Fatalf("loading scan config: %v", err)
	}
	if loaded && verbose {
		log.Printf("scan config: %s", scanConfigPath)
	}
}

// ---------------------------------------------------------------------------
// bindings
// ---------------------------------------------------------------------------

func runBindings(args []string) {
	fs := flag.NewFlagSet("bindings", flag.ExitOnError)
	metaDir := fs.String("metadata-dir", "./metadata", "Directory containing .gometa.json files")
	frameworksOut := fs.String("frameworks-out", defaultRawFrameworksOutDir, "Output directory for purego ObjC framework packages")
	librariesOut := fs.String("libraries-out", defaultRawLibrariesOutDir, "Output directory for CGo C library packages")
	diagnosticsOut := fs.String("diagnostics", "", "Write type-degradation diagnostics to this JSON file (use to create or refresh the baseline)")
	diagnosticsBaseline := fs.String("diagnostics-baseline", "", "Fail when a diagnostic appears that is not in this baseline JSON file")
	clibraries := fs.String("clibraries", "./metadata/clibraries.json", "C library registry JSON (falls back to built-in defaults when absent)")
	verbose := fs.Bool("v", false, "Verbose output")
	_ = fs.Parse(args)

	// The registry's per-library "backend" key decides which pipeline emits
	// each C library, so emission needs it loaded, not just scans.
	loadCLibraryRegistry(*clibraries, *verbose)

	pureReg := loadPureRegistry(*metaDir, defaultFrameworksModulePrefix, defaultLibrariesModulePrefix)
	cgoReg := loadCGORegistry(*metaDir, defaultLibrariesModulePrefix)

	var collected []string

	// Phase A: purego ObjC frameworks.
	log.Printf("emitting purego ObjC frameworks → %s", *frameworksOut)
	if err := purepipeline.GenerateBindings(purepipeline.BindingsConfig{
		Registry:         pureReg,
		FrameworksOutDir: *frameworksOut,
		LibrariesOutDir:  "", // libraries are emitted by the CGo pipeline below
		Verbose:          *verbose,
		DiagnosticsSink:  &collected,
	}); err != nil {
		log.Fatalf("bindings (purego frameworks): %v", err)
	}

	// Phase B: C libraries. The CGo pipeline is the single authority for the
	// library Go surface; a clibraries.json "backend": "purego" entry swaps
	// only the emitted bodies (dlopen/RegisterLibFunc instead of the bridge).
	log.Printf("emitting C libraries → %s", *librariesOut)
	if err := cgopipeline.GenerateBindings(cgopipeline.BindingsConfig{
		Registry:        cgoReg,
		LibrariesOutDir: *librariesOut,
		Verbose:         *verbose,
		DiagnosticsSink: &collected,
	}); err != nil {
		log.Fatalf("bindings (CGo libraries): %v", err)
	}

	reportDiagnostics(collected, *diagnosticsOut, *diagnosticsBaseline)

	log.Printf("done: %d framework(s)", len(pureReg.Frameworks))
}

// reportDiagnostics writes and/or baseline-checks the collected type-degradation
// diagnostics. A baseline violation (new degradations) is fatal — this is the
// CI ratchet that prevents generator changes from silently degrading types.
func reportDiagnostics(collected []string, outPath, baselinePath string) {
	unique := diagnostics.Normalise(collected)
	if outPath != "" {
		if err := diagnostics.Write(outPath, unique); err != nil {
			log.Fatalf("bindings: %v", err)
		}
		log.Printf("wrote %d diagnostic(s) → %s", len(unique), outPath)
	}
	if baselinePath == "" {
		return
	}
	newEntries, fixedEntries, err := diagnostics.CheckBaseline(baselinePath, unique)
	if err != nil {
		log.Fatalf("bindings: %v", err)
	}
	if len(fixedEntries) > 0 {
		log.Printf("%d baseline diagnostic(s) no longer occur — shrink the baseline with --diagnostics %s:", len(fixedEntries), baselinePath)
		for _, entry := range fixedEntries {
			log.Printf("  fixed: %s", entry)
		}
	}
	if len(newEntries) > 0 {
		log.Printf("%d NEW type degradation(s) not in baseline %s:", len(newEntries), baselinePath)
		for _, entry := range newEntries {
			log.Printf("  new: %s", entry)
		}
		log.Fatalf("bindings: %d new type degradation(s) — fix them or update the baseline deliberately (--diagnostics %s)", len(newEntries), baselinePath)
	}
	log.Printf("diagnostics baseline OK (%d known, 0 new)", len(unique))
}

// ---------------------------------------------------------------------------
// all
// ---------------------------------------------------------------------------

func runAll(args []string) {
	fs := flag.NewFlagSet("all", flag.ExitOnError)
	framework := fs.String("framework", "", `Framework(s) to scan before emitting (name, comma-separated, or "all"; omit to skip scan)`)
	sdkPath := fs.String("sdk", "", "macOS SDK path (default: auto-detected)")
	sdkVersion := fs.String("sdk-version", "", "SDK version string (default: auto-detected)")
	parallel := fs.Int("parallel", runtime.NumCPU(), "Number of frameworks to scan concurrently")
	metaDir := fs.String("metadata-dir", "./metadata", "Directory for .gometa.json files")
	arch := fs.String("arch", "arm64", "Target architecture")
	frameworksOut := fs.String("frameworks-out", defaultRawFrameworksOutDir, "Output directory for purego ObjC framework packages")
	librariesOut := fs.String("libraries-out", defaultRawLibrariesOutDir, "Output directory for CGo C library packages")
	clibraries := fs.String("clibraries", "./metadata/clibraries.json", "C library registry JSON (falls back to built-in defaults when absent)")
	verbose := fs.Bool("v", false, "Verbose output")
	_ = fs.Parse(args)

	loadCLibraryRegistry(*clibraries, *verbose)
	sdk := detectSDK(*sdkPath, *verbose)
	_ = arch

	if *framework != "" {
		ver := detectSDKVersion(*sdkVersion)
		doScan(*framework, sdk, ver, *metaDir, *arch, *parallel, *verbose)
	}

	pureReg := loadPureRegistry(*metaDir, defaultFrameworksModulePrefix, defaultLibrariesModulePrefix)
	cgoReg := loadCGORegistry(*metaDir, defaultLibrariesModulePrefix)

	log.Printf("emitting purego ObjC frameworks → %s", *frameworksOut)
	if err := purepipeline.GenerateBindings(purepipeline.BindingsConfig{
		Registry:         pureReg,
		FrameworksOutDir: *frameworksOut,
		LibrariesOutDir:  "",
		Verbose:          *verbose,
	}); err != nil {
		log.Fatalf("bindings (purego frameworks): %v", err)
	}

	log.Printf("emitting C libraries → %s", *librariesOut)
	if err := cgopipeline.GenerateBindings(cgopipeline.BindingsConfig{
		Registry:        cgoReg,
		LibrariesOutDir: *librariesOut,
		Verbose:         *verbose,
	}); err != nil {
		log.Fatalf("bindings (CGo libraries): %v", err)
	}

	log.Printf("done: %d framework(s)", len(pureReg.Frameworks))
}

// ---------------------------------------------------------------------------
// class-hierarchy
// ---------------------------------------------------------------------------

func runClassHierarchy(args []string) {
	fs := flag.NewFlagSet("class-hierarchy", flag.ExitOnError)
	metaDir := fs.String("metadata-dir", "./metadata", "Directory containing .gometa.json files")
	outPath := fs.String("out", "./metadata/objcclasshierarchy/objc_class_hierarchy_generated.go", "Output path for the ObjC class hierarchy Go file")
	arch := fs.String("arch", "arm64", "Target architecture")
	_ = fs.Parse(args)

	var all []string
	globs := []string{
		filepath.Join(*metaDir, "frameworks", "*", "*.gometa.json"),
		filepath.Join(*metaDir, "frameworks", "*", "*", "*.gometa.json"),
		filepath.Join(*metaDir, "libraries", "*", "*.gometa.json"),
	}
	for _, pattern := range globs {
		m, err := filepath.Glob(pattern)
		if err != nil {
			log.Fatalf("searching metadata dir %s: %v", *metaDir, err)
		}
		all = append(all, m...)
	}
	if len(all) == 0 {
		log.Fatalf("no metadata found in %s — run 'generate scan' first", *metaDir)
	}
	paths := pickArch(all, splitTrimmed(*arch, ",")[0])
	log.Printf("deriving ObjC class hierarchy from %d framework(s)", len(paths))
	if err := cgopipeline.EmitObjCClassHierarchy(paths, *outPath); err != nil {
		log.Fatalf("class-hierarchy: %v", err)
	}
	log.Printf("wrote %s", *outPath)
}

// ---------------------------------------------------------------------------
// list
// ---------------------------------------------------------------------------

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	sdkPath := fs.String("sdk", "", "macOS SDK path (default: auto-detected)")
	verbose := fs.Bool("v", false, "Verbose output")
	_ = fs.Parse(args)

	sdk := detectSDK(*sdkPath, *verbose)
	frameworks, err := scanner.ListFrameworks(sdk)
	if err != nil {
		log.Fatalf("listing frameworks: %v", err)
	}
	for _, f := range frameworks {
		fmt.Println(f)
	}
}

// ---------------------------------------------------------------------------
// diff
// ---------------------------------------------------------------------------

// runDiff compares two metadata trees and prints a semantic API change report.
// To diff against a git revision, check the old tree out first, e.g.:
//
//	git worktree add /tmp/old <ref>
//	go run ./cmd/generate/ diff --old /tmp/old/metadata
func runDiff(args []string) {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	oldDir := fs.String("old", "", "Old metadata directory (required)")
	newDir := fs.String("new", "./metadata", "New metadata directory")
	jsonOut := fs.Bool("json", false, "Emit the report as JSON instead of markdown")
	_ = fs.Parse(args)

	if *oldDir == "" {
		fmt.Fprintln(os.Stderr, "diff: --old is required")
		fs.Usage()
		os.Exit(1)
	}

	report := metadiff.Compare(readMetaTree(*oldDir), readMetaTree(*newDir))
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			log.Fatalf("diff: %v", err)
		}
		return
	}
	report.WriteMarkdown(os.Stdout)
}

// readMetaTree loads every .gometa.json under dir (arm64 preferred).
func readMetaTree(dir string) []*macosplatformmetadata.FrameworkMeta {
	var frameworks []*macosplatformmetadata.FrameworkMeta
	for _, p := range collectMetaPaths(dir) {
		framework, err := macosplatformmetadata.Read(p)
		if err != nil {
			log.Fatalf("loading metadata: %v", err)
		}
		frameworks = append(frameworks, framework)
	}
	return frameworks
}

// ---------------------------------------------------------------------------
// validate
// ---------------------------------------------------------------------------

func runValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	metaDir := fs.String("metadata-dir", "./metadata", "Directory containing .gometa.json files")
	_ = fs.Parse(args)

	frameworks := readMetaTree(*metaDir)
	findings := validate.Frameworks(frameworks)
	errors, warnings := 0, 0
	for _, finding := range findings {
		fmt.Fprintln(os.Stderr, finding)
		if finding.Severity == validate.SeverityError {
			errors++
		} else {
			warnings++
		}
	}
	log.Printf("validate: %d framework(s), %d error(s), %d warning(s)", len(frameworks), errors, warnings)
	if validate.HasErrors(findings) {
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// shared helpers
// ---------------------------------------------------------------------------

func detectSDK(sdkPath string, verbose bool) string {
	if sdkPath != "" {
		return sdkPath
	}
	p, err := scanner.SDKPath()
	if err != nil {
		log.Fatalf("auto-detecting SDK: %v", err)
	}
	if verbose {
		log.Printf("SDK: %s", p)
	}
	return p
}

func detectSDKVersion(ver string) string {
	if ver != "" {
		return ver
	}
	v, err := scanner.SDKVersion()
	if err != nil {
		log.Fatalf("auto-detecting SDK version: %v", err)
	}
	return v
}

func doScan(framework, sdk, sdkVersion, metaDir, arch string, parallel int, verbose bool) {
	frameworks := resolveFrameworks(framework, sdk)
	archList := splitTrimmed(arch, ",")
	total := len(frameworks)

	// Toolchain provenance, recorded in every scanned metadata file. Failure is
	// non-fatal: a missing xcodebuild (Command Line Tools-only hosts) degrades
	// to empty provenance rather than blocking the scan.
	clangVersion, err := scanner.ClangVersion()
	if err != nil {
		log.Printf("warning: could not detect clang version: %v", err)
	}
	xcodeVersion, err := scanner.XcodeVersion()
	if err != nil {
		log.Printf("warning: could not detect Xcode version: %v", err)
	}
	if verbose {
		log.Printf("toolchain: %s | %s", clangVersion, xcodeVersion)
	}

	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		log.Fatalf("creating metadata dir %s: %v", metaDir, err)
	}

	workers := max(parallel, 1)
	sem := make(chan struct{}, workers)

	var (
		mu        sync.Mutex
		failedFWs []string
		wg        sync.WaitGroup
	)

	log.Printf("scanning %d framework(s) with %d parallel worker(s)", total, workers)

	for i, fw := range frameworks {
		for _, a := range archList {
			wg.Add(1)
			sem <- struct{}{}
			go func(idx int, fw, arch string) {
				defer wg.Done()
				defer func() { <-sem }()

				if total > 1 || verbose {
					log.Printf("[%d/%d] scanning %s (%s, macOS %s)", idx+1, total, fw, arch, sdkVersion)
				}

				root, err := scanner.DumpAST(sdk, fw, arch)
				if err != nil {
					log.Printf("  SKIP %s: %v", fw, err)
					mu.Lock()
					failedFWs = append(failedFWs, fw)
					mu.Unlock()
					return
				}

				// Authoritative C record layouts (offsets + sizes) from a second
				// clang pass; nil on failure so extraction degrades to computed
				// layout rather than skipping the framework.
				layouts, err := scanner.DumpRecordLayouts(sdk, fw, arch)
				if err != nil {
					log.Printf("  %s: record-layout dump failed (%v); using computed layout", fw, err)
					layouts = nil
				}

				framework := scanner.Extract(root, sdk, fw, sdkVersion, arch, layouts)
				framework.ClangVersion = clangVersion
				framework.XcodeVersion = xcodeVersion

				metaSubdir := "frameworks"
				if scanner.IsCLibraryName(fw) {
					metaSubdir = "libraries"
				}
				fwMetaDir := filepath.Join(metaDir, metaSubdir, strings.ToLower(fw))
				if err := macosplatformmetadata.Write(framework, fwMetaDir); err != nil {
					log.Fatalf("writing metadata for %s: %v", fw, err)
				}

				if verbose || total > 1 {
					log.Printf("  %s: classes=%d protocols=%d enums=%d structs=%d functions=%d externs=%d",
						fw, len(framework.Classes), len(framework.Protocols), len(framework.Enums),
						len(framework.Structs), len(framework.Functions), len(framework.Externs))
				}
			}(i, fw, a)
		}
	}
	wg.Wait()

	succeeded := total - len(failedFWs)
	if total > 1 {
		log.Printf("scanned %d/%d framework(s)", succeeded, total)
	}
	if len(failedFWs) > 0 {
		log.Printf("FAILED %d framework(s): %s", len(failedFWs), strings.Join(failedFWs, ", "))
		os.Exit(1)
	}
}

// loadPureRegistry loads metadata into a purego Registry for ObjC framework emission.
func loadPureRegistry(metaDir, frameworksModulePrefix, librariesModulePrefix string) *purepipeline.Registry {
	paths := collectMetaPaths(metaDir)
	log.Printf("loading %d cached framework(s)", len(paths))
	reg, err := purepipeline.LoadAll(paths, frameworksModulePrefix, librariesModulePrefix)
	if err != nil {
		log.Fatalf("loading metadata (purego): %v", err)
	}
	return reg
}

// loadCGORegistry loads metadata into a CGo Registry for C library emission.
// Only metadata/libraries/ is loaded: C library packages are self-contained
// CGo code and must not reference types from the purego ObjC framework
// packages (incompatible pointer representations). With framework metadata
// absent, ObjC class references degrade to unsafe.Pointer and shared C types
// (e.g. os_log_type_t) resolve to the owning library itself.
func loadCGORegistry(metaDir, modulePrefix string) *cgopipeline.Registry {
	paths := globMetaPaths(metaDir, []string{
		filepath.Join(metaDir, "libraries", "*", "*.gometa.json"),
	})
	reg, err := cgopipeline.LoadAll(paths, modulePrefix)
	if err != nil {
		log.Fatalf("loading metadata (cgo): %v", err)
	}
	return reg
}

// collectMetaPaths globs all .gometa.json files in metaDir and picks arm64.
func collectMetaPaths(metaDir string) []string {
	return globMetaPaths(metaDir, []string{
		filepath.Join(metaDir, "frameworks", "*", "*.gometa.json"),
		filepath.Join(metaDir, "frameworks", "*", "*", "*.gometa.json"),
		filepath.Join(metaDir, "libraries", "*", "*.gometa.json"),
	})
}

// globMetaPaths expands the given glob patterns and picks arm64 metadata.
func globMetaPaths(metaDir string, globs []string) []string {
	var all []string
	for _, pattern := range globs {
		m, err := filepath.Glob(pattern)
		if err != nil {
			log.Fatalf("searching metadata dir %s: %v", metaDir, err)
		}
		all = append(all, m...)
	}
	if len(all) == 0 {
		log.Fatalf("no metadata found in %s — run 'generate scan' first", metaDir)
	}
	return pickArch(all, "arm64")
}

func resolveFrameworks(spec, sdkPath string) []string {
	if strings.EqualFold(spec, "all") {
		fws, err := scanner.ListFrameworks(sdkPath)
		if err != nil {
			log.Fatalf("listing frameworks: %v", err)
		}
		clibs, err := scanner.ListCLibraries(sdkPath)
		if err != nil {
			log.Fatalf("listing C libraries: %v", err)
		}
		fws = append(fws, clibs...)
		sort.Strings(fws)
		return fws
	}
	return splitTrimmed(spec, ",")
}

func splitTrimmed(s, sep string) []string {
	var out []string
	for part := range strings.SplitSeq(s, sep) {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func pickArch(files []string, arch string) []string {
	byDir := map[string][]string{}
	for _, f := range files {
		dir := filepath.Dir(f)
		byDir[dir] = append(byDir[dir], f)
	}
	var out []string
	for _, group := range byDir {
		chosen := group[0]
		for _, f := range group {
			b := filepath.Base(f)
			if strings.Contains(b, "-"+arch+"-") || strings.HasSuffix(b, "-"+arch+".gometa.json") {
				chosen = f
				break
			}
		}
		out = append(out, chosen)
	}
	sort.Strings(out)
	return out
}
