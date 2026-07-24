// Command gensymbolgate generates the symbol-resolution acceptance gate:
// bindings/acceptance/symbolgate_generated_test.go. For every bound C function
// and extern symbol it emits a dlsym check, and for every ObjC class an
// objc_getClass check, against the real system dylibs — WITHOUT calling anything.
//
// This is Tier A of the acceptance strategy: a generic, exhaustive,
// side-effect-free test of the dynamic-linking contract the codegen's Go
// structure cannot verify. A wrong, renamed, or missing symbol in any binding
// fails here on every PR instead of shipping silently until first call.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	mpm "github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

type entry struct {
	framework string
	dylib     string
	funcs     []string
	externs   []string
	classes   []string
}

func main() {
	out := flag.String("out", "bindings/symbolgate/symbolgate_generated_test.go", "output test file")
	baselinePath := flag.String("baseline", "metadata/symbolgate-baseline.json", "known-unresolved-symbol baseline")
	flag.Parse()

	baseline, err := loadBaseline(*baselinePath)
	if err != nil {
		fatalf("load baseline %s: %v", *baselinePath, err)
	}

	metaDirs := []string{"metadata/frameworks", "metadata/libraries"}
	files, err := collectMetaFiles(metaDirs)
	if err != nil {
		fatal(err)
	}

	var entries []entry
	for _, f := range files {
		fw, err := loadFramework(f)
		if err != nil {
			fatalf("load %s: %v", f, err)
		}
		e := buildEntry(fw)
		if len(e.funcs)+len(e.externs)+len(e.classes) == 0 {
			continue
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].framework < entries[j].framework })

	if err := emit(*out, entries, baseline); err != nil {
		fatal(err)
	}
	fmt.Printf("gensymbolgate: %d frameworks, %d funcs, %d externs, %d classes → %s\n",
		len(entries), count(entries, func(e entry) int { return len(e.funcs) }),
		count(entries, func(e entry) int { return len(e.externs) }),
		count(entries, func(e entry) int { return len(e.classes) }), *out)
}

// buildEntry collects the resolvable symbols the bindings actually reference:
// non-inline available functions (an inline function has no dylib symbol), every
// available extern, and every available ObjC class.
func buildEntry(fw *mpm.FrameworkMeta) entry {
	e := entry{framework: fw.Framework, dylib: dylibPath(fw)}
	seen := func(s []string) []string { sort.Strings(s); return dedup(s) }

	for _, fn := range fw.Functions {
		if fn.Availability.IsUnavailable || fn.IsInline || skip(fn.Name) {
			continue
		}
		e.funcs = append(e.funcs, fn.Name)
	}
	for _, ex := range fw.Externs {
		if ex.Availability.IsUnavailable || skip(ex.Name) {
			continue
		}
		e.externs = append(e.externs, ex.Name)
	}
	for name, cl := range fw.Classes {
		if cl.Availability.IsUnavailable || skip(name) {
			continue
		}
		e.classes = append(e.classes, name)
	}
	e.funcs, e.externs, e.classes = seen(e.funcs), seen(e.externs), seen(e.classes)
	return e
}

func skip(name string) bool {
	return name == "" || strings.Contains(name, "DO_NOT_USE")
}

// dylibPath mirrors the frameworks pipeline's frameworkDylibPath: a C library
// (LinkLib) is /usr/lib/lib<name>.dylib; a framework (or sub-framework, via its
// parent) is /System/Library/Frameworks/<parent>.framework/<parent>.
func dylibPath(fw *mpm.FrameworkMeta) string {
	if fw.LinkLib != "" {
		return "/usr/lib/lib" + fw.LinkLib + ".dylib"
	}
	parent := fw.Framework
	if fw.ParentFramework != "" {
		parent = fw.ParentFramework
	}
	return fmt.Sprintf("/System/Library/Frameworks/%s.framework/%s", parent, parent)
}

// baselineEntry is one known-unresolved symbol tolerated by the gate. kind is
// "function", "extern", "class", or "dlopen".
type baselineEntry struct {
	Framework string `json:"framework"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
}

// loadBaseline reads the known-unresolved-symbol baseline into a lookup keyed
// "kind|framework|name". A missing file is an empty baseline (every failure is a
// hard failure). The baseline captures pre-existing debt — mostly bindings the
// scanner mis-emits for static-inline functions that have no dylib symbol — so a
// PR fails only on a NEW unresolved symbol; fixing the root cause shrinks it.
func loadBaseline(path string) (map[string]bool, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []baselineEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(entries))
	for _, e := range entries {
		out[e.Kind+"|"+e.Framework+"|"+e.Name] = true
	}
	return out, nil
}

func loadFramework(path string) (*mpm.FrameworkMeta, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fw mpm.FrameworkMeta
	if err := json.Unmarshal(b, &fw); err != nil {
		return nil, err
	}
	return &fw, nil
}

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

// selectBestArch picks one metadata file per directory, preferring arm64.
func selectBestArch(files []string) []string {
	byDir := map[string][]string{}
	for _, f := range files {
		byDir[filepath.Dir(f)] = append(byDir[filepath.Dir(f)], f)
	}
	var out []string
	for _, group := range byDir {
		best := group[0]
		for _, f := range group[1:] {
			if strings.Contains(f, "arm64") && !strings.Contains(best, "arm64") {
				best = f
			}
		}
		out = append(out, best)
	}
	return out
}

func dedup(sorted []string) []string {
	out := sorted[:0]
	var prev string
	for i, s := range sorted {
		if i == 0 || s != prev {
			out = append(out, s)
		}
		prev = s
	}
	return out
}

func count(entries []entry, f func(entry) int) int {
	n := 0
	for _, e := range entries {
		n += f(e)
	}
	return n
}

func fatal(err error)                 { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
func fatalf(f string, a ...any)       { fmt.Fprintf(os.Stderr, f+"\n", a...); os.Exit(1) }
