package pipeline

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/libraries"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// IdiomaticConfig configures GenerateIdiomaticLibraries.
type IdiomaticConfig struct {
	Registry  *Registry
	OutDir    string   // output root, e.g. ./opinionated/idiomatic/libraries
	Libraries []string // optional case-insensitive filter; empty = all libraries
	Verbose   bool

	// RawSourceRoot is the directory holding the raw C-library packages the
	// idiomatic layer wraps and re-exports (one <name>/ subdirectory each).
	// Defaults to ./bindings/libraries when empty. The alias emitter reads each
	// raw package's emitted source from here to re-export its exported symbols.
	RawSourceRoot string
}

// GenerateIdiomaticLibraries emits the opinionated idiomatic wrapper layer for
// every CGo C-library package (framework.LinkLib != "") into OutDir/<name>/.
// It mirrors the frameworks idiomatic pipeline but targets the CGo libraries,
// wrapping the already-clean raw bindings with handle-method/free-function
// wrappers and reusing the spec/slice/async scaffolding for any class-bearing
// idiolib.
func GenerateIdiomaticLibraries(cfg IdiomaticConfig) error {
	reg := cfg.Registry
	m := buildMapper(reg, false)

	filter := make(map[string]bool, len(cfg.Libraries))
	for _, name := range cfg.Libraries {
		filter[strings.ToLower(name)] = true
	}

	rawSrcRoot := cfg.RawSourceRoot
	if rawSrcRoot == "" {
		rawSrcRoot = filepath.Join("bindings", "libraries")
	}

	// On a full regen (no library filter) drop every library package directory
	// under OutDir except the public bsd support package, so raw packages this
	// public tree replaced during the switch — and packages for libraries that
	// left the metadata — do not linger.
	if len(filter) == 0 {
		if entries, err := os.ReadDir(cfg.OutDir); err == nil {
			for _, ent := range entries {
				if ent.IsDir() && ent.Name() != "bsd" {
					if err := os.RemoveAll(filepath.Join(cfg.OutDir, ent.Name())); err != nil {
						return fmt.Errorf("clean idiomatic library dir %s: %w", ent.Name(), err)
					}
				}
			}
		}
	}

	for _, framework := range reg.Frameworks {
		if framework.LinkLib == "" || framework.IsSwiftOnly {
			continue // C libraries only
		}
		if len(filter) > 0 && !filter[strings.ToLower(framework.Framework)] {
			continue
		}
		pkgName := strings.ToLower(framework.Framework)
		outDir := filepath.Join(cfg.OutDir, pkgName)
		rawImportPath := reg.ModulePrefix + "/" + pkgName
		rawSrcDir := filepath.Join(rawSrcRoot, pkgName)

		if err := emitIdiomaticLibrary(outDir, pkgName, rawImportPath, rawSrcDir, framework, m); err != nil {
			return fmt.Errorf("idiomatic library %s: %w", framework.Framework, err)
		}
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "idiomatic library: emitted %s → %s\n", framework.Framework, outDir)
		}
	}
	return nil
}

// emitIdiomaticLibrary runs every library emitter, writing only the non-empty
// outputs (gofmt-formatted) plus a doc.go. Stale *_generated.go files are removed
// first so a regeneration never leaves orphans behind.
func emitIdiomaticLibrary(outDir, pkgName, rawImportPath, rawSrcDir string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper) error {
	knownClasses := make(map[string]bool, len(framework.Classes))
	for name := range framework.Classes {
		knownClasses[name] = true
	}

	// The ergonomic emitters run first; the alias re-export runs last so it can
	// skip any name they already declare (a handle wrapper the cfunctions emitter
	// builds from a raw struct type would otherwise redeclare that type).
	emitters := []struct {
		suffix string
		fn     func(*bytes.Buffer) error
	}{
		{"cfunctions", func(b *bytes.Buffer) error {
			return idiolib.EmitCFunctions(b, pkgName, rawImportPath, framework, m, knownClasses)
		}},
		{"specs", func(b *bytes.Buffer) error {
			return idiolib.EmitSpecs(b, pkgName, rawImportPath, framework, m, knownClasses)
		}},
		{"slices", func(b *bytes.Buffer) error {
			return idiolib.EmitSlices(b, pkgName, rawImportPath, framework, m, knownClasses)
		}},
		{"async", func(b *bytes.Buffer) error {
			return idiolib.EmitAsync(b, pkgName, rawImportPath, framework, m, knownClasses)
		}},
	}

	files := map[string][]byte{}
	declared := map[string]bool{}
	for _, e := range emitters {
		var b bytes.Buffer
		if err := e.fn(&b); err != nil {
			return fmt.Errorf("%s: %w", e.suffix, err)
		}
		if b.Len() == 0 {
			continue
		}
		formatted, err := format.Source(b.Bytes())
		if err != nil {
			return fmt.Errorf("%s: gofmt: %w", e.suffix, err)
		}
		names, err := idiolib.CollectDeclaredNames(formatted)
		if err != nil {
			return fmt.Errorf("%s: scan declared names: %w", e.suffix, err)
		}
		for n := range names {
			declared[n] = true
		}
		files[pkgName+"_"+e.suffix+"_generated.go"] = formatted
	}

	// Alias re-export of every raw exported symbol the ergonomic emitters did not
	// already provide, so callers reach the library's enums, structs, protocols,
	// and globals through this package.
	var aliasBuf bytes.Buffer
	if err := idiolib.EmitAliases(&aliasBuf, pkgName, rawImportPath, rawSrcDir, declared); err != nil {
		return fmt.Errorf("aliases: %w", err)
	}
	if aliasBuf.Len() > 0 {
		formatted, err := format.Source(aliasBuf.Bytes())
		if err != nil {
			return fmt.Errorf("aliases: gofmt: %w", err)
		}
		files[pkgName+"_aliases_generated.go"] = formatted
	}

	if len(files) == 0 {
		return nil // nothing idiomatic to emit for this library
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if entries, err := os.ReadDir(outDir); err == nil {
		for _, ent := range entries {
			if strings.HasSuffix(ent.Name(), "_generated.go") {
				_ = os.Remove(filepath.Join(outDir, ent.Name()))
			}
		}
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(outDir, name), content, 0o644); err != nil {
			return err
		}
	}
	return writeIdiomaticLibraryDoc(outDir, pkgName, framework.Framework)
}

func writeIdiomaticLibraryDoc(outDir, pkgName, frameworkName string) error {
	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by go-bindings-codegen. DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "//go:build darwin\n\n")
	fmt.Fprintf(&b, "// Package %s provides an idiomatic Go wrapper over the raw %s C-library\n", pkgName, frameworkName)
	fmt.Fprintf(&b, "// bindings: opaque handles become typed wrappers with methods, the C symbol\n")
	fmt.Fprintf(&b, "// prefix is stripped, and OSStatus/kern_return_t results become Go errors.\n")
	fmt.Fprintf(&b, "package %s\n", pkgName)
	return os.WriteFile(filepath.Join(outDir, "doc.go"), b.Bytes(), 0o644)
}
