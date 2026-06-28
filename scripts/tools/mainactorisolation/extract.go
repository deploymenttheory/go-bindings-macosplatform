package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/mainactor"
)

// symbolGraph is the subset of a *.symbols.json file we read.
type symbolGraph struct {
	Symbols []symbol `json:"symbols"`
}

type symbol struct {
	Identifier struct {
		Precise string `json:"precise"`
	} `json:"identifier"`
	DeclarationFragments []fragment `json:"declarationFragments"`
}

type fragment struct {
	Kind              string `json:"kind"`
	Spelling          string `json:"spelling"`
	PreciseIdentifier string `json:"preciseIdentifier"`
}

// preciseRe decomposes an Objective-C precise symbol identifier:
//
//	c:objc(cs)NSView                 → kind=cs name=NSView
//	c:objc(cs)NSView(im)addSubview:  → kind=cs name=NSView member=im sel=addSubview:
//	c:objc(cs)NSColor(cm)redColor    → kind=cs name=NSColor member=cm sel=redColor
//	c:objc(cs)NSView(py)wantsLayer   → kind=cs name=NSView member=py sel=wantsLayer
//	c:objc(pl)NSWindowDelegate       → kind=pl name=NSWindowDelegate
var preciseRe = regexp.MustCompile(`^c:objc\((cs|pl)\)([^(]+)(?:\((im|cm|py)\)(.+))?$`)

// harvest runs swift-symbolgraph-extract for one framework and folds the result
// into an Isolation sidecar.
func harvest(framework, sdkPath, target string) (*mainactor.Isolation, string, error) {
	dir, err := os.MkdirTemp("", "mainactor-"+framework+"-")
	if err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(dir)

	cmd := exec.Command("xcrun", "swift-symbolgraph-extract",
		"-module-name", framework,
		"-target", target,
		"-sdk", sdkPath,
		"-output-dir", dir,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, "", fmt.Errorf("swift-symbolgraph-extract failed: %w\n%s", err, out)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.symbols.json"))
	if err != nil {
		return nil, "", err
	}
	if len(files) == 0 {
		return nil, "", fmt.Errorf("no symbol graph produced (module not importable?)")
	}

	iso := &mainactor.Isolation{
		SchemaVersion:        mainactor.CurrentSchemaVersion,
		Framework:            framework,
		MainActorSelectors:   map[string][]string{},
		NonisolatedSelectors: map[string][]string{},
	}
	classSet := map[string]bool{}
	protoSet := map[string]bool{}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, "", err
		}
		var sg symbolGraph
		if err := json.Unmarshal(data, &sg); err != nil {
			return nil, "", fmt.Errorf("parsing %s: %w", filepath.Base(f), err)
		}
		for _, s := range sg.Symbols {
			foldSymbol(s, iso, classSet, protoSet)
		}
	}

	for c := range classSet {
		iso.MainActorClasses = append(iso.MainActorClasses, c)
	}
	for p := range protoSet {
		iso.MainActorProtocols = append(iso.MainActorProtocols, p)
	}
	// Drop main-actor selector entries on classes already wholly isolated — the
	// class-level entry already covers every instance method, so per-selector
	// noise would be redundant.
	for c := range classSet {
		delete(iso.MainActorSelectors, c)
	}
	pruneEmpty(iso.MainActorSelectors)
	pruneEmpty(iso.NonisolatedSelectors)

	stats := fmt.Sprintf("classes=%d protocols=%d extra-selectors=%d nonisolated=%d",
		len(iso.MainActorClasses), len(iso.MainActorProtocols),
		countValues(iso.MainActorSelectors), countValues(iso.NonisolatedSelectors))
	return iso, stats, nil
}

// foldSymbol classifies one symbol's isolation and records it in iso.
func foldSymbol(s symbol, iso *mainactor.Isolation, classSet, protoSet map[string]bool) {
	m := preciseRe.FindStringSubmatch(s.Identifier.Precise)
	if m == nil {
		return // not an Objective-C symbol
	}
	typeKind, typeName, member, sel := m[1], m[2], m[3], m[4]

	isMain := isMainActor(s.DeclarationFragments)
	isNon := isNonisolated(s.DeclarationFragments)
	if !isMain && !isNon {
		return
	}

	if member == "" {
		// A type declaration (class or protocol).
		if !isMain {
			return // a nonisolated type carries no obligation
		}
		switch typeKind {
		case "cs":
			classSet[typeName] = true
		case "pl":
			protoSet[typeName] = true
		}
		return
	}

	// A member. Class (cm) selectors are stored with a leading "+" to match the
	// scanner's IsClassMethod flag; (py) properties are stored by their name.
	key := sel
	if member == "cm" {
		key = "+" + sel
	}
	switch {
	case isNon:
		iso.NonisolatedSelectors[typeName] = appendUnique(iso.NonisolatedSelectors[typeName], key)
	case isMain:
		iso.MainActorSelectors[typeName] = appendUnique(iso.MainActorSelectors[typeName], key)
	}
}

// isMainActor reports whether the fragments carry the @MainActor attribute. The
// symbol graph encodes it as an attribute fragment whose preciseIdentifier is
// the Swift MainActor type (s:ScM); we also accept the spelling as a fallback.
func isMainActor(frags []fragment) bool {
	for _, f := range frags {
		if f.Kind == "attribute" && (f.PreciseIdentifier == "s:ScM" || strings.Contains(f.Spelling, "MainActor")) {
			return true
		}
	}
	return false
}

// isNonisolated reports whether the fragments carry a `nonisolated` modifier,
// the per-member opt-out from a class's main-actor isolation.
func isNonisolated(frags []fragment) bool {
	for _, f := range frags {
		if strings.Contains(f.Spelling, "nonisolated") {
			return true
		}
	}
	return false
}

func appendUnique(xs []string, x string) []string {
	if slices.Contains(xs, x) {
		return xs
	}
	return append(xs, x)
}

func pruneEmpty(m map[string][]string) {
	for k, v := range m {
		if len(v) == 0 {
			delete(m, k)
		}
	}
}

func countValues(m map[string][]string) int {
	n := 0
	for _, v := range m {
		n += len(v)
	}
	return n
}

// --- toolchain helpers ---------------------------------------------------------

func xcrunOutput(args ...string) (string, error) {
	out, err := exec.Command("xcrun", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// defaultTarget derives an arm64 macOS target triple from the active SDK
// version, so the symbol graph covers everything the committed metadata does.
func defaultTarget() (string, error) {
	ver, err := xcrunOutput("--sdk", "macosx", "--show-sdk-version")
	if err != nil {
		return "", fmt.Errorf("reading SDK version: %w", err)
	}
	return "arm64-apple-macosx" + ver, nil
}

func marshalIndent(iso *mainactor.Isolation) ([]byte, error) {
	data, err := json.MarshalIndent(iso, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling sidecar: %w", err)
	}
	return append(data, '\n'), nil
}
