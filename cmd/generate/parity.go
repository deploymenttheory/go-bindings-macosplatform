//go:build darwin

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	idiolibraries "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/libraries"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emitmanifest"
	purepipeline "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/pipeline"
)

// runParity proves emittance parity between the raw and idiomatic ObjC-framework
// emitters. It generates both trees into throwaway directories with a manifest
// recorder attached to each, then reports every raw construct that has no
// idiomatic counterpart (keyed on the ObjC/C name, so renames are not gaps).
//
// The library side is instrumented separately once its emitters exist (Phase 6);
// this command covers the frameworks pipeline, which is where the enum/struct/
// extern/protocol coverage gaps live.
func runParity(args []string) {
	fs := flag.NewFlagSet("parity", flag.ExitOnError)
	metaDir := fs.String("metadata-dir", "./metadata", "Directory containing .gometa.json files")
	framework := fs.String("framework", "", `Restrict to framework(s): name or comma-separated list (default: all)`)
	kindFilter := fs.String("kind", "", "Restrict the report to one construct kind (e.g. enum, struct)")
	rawOut := fs.String("raw-manifest", "", "Also write the raw manifest to this JSON file")
	idioOut := fs.String("idiomatic-manifest", "", "Also write the idiomatic manifest to this JSON file")
	showRenames := fs.Bool("renames", false, "Also list constructs emitted under a different Go name")
	limit := fs.Int("limit", 25, "Max missing entries to list per group (0 = all)")
	baseline := fs.String("baseline", "", "Ratchet against this baseline of known-missing constructs: fail only on gaps NOT already listed (a regression)")
	writeBaseline := fs.String("write-baseline", "", "Write the current missing set to this baseline file (use after a phase closes gaps)")
	fs.Parse(args)

	var names []string
	if *framework != "" {
		names = splitTrimmed(*framework, ",")
	}

	tmp, err := os.MkdirTemp("", "parity-*")
	if err != nil {
		log.Fatalf("parity: temp dir: %v", err)
	}
	defer os.RemoveAll(tmp)

	rawRec := emitmanifest.New()
	idioRec := emitmanifest.New()

	// Raw frameworks only (LibrariesOutDir empty), so the CGo trampolines and the
	// committed bindings/runtime tree are never touched.
	reg := loadPureRegistry(*metaDir, defaultFrameworksModulePrefix, defaultLibrariesModulePrefix)
	if err := purepipeline.GenerateBindings(purepipeline.BindingsConfig{
		Registry:         reg,
		FrameworksOutDir: filepath.Join(tmp, "raw"),
		LibrariesOutDir:  "",
		Manifest:         rawRec,
	}); err != nil {
		log.Fatalf("parity: raw generation: %v", err)
	}

	idioReg := loadPureRegistry(*metaDir, defaultFrameworksModulePrefix, defaultLibrariesModulePrefix)
	if err := purepipeline.GenerateIdiomatic(purepipeline.IdiomaticConfig{
		Registry:   idioReg,
		OutDir:     filepath.Join(tmp, "idiomatic", "framework"),
		Frameworks: names,
		Manifest:   idioRec,
	}); err != nil {
		log.Fatalf("parity: idiomatic generation: %v", err)
	}

	if *rawOut != "" {
		if err := rawRec.Write(*rawOut); err != nil {
			log.Fatalf("parity: write raw manifest: %v", err)
		}
	}
	if *idioOut != "" {
		if err := idioRec.Write(*idioOut); err != nil {
			log.Fatalf("parity: write idiomatic manifest: %v", err)
		}
	}

	report := emitmanifest.Compare(rawRec.Entries(), idioRec.Entries())
	printParityReport(report, *kindFilter, *limit, *showRenames)

	// Library parity: the CGo library layer re-exports raw types/consts/vars as
	// idiomatic aliases (identical names), so a plain exported-symbol comparison of
	// the committed trees proves coverage without the metadata manifest.
	libMissing := reportLibraryParity(*limit)

	current := missingKeys(report)
	current = append(current, libMissing...)
	sort.Strings(current)

	if *writeBaseline != "" {
		if err := writeParityBaseline(*writeBaseline, current); err != nil {
			log.Fatalf("parity: write baseline: %v", err)
		}
		log.Printf("parity: wrote baseline (%d known-missing) → %s", len(current), *writeBaseline)
		return
	}

	// Ratchet: fail only on constructs missing now but not in the baseline (a
	// regression). Constructs the baseline lists that are now present are
	// improvements — shrink the baseline with --write-baseline.
	if *baseline != "" {
		regressions, fixed, err := checkParityBaseline(*baseline, current)
		if err != nil {
			log.Fatalf("parity: %v", err)
		}
		if len(fixed) > 0 {
			log.Printf("parity: %d baseline gap(s) now closed — shrink the baseline with --write-baseline %s", len(fixed), *baseline)
		}
		if len(regressions) > 0 {
			log.Printf("parity: %d NEW gap(s) not in the baseline (a regression):", len(regressions))
			for i, r := range regressions {
				if i >= 25 {
					log.Printf("    … and %d more", len(regressions)-25)
					break
				}
				log.Printf("    %s", r)
			}
			os.Exit(1)
		}
		log.Printf("parity baseline OK (%d known-missing, 0 new)", len(current))
		return
	}

	// No baseline: strict mode — any gap is a failure.
	if len(report.Missing) > 0 {
		os.Exit(1)
	}
}

