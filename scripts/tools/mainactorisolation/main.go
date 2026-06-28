// Command mainactorisolation harvests Swift @MainActor isolation facts for
// Objective-C frameworks and writes them into per-framework sidecar files
// (mainactor.json) next to the committed .gometa.json metadata.
//
// Why a separate tool (and not the scanner): Apple expresses main-thread
// affinity as Swift's @MainActor. In the headers it is the macro
// NS_SWIFT_UI_ACTOR, which expands to __attribute__((swift_attr("@MainActor"))).
// Clang parses the attribute, but `clang -ast-dump=json` — the form the scanner
// consumes — omits the swift_attr argument string, so the isolation is invisible
// on that path. The Swift symbol graph carries it faithfully (as the s:ScM
// attribute fragment), so this tool drives `swift-symbolgraph-extract` and reads
// the isolation out of the resulting symbol graph.
//
// The committed sidecars let `go run ./cmd/generate/ idiomatic` apply the
// isolation without a Swift toolchain, exactly as appledocs.json works.
//
// Usage:
//
//	go run ./scripts/tools/mainactorisolation fetch --framework AppKit
//	go run ./scripts/tools/mainactorisolation fetch --framework AppKit,WebKit
//	go run ./scripts/tools/mainactorisolation fetch --framework all
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/mainactor"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "fetch" {
		usage()
		os.Exit(2)
	}

	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	frameworks := fs.String("framework", "", "comma-separated framework names, or 'all'")
	metadataDir := fs.String("metadata", "./metadata", "metadata directory holding .gometa.json files")
	target := fs.String("target", "", "Swift target triple (default derived from the active SDK version)")
	_ = fs.Parse(os.Args[2:])

	if *frameworks == "" {
		fmt.Fprintln(os.Stderr, "error: --framework is required (a name, comma list, or 'all')")
		usage()
		os.Exit(2)
	}

	if err := run(*frameworks, *metadataDir, *target); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(frameworkArg, metadataDir, target string) error {
	wanted := parseWanted(frameworkArg)

	sdkPath, err := xcrunOutput("--show-sdk-path")
	if err != nil {
		return fmt.Errorf("locating SDK (is Xcode installed?): %w", err)
	}
	if target == "" {
		target, err = defaultTarget()
		if err != nil {
			return err
		}
	}

	paths, err := discoverFrameworkMetaFiles(metadataDir)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no framework .gometa.json files found under %s", metadataDir)
	}

	matched := 0
	for _, path := range paths {
		framework, err := macosplatformmetadata.Read(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: skipping %s: %v\n", path, err)
			continue
		}
		if framework.IsSwiftOnly {
			continue
		}
		if !wanted["all"] && !wanted[strings.ToLower(framework.Framework)] {
			continue
		}
		matched++

		iso, stats, err := harvest(framework.Framework, sdkPath, target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", framework.Framework, err)
			continue
		}
		out, err := writeSidecar(path, iso)
		if err != nil {
			return err
		}
		fmt.Printf("==> %s (%s)\n    %s\n    wrote %s\n", framework.Framework, path, stats, out)
	}

	if matched == 0 {
		return fmt.Errorf("no frameworks matched %q (under %s)", frameworkArg, metadataDir)
	}
	return nil
}

// writeSidecar writes iso as mainactor.json next to metaPath, with stable key
// ordering so the committed file is diff-friendly.
func writeSidecar(metaPath string, iso *mainactor.Isolation) (string, error) {
	sort.Strings(iso.MainActorClasses)
	sort.Strings(iso.MainActorProtocols)
	sortMapValues(iso.MainActorSelectors)
	sortMapValues(iso.NonisolatedSelectors)

	out := filepath.Join(filepath.Dir(metaPath), mainactor.FileName)
	data, err := marshalIndent(iso)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return "", fmt.Errorf("writing %s: %w", out, err)
	}
	return out, nil
}

func sortMapValues(m map[string][]string) {
	for k := range m {
		sort.Strings(m[k])
	}
}

// discoverFrameworkMetaFiles finds every framework .gometa.json (the libraries
// tree is excluded — C libraries carry no @MainActor), preferring the arm64
// variant when a directory holds several arches.
func discoverFrameworkMetaFiles(metadataDir string) ([]string, error) {
	base := filepath.Join(metadataDir, "frameworks")
	patterns := []string{
		filepath.Join(base, "*", "*.gometa.json"),
		filepath.Join(base, "*", "*", "*.gometa.json"),
	}

	byDir := map[string][]string{}
	for _, pat := range patterns {
		matches, err := filepath.Glob(pat)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			dir := filepath.Dir(m)
			byDir[dir] = append(byDir[dir], m)
		}
	}

	var paths []string
	for _, group := range byDir {
		chosen := group[0]
		for _, f := range group {
			if strings.Contains(filepath.Base(f), "-arm64-") {
				chosen = f
				break
			}
		}
		paths = append(paths, chosen)
	}
	sort.Strings(paths)
	return paths, nil
}

func parseWanted(arg string) map[string]bool {
	wanted := map[string]bool{}
	for name := range strings.SplitSeq(arg, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			wanted[strings.ToLower(name)] = true
		}
	}
	return wanted
}

func usage() {
	fmt.Fprint(os.Stderr, `mainactorisolation — harvest Swift @MainActor isolation into metadata sidecars

Usage:
  go run ./scripts/tools/mainactorisolation fetch --framework <Name|list|all> [flags]

Flags:
  --framework   comma-separated framework names, or 'all' (required)
  --metadata    metadata directory (default ./metadata)
  --target      Swift target triple (default derived from the active SDK)

Examples:
  go run ./scripts/tools/mainactorisolation fetch --framework AppKit
  go run ./scripts/tools/mainactorisolation fetch --framework AppKit,WebKit,MapKit
  go run ./scripts/tools/mainactorisolation fetch --framework all
`)
}
