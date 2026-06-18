// Command appledeveloperdocs harvests Apple's developer documentation from the
// DocC render API and writes it into per-framework sidecar files
// (appledocs.json) next to the committed .gometa.json metadata.
//
// The codegen loaders merge these sidecars into the Doc fields at load time
// (Apple-preferred, header fallback), so the generated raw bindings and the
// idiomatic layer carry Apple's prose alongside the code without touching the
// pure scanned .gometa.json.
//
// Usage:
//
//	go run ./scripts/tools/appledeveloperdocs fetch --framework Foundation
//	go run ./scripts/tools/appledeveloperdocs fetch --framework Foundation,AppKit --deep
//	go run ./scripts/tools/appledeveloperdocs fetch --framework all
//
// This is a mirror, in Go, of csandrew-dev/AppleDocParser. Where that tool drove
// a Selenium browser and scraped rendered HTML for Apple's REST API schema
// pages, this fetches Apple's structured DocC render JSON directly (no browser)
// and targets framework symbols (classes, methods, properties, enums).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
	"github.com/deploymenttheory/go-bindings-macosplatform/scripts/tools/appledeveloperdocs/docc"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "fetch" {
		usage()
		os.Exit(2)
	}

	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	frameworks := fs.String("framework", "", "comma-separated framework names, or 'all'")
	metadataDir := fs.String("metadata", "./metadata", "metadata directory holding .gometa.json files")
	cacheDir := fs.String("cache", "", "HTTP response cache dir (default <metadata>/.appledocs-httpcache)")
	deep := fs.Bool("deep", false, "also harvest each symbol's Discussion, not just the abstract")
	delay := fs.Duration("delay", 50*time.Millisecond, "minimum delay between live HTTP requests")
	_ = fs.Parse(os.Args[2:])

	if *frameworks == "" {
		fmt.Fprintln(os.Stderr, "error: --framework is required (a name, comma list, or 'all')")
		usage()
		os.Exit(2)
	}
	if *cacheDir == "" {
		*cacheDir = filepath.Join(*metadataDir, ".appledocs-httpcache")
	}

	if err := run(*frameworks, *metadataDir, *cacheDir, *deep, *delay); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(frameworkArg, metadataDir, cacheDir string, deep bool, delay time.Duration) error {
	wanted := parseWanted(frameworkArg)

	paths, err := discoverMetaFiles(metadataDir)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no .gometa.json files found under %s", metadataDir)
	}

	h := &harvester{
		client: docc.NewClient(cacheDir, delay),
		deep:   deep,
		logf:   func(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...) },
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

		fmt.Printf("==> %s (%s)\n", framework.Framework, path)
		docs, stats := h.extract(framework)
		out, err := writeSidecar(path, docs)
		if err != nil {
			return err
		}
		fmt.Printf("    %s\n    wrote %s\n", stats, out)
	}

	if matched == 0 {
		return fmt.Errorf("no frameworks matched %q (under %s)", frameworkArg, metadataDir)
	}
	return nil
}

func parseWanted(arg string) map[string]bool {
	wanted := map[string]bool{}
	for _, name := range strings.Split(arg, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			wanted[strings.ToLower(name)] = true
		}
	}
	return wanted
}

func usage() {
	fmt.Fprint(os.Stderr, `appledeveloperdocs — harvest Apple developer docs into metadata sidecars

Usage:
  go run ./scripts/tools/appledeveloperdocs fetch --framework <Name|list|all> [flags]

Flags:
  --framework   comma-separated framework names, or 'all' (required)
  --metadata    metadata directory (default ./metadata)
  --cache       HTTP cache dir (default <metadata>/.appledocs-httpcache)
  --deep        also harvest each symbol's Discussion
  --delay       minimum delay between live HTTP requests (default 50ms)

Examples:
  go run ./scripts/tools/appledeveloperdocs fetch --framework Foundation
  go run ./scripts/tools/appledeveloperdocs fetch --framework Virtualization --deep
  go run ./scripts/tools/appledeveloperdocs fetch --framework all
`)
}