// reportLibraryParity compares each committed raw C-library package's
// re-exportable exports (types, consts, vars) against the idiomatic package's
// exported symbols (aliases plus ergonomic declarations), printing and returning
// the raw symbols with no idiomatic counterpart as "library:<pkg>:<name>" keys.
func reportLibraryParity(limit int) []string {
	const rawRoot = "bindings/libraries"
	const idioRoot = "opinionated/idiomatic/libraries"

	entries, err := os.ReadDir(rawRoot)
	if err != nil {
		return nil
	}
	var missing []string
	total, covered := 0, 0
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		pkg := ent.Name()
		rawExports, err := idiolibraries.ReExportableExports(filepath.Join(rawRoot, pkg))
		if err != nil {
			continue
		}
		idioExports, err := idiolibraries.PackageExports(filepath.Join(idioRoot, pkg))
		if err != nil {
			idioExports = map[string]bool{} // idiomatic package absent → all missing
		}
		for name := range rawExports {
			total++
			if idioExports[name] {
				covered++
				continue
			}
			missing = append(missing, "library:"+pkg+":"+name)
		}
	}
	sort.Strings(missing)

	fmt.Printf("\nLibrary parity: %d raw type/const/var exports, %d covered by idiomatic; %d missing.\n",
		total, covered, len(missing))
	shown := missing
	if limit > 0 && len(shown) > limit {
		shown = shown[:limit]
	}
	for _, m := range shown {
		fmt.Printf("    %s\n", m)
	}
	if limit > 0 && len(missing) > limit {
		fmt.Printf("    … and %d more\n", len(missing)-limit)
	}
	return missing
}

// missingKeys renders the report's missing constructs as sorted, compact
// "<kind> <metaKey>" lines for the baseline file. Kind never contains a space,
// so the first space is an unambiguous separator.
func missingKeys(report emitmanifest.Report) []string {
	out := make([]string, 0, len(report.Missing))
	for _, m := range report.Missing {
		out = append(out, m.Kind+" "+m.MetaKey)
	}
	sort.Strings(out)
	return out
}

func writeParityBaseline(path string, keys []string) error {
	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

// checkParityBaseline compares the current missing set against the committed
// baseline. regressions are missing now but absent from the baseline; fixed are
// baseline entries now present (emitted).
func checkParityBaseline(path string, current []string) (regressions, fixed []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read baseline %s: %w", path, err)
	}
	var baseline []string
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, nil, fmt.Errorf("parse baseline %s: %w", path, err)
	}
	baseSet := make(map[string]bool, len(baseline))
	for _, b := range baseline {
		baseSet[b] = true
	}
	curSet := make(map[string]bool, len(current))
	for _, c := range current {
		curSet[c] = true
		if !baseSet[c] {
			regressions = append(regressions, c)
		}
	}
	for _, b := range baseline {
		if !curSet[b] {
			fixed = append(fixed, b)
		}
	}
	sort.Strings(regressions)
	return regressions, fixed, nil
}

// printParityReport writes a grouped summary of the missing constructs to stdout.
// When kindFilter is set, only that kind's groups are shown (the totals still
// reflect every kind).
func printParityReport(report emitmanifest.Report, kindFilter string, limit int, showRenames bool) {
	groups := report.MissingByGroup()
	groupKeys := make([]string, 0, len(groups))
	for k := range groups {
		groupKeys = append(groupKeys, k)
	}
	sort.Strings(groupKeys)

	// Per-kind totals across all frameworks.
	kindTotals := make(map[string]int)
	for _, m := range report.Missing {
		kindTotals[m.Kind]++
	}

	fmt.Printf("Parity: %d raw constructs, %d idiomatic; %d missing from idiomatic.\n",
		report.RawTotal, report.IdiomaticTotal, len(report.Missing))

	if len(kindTotals) > 0 {
		fmt.Println("\nMissing by kind:")
		kinds := make([]string, 0, len(kindTotals))
		for k := range kindTotals {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)
		for _, k := range kinds {
			fmt.Printf("  %-16s %d\n", k, kindTotals[k])
		}
	}

	for _, gk := range groupKeys {
		items := groups[gk]
		if kindFilter != "" && items[0].Kind != kindFilter {
			continue
		}
		fmt.Printf("\n%s — %d missing\n", gk, len(items))
		shown := items
		if limit > 0 && len(shown) > limit {
			shown = shown[:limit]
		}
		for _, m := range shown {
			fmt.Printf("    %s (raw %s)\n", m.MetaKey, m.GoSymbol)
		}
		if limit > 0 && len(items) > limit {
			fmt.Printf("    … and %d more\n", len(items)-limit)
		}
	}

	if showRenames && len(report.Renames) > 0 {
		fmt.Printf("\nRenames (informational, not gaps): %d\n", len(report.Renames))
		shown := report.Renames
		if limit > 0 && len(shown) > limit {
			shown = shown[:limit]
		}
		for _, r := range shown {
			fmt.Printf("    %s: raw %s → idiomatic %s\n", r.MetaKey, r.RawSymbol, r.IdioSymbol)
		}
	}
}
