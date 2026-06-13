package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/overrides"
	"github.com/deploymenttheory/go-bindings-macosplatform/metadata/objcclasshierarchy"
)

// Registry holds combined metadata from multiple frameworks. It is the single
// source of truth for cross-framework type resolution: OwnerIndex tells
// every emitter which Go package owns each ObjC class name, enabling precise
// import statements rather than unsafe.Pointer fallbacks.
type Registry struct {
	Frameworks     []*meta.FrameworkMeta
	ClassNameIndex map[string]bool // all class names across all loaded frameworks
	GenericClasses map[string]bool // classes with ObjC generic params
	// GenericParamIndex maps a generic class name to its ordered list of
	// ObjC generic type parameter names (e.g. "NSArray" → ["ObjectType"],
	// "NSDictionary" → ["KeyType", "ObjectType"]). Used by the bridge emitter
	// to substitute placeholders in block-type casts on foreign-extension
	// methods (categories defined on classes owned by another framework).
	GenericParamIndex map[string][]string
	OwnerIndex        map[string]string // className → framework name (e.g. "NSString" → "Foundation")
	// ProtocolIndex maps protocol name → owning framework (e.g. "NSCopying" → "Foundation").
	// Used by the protocol emitter to resolve cross-framework parent protocol embeds.
	ProtocolIndex map[string]string
	// ProtocolProxyIndex maps protocol name → owning framework for protocols that
	// have generated proxy types. Currently mirrors ProtocolIndex (every protocol
	// gets a proxy). Used by the type mapper to resolve return-position id<Protocol>
	// to *<GoProtoName>Proxy instead of unsafe.Pointer.
	ProtocolProxyIndex map[string]string
	// EnumIndex maps enum type name → owning framework (e.g. "VZVirtualMachineState" → "Virtualization").
	// Used to resolve enum-typed return values that would otherwise fall through to unsafe.Pointer.
	EnumIndex map[string]string
	// EnumGoTypeIndex maps enum type name → underlying Go integer type (e.g. "int64", "uint64").
	// Used by the bridge emitter to produce correct C integer casts for enum-typed arguments.
	EnumGoTypeIndex map[string]string
	// TypedefIndex maps typedef name → target ObjC qualType, merged across all frameworks.
	// e.g. "VZMemorySize" → "NSInteger", "AVMIDINoteDuration" → "double"
	// Used by the typemap as a last-resort fallback before degrading to unsafe.Pointer.
	TypedefIndex map[string]string
	// TypedefOwnerIndex maps typedef name → the framework that defined it
	// (e.g. "CFRunLoopRef" → "CoreFoundation"). Used by cycle detection so that
	// typedef-based cross-framework references (CF opaque pointers like CFArrayRef)
	// contribute import edges, not just class/enum references.
	TypedefOwnerIndex map[string]string
	// StructIndex maps a struct name (CGSize, CGRect, NSOperatingSystemVersion,
	// ether_addr_t, …) to the framework that owns the complete definition.
	// Used by the type mapper to resolve value-type struct references as
	// `<pkg>.<Name>` (or bare `<Name>` when same-framework) instead of falling
	// through to unsafe.Pointer. Population prefers the first metadata entry
	// with non-empty fields — forward-declared/placeholder entries from
	// transitively-included headers do not claim ownership.
	StructIndex map[string]string
	// CFTypeIndex maps framework-specific CF opaque typedef names to
	// their owning framework. These are CF-style reference types (e.g. CFHostRef,
	// CFHTTPMessageRef from CFNetwork) that are defined outside CoreFoundation
	// and therefore not in cfTypedefSet. The mapper routes them to
	// *<pkgAlias>.TypedefName rather than the corefoundation package.
	CFTypeIndex map[string]string
	// ClassIndex maps every class name to its full definition across all loaded frameworks.
	// Used by the class emitter to walk cross-framework superclass chains when building
	// struct embedding hierarchies and value-chain constructors.
	ClassIndex map[string]meta.Class
	// ModulePrefix is the Go module path prefix for framework packages,
	// e.g. "github.com/deploymenttheory/go-bindings-macosplatform/frameworks".
	// Cross-framework import paths are built as ModulePrefix + "/" + pkgName.
	ModulePrefix string
}

// LoadAll reads one or more .gometa.json files and builds a Registry.
// paths may be individual .gometa.json files or directories; directories are
// scanned one level deep for all *.gometa.json files.
// When a directory contains multiple arch variants, arm64 is preferred.
func LoadAll(paths []string, modulePrefix string) (*Registry, error) {
	reg := &Registry{
		ClassNameIndex:     make(map[string]bool),
		GenericClasses:     make(map[string]bool),
		GenericParamIndex:  make(map[string][]string),
		OwnerIndex:         make(map[string]string),
		ProtocolIndex:      make(map[string]string),
		ProtocolProxyIndex: make(map[string]string),
		EnumIndex:          make(map[string]string),
		EnumGoTypeIndex:    make(map[string]string),
		TypedefIndex:       make(map[string]string),
		TypedefOwnerIndex:  make(map[string]string),
		StructIndex:        make(map[string]string),
		CFTypeIndex:        make(map[string]string),
		ClassIndex:         make(map[string]meta.Class),
		ModulePrefix:       modulePrefix,
	}

	resolved, err := resolvePaths(paths)
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("no .gometa.json files found in: %v", paths)
	}

	// Pass 1: load all metadata files without deduplication, recording paths.
	type fmWithPath struct {
		framework *meta.FrameworkMeta
		path      string
	}
	var all []fmWithPath
	for _, p := range resolved {
		framework, err := meta.Read(p)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", p, err)
		}
		// Apply the framework's declarative overrides (metadata/…/overrides.json)
		// so the committed .gometa.json stays pure observed Clang output.
		warnings, err := overrides.ApplyAdjacent(p, framework)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", p, err)
		}
		for _, warning := range warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", warning)
		}
		all = append(all, fmWithPath{framework, p})
	}

	// Infer ParentFramework from path depth for metadata that predate the field.
	// Sub-framework files sit two levels deep: {metaDir}/{parent}/{child}/{file}
	// Top-level files sit one level deep:       {metaDir}/{fw}/{file}
	// We detect sub-frameworks by checking whether the grandparent directory
	// is itself a known framework directory (identified by absolute path).
	fwDirPaths := make(map[string]string) // abs dir path → framework name
	for _, lf := range all {
		dir := filepath.Clean(filepath.Dir(lf.path))
		fwDirPaths[dir] = lf.framework.Framework
	}
	for _, lf := range all {
		if lf.framework.ParentFramework != "" {
			continue // already set by scanner
		}
		dir := filepath.Dir(lf.path)
		grandpar := filepath.Clean(filepath.Dir(dir))
		if parentFW, ok := fwDirPaths[grandpar]; ok && parentFW != lf.framework.Framework {
			lf.framework.ParentFramework = parentFW
		}
	}

	// Deduplicate: when a framework appears as both a top-level entry (no parent)
	// and a sub-framework entry (has parent), the top-level version wins.
	// This handles frameworks like CoreGraphics that exist in both
	// the SDK's top-level Frameworks/ and inside ApplicationServices.framework/Frameworks/.
	byName := make(map[string]fmWithPath)
	for _, lf := range all {
		prev, exists := byName[lf.framework.Framework]
		if !exists {
			byName[lf.framework.Framework] = lf
			continue
		}
		// Prefer top-level (ParentFramework == "") over sub-framework.
		prevIsTop := prev.framework.ParentFramework == ""
		curIsTop := lf.framework.ParentFramework == ""
		if curIsTop && !prevIsTop {
			byName[lf.framework.Framework] = lf
		}
		// If both are top-level or both are sub-framework, keep first (earlier in sorted order).
	}

	var frameworks []*meta.FrameworkMeta
	for _, lf := range all {
		if byName[lf.framework.Framework].path != lf.path {
			continue // deduplicated away
		}
		frameworks = append(frameworks, lf.framework)
		reg.Frameworks = append(reg.Frameworks, lf.framework)
	}

	// Pass 2: determine canonical owner for each class.
	//
	// We use "fewest non-zero methods wins": the framework with the fewest
	// non-ghost methods/properties is the canonical owner. This is the inverse of
	// what might seem intuitive, but it's correct:
	//
	//   • When framework B imports framework A's headers, B's metadata for a class
	//     may include A's original methods PLUS B's own category methods — a superset.
	//     "Most methods wins" would incorrectly pick B.
	//   • The canonical framework (A) has only its own primary methods — the minimum.
	//     "Fewest non-zero wins" picks A. ✓
	//
	// Ghost entries (score == 0) are excluded: a class present only as a forward
	// declaration in a framework's headers should not win ownership over a framework
	// that actually defines it.
	//
	// If all entries for a class are ghosts (score == 0), the one from the
	// alphabetically-first framework wins as a stable fallback.
	type classEntry struct {
		framework string
		score     int
		cls       meta.Class
	}
	// allEntries collects every framework's definition for each class.
	allEntries := make(map[string][]classEntry)
	for _, framework := range frameworks {
		for name, cls := range framework.Classes {
			score := len(cls.Methods) + len(cls.Properties)
			allEntries[name] = append(allEntries[name], classEntry{framework.Framework, score, cls})
		}
	}

	type bestEntry struct {
		framework string
		score     int
		cls       meta.Class
	}
	best := make(map[string]bestEntry)
	for name, entries := range allEntries {
		// Separate non-ghost entries (score > 0) from ghost ones.
		var nonGhost []classEntry
		for _, e := range entries {
			if e.score > 0 {
				nonGhost = append(nonGhost, e)
			}
		}
		candidates := nonGhost
		if len(candidates) == 0 {
			candidates = entries // all ghosts — fall back to the full set
		}
		// Among candidates, pick the one with the fewest methods (most minimal =
		// most canonical). Break ties with alphabetical framework name.
		winner := candidates[0]
		for _, e := range candidates[1:] {
			if e.score < winner.score ||
				(e.score == winner.score && e.framework < winner.framework) {
				winner = e
			}
		}
		best[name] = bestEntry(winner)
	}

	// Pass 3: populate registry from canonical winners.
	for name, entry := range best {
		reg.ClassNameIndex[name] = true
		reg.OwnerIndex[name] = entry.framework
		reg.ClassIndex[name] = entry.cls
		if len(entry.cls.GenericParams) > 0 {
			reg.GenericClasses[name] = true
			reg.GenericParamIndex[name] = entry.cls.GenericParams
		}
	}

	// Pass 4: populate enum registry from deduplicated framework metadata.
	// First-write-wins when the same enum name appears in multiple frameworks.
	// EnumGoTypeIndex is keyed by both the raw ObjC name (e.g. "interface_event_t")
	// AND the exported Go name (e.g. "Interface_event_t") so that enumCType can find
	// C-style lowercase enum names after resolveGoType capitalises them.
	for _, framework := range frameworks {
		for enumName, enum := range framework.Enums {
			if _, already := reg.EnumIndex[enumName]; !already {
				reg.EnumIndex[enumName] = framework.Framework
			}
			if enum.GoType != "" {
				if _, already := reg.EnumGoTypeIndex[enumName]; !already {
					reg.EnumGoTypeIndex[enumName] = enum.GoType
				}
				// Also store under the exported Go identifier so that enumCType can
				// look it up after resolveGoType capitalises the name.
				goName := capitaliseFirst(enumName)
				if _, already := reg.EnumGoTypeIndex[goName]; !already {
					reg.EnumGoTypeIndex[goName] = enum.GoType
				}
			}
		}
	}

	// Pass 5: populate protocol registry. The same protocol name often appears
	// in many frameworks' metadata because Apple's headers forward-declare
	// `@protocol Foo;` for cross-framework references. Forward declarations
	// carry no methods and no parents; the canonical definition has at least
	// one of those. We score each entry by `len(Methods) + len(Implements)`
	// and prefer higher scores. Ties break first on framework-name prefix
	// (MTLBuffer → Metal, NSCopying → Foundation) — a strong convention in
	// Apple's frameworks — then on lexicographic order for determinism.
	for _, framework := range frameworks {
		for protoName, proto := range framework.Protocols {
			existing, has := reg.ProtocolIndex[protoName]
			if !has {
				reg.ProtocolIndex[protoName] = framework.Framework
				continue
			}
			priorProto := lookupProtocol(frameworks, existing, protoName)
			priorScore := 0
			if priorProto != nil {
				priorScore = len(priorProto.Methods) + len(priorProto.InheritedProtocols)
			}
			curScore := len(proto.Methods) + len(proto.InheritedProtocols)
			if curScore > priorScore {
				reg.ProtocolIndex[protoName] = framework.Framework
				continue
			}
			if curScore < priorScore {
				continue
			}
			// Tie-break by prefix match: Apple's protocol names use the
			// owning framework's prefix (MTLBuffer → Metal, NSCopying →
			// Foundation, AVCapture → AVFoundation/CoreMedia/AVKit).
			curPrefixMatch := protocolNameMatchesFramework(protoName, framework.Framework)
			priorPrefixMatch := protocolNameMatchesFramework(protoName, existing)
			if curPrefixMatch && !priorPrefixMatch {
				reg.ProtocolIndex[protoName] = framework.Framework
				continue
			}
			if priorPrefixMatch && !curPrefixMatch {
				continue
			}
			// Last resort: alphabetical framework name.
			if framework.Framework < existing {
				reg.ProtocolIndex[protoName] = framework.Framework
			}
		}
	}

	// Populate ProtocolProxyIndex by scanning all method return types across all
	// frameworks. Only protocols that actually appear as id<Protocol> in a single-
	// protocol return position get a proxy type — NSCopying, NSObject, and the
	// many other parameter-only or constraint-only protocols are excluded.
	for i := range all {
		framework := all[i].framework
		buildProtocolProxyIndex(framework.Classes, reg.ProtocolIndex, reg.ProtocolProxyIndex)
		indexProtocolProxiesFromMethods(
			framework.ForeignExtensions,
			reg.ProtocolIndex,
			reg.ProtocolProxyIndex,
		)
	}

	// Pass 6: merge typedef map. First-write-wins; typedefs are definitional.
	// Also record the owning framework so cycle detection can attribute typedef-
	// based cross-framework references (CF opaque pointers, etc.).
	//
	// CoreFoundation opaque-pointer typedefs (CFStringRef, CFRunLoopRef, …) are
	// transitively #included by virtually every other framework's headers, so
	// first-write-wins on alphabetical framework order would attribute them to
	// e.g. Accelerate. The type mapper hard-routes these to *corefoundation.X,
	// so cycle detection must do the same — pin their owner to CoreFoundation.
	for _, framework := range frameworks {
		for name, target := range framework.Typedefs {
			if _, already := reg.TypedefIndex[name]; !already {
				reg.TypedefIndex[name] = target
				reg.TypedefOwnerIndex[name] = framework.Framework
			}
			if typemap.IsCoreFoundationOpaqueRef(name) {
				reg.TypedefOwnerIndex[name] = "CoreFoundation"
			}
		}
	}

	// Pass 7: populate struct registry. A struct can appear in many frameworks'
	// metadata because Apple's headers re-export each other (CGSize ships in
	// CoreFoundation/CFCGTypes.h but is seen by every framework that includes
	// CoreGraphics). We prefer the entry with non-empty fields; among
	// equally-complete entries we prefer the lexicographically smallest
	// framework name for determinism. This guarantees that consumers always
	// receive a complete definition for cross-framework value-type resolution.
	for _, framework := range frameworks {
		for name, s := range framework.Structs {
			existingOwner, has := reg.StructIndex[name]
			if !has {
				// Only register structs with at least one field. Opaque C types
				// (e.g. _CGLContextObject, ColorSyncProfile) have 0 fields and must
				// not be resolved to typed Go structs — they should remain as
				// unsafe.Pointer in generated code.
				if len(s.Fields) > 0 {
					reg.StructIndex[name] = framework.Framework
				}
				continue
			}
			// Replace placeholder (zero-fields) entries when we find a complete one.
			if len(s.Fields) > 0 {
				existing := reg.Frameworks
				_ = existing
				prior := lookupStruct(frameworks, existingOwner, name)
				if prior == nil || len(prior.Fields) == 0 {
					reg.StructIndex[name] = framework.Framework
					continue
				}
				// Both have fields — keep lexicographically smaller owner for stability.
				if framework.Framework < existingOwner {
					reg.StructIndex[name] = framework.Framework
				}
			}
		}
	}

	// Pass 7b: register typedef aliases for underscore-prefixed struct tags.
	// C idiom: "typedef struct _NSRange NSRange;" stores the struct under "_NSRange" in
	// metadata but exposes the public name "NSRange" via a typedef. Register the clean
	// typedef name → same owner so the mapper finds "NSRange" directly and emits the
	// exported alias (foundation.NSRange) rather than the unexported tag (foundation._NSRange).
	for name, owner := range reg.StructIndex {
		if !strings.HasPrefix(name, "_") {
			continue
		}
		target := "struct " + name
		for tName, tTarget := range reg.TypedefIndex {
			if tTarget == target && !strings.HasPrefix(tName, "_") {
				if _, already := reg.StructIndex[tName]; !already {
					reg.StructIndex[tName] = owner
				}
			}
		}
	}

	// Pass 7c: register framework-specific opaque pointer typedef types.
	// Pattern: any typedef XxxRef → "struct Opaque* *" (or "const struct * *") where
	// the target struct has 0 fields. These types are emitted as CF-style wrapper types
	// by the struct emitter but are NOT in cfTypedefSet (which only covers CoreFoundation).
	// The mapper needs to route them to the correct framework package.
	// Covers: CGColorRef (CoreGraphics), CMSampleBufferRef (CoreMedia),
	//         AudioQueueRef (AudioToolbox), SecKeyRef (Security), etc.
	for _, framework := range frameworks {
		// Build set of all zero-field structs in this framework.
		zeroFieldStructs := make(map[string]bool)
		for name, s := range framework.Structs {
			if len(s.Fields) == 0 {
				zeroFieldStructs[name] = true
			}
		}
		for tName, target := range framework.Typedefs {
			if typemap.IsCoreFoundationOpaqueRef(tName) {
				continue // already in cfTypedefSet, handled by corefoundation path
			}
			t := strings.TrimSpace(target)
			t = strings.TrimPrefix(t, "const ")
			t = strings.TrimSpace(t)
			if !strings.HasPrefix(t, "struct ") {
				continue
			}
			// Extract the struct name from "struct Foo *" or "struct Foo*".
			bare := strings.TrimSpace(t[7:]) // strip "struct "
			bare = strings.TrimSuffix(bare, "*")
			bare = strings.TrimSpace(bare)
			if bare == "" || strings.ContainsAny(bare, " *<>^()") {
				continue
			}
			if zeroFieldStructs[bare] {
				if _, already := reg.CFTypeIndex[tName]; !already {
					reg.CFTypeIndex[tName] = framework.Framework
				}
			}
		}
	}

	// Validation: warn when a sub-framework was loaded without its parent framework.
	// Missing parents mean cross-framework references to parent-owned classes silently
	// degrade to unsafe.Pointer.
	loadedFWs := make(map[string]bool, len(reg.Frameworks))
	for _, framework := range reg.Frameworks {
		loadedFWs[strings.ToLower(framework.Framework)] = true
	}
	for _, framework := range reg.Frameworks {
		if framework.ParentFramework != "" &&
			!loadedFWs[strings.ToLower(framework.ParentFramework)] {
			fmt.Fprintf(
				os.Stderr,
				"warning: sub-framework %q loaded without parent %q — cross-framework references may degrade to unsafe.Pointer\n",
				framework.Framework,
				framework.ParentFramework,
			)
		}
	}

	// Validation: detect circular superclass chains in the combined class graph.
	if err := validateSuperchainAcyclic(reg.ClassIndex); err != nil {
		return nil, err
	}

	// Filter each class to novel (non-inherited) methods only. This eliminates
	// bridge-function duplication: inherited methods are handled purely by Go
	// struct embedding, not by per-subclass C bridge functions.
	filterInheritedClassMethods(reg)

	return reg, nil
}

// filterInheritedClassMethods strips inherited instance methods from every ObjC
// class in the registry, leaving only methods first declared on that class.
// After filtering, downstream emitters (bridge, classes, interfaces) see only
// novel methods — no per-emitter inheritance deduplication loops are needed.
//
// Each class retains:
//   - Init methods (covariant return types require a declaration per subclass)
//   - Class methods (not promoted via Go struct embedding; each class owns its own)
//   - Instance methods whose selector is not bridged by any ancestor class
func filterInheritedClassMethods(reg *Registry) {
	// Build reverse map: ObjC class name → the FrameworkMeta pointer that owns it.
	ownerFramework := make(map[string]*meta.FrameworkMeta, len(reg.OwnerIndex))
	for _, framework := range reg.Frameworks {
		for className := range framework.Classes {
			if reg.OwnerIndex[className] == framework.Framework {
				ownerFramework[className] = framework
			}
		}
	}

	for className := range reg.ClassIndex {
		// Walk the full ancestor chain via ObjCClassSuperclassIndex and collect
		// every selector that an ancestor class bridges. Walking to the root
		// guarantees correctness regardless of the order classes are processed:
		// even if an intermediate ancestor has already been filtered, root-class
		// methods remain visible in the ClassIndex.
		ancestorSelectors := make(map[string]bool)
		for super := objcclasshierarchy.ObjCClassSuperclassIndex[className]; super != ""; super = objcclasshierarchy.ObjCClassSuperclassIndex[super] {
			parentClass, ok := reg.ClassIndex[super]
			if !ok {
				break
			}
			for _, method := range parentClass.Methods {
				if classMethodIsBridgeable(method) {
					ancestorSelectors[method.Selector] = true
				}
			}
		}

		if len(ancestorSelectors) == 0 {
			continue // root class — nothing to strip
		}

		class := reg.ClassIndex[className]
		novel := make([]meta.Method, 0, len(class.Methods))
		for _, method := range class.Methods {
			// Always keep init methods (covariant return type per subclass)
			// and class methods (not promoted via Go struct embedding).
			if method.IsInit || method.IsClassMethod {
				novel = append(novel, method)
				continue
			}
			if !ancestorSelectors[method.Selector] {
				novel = append(novel, method)
			}
		}

		if len(novel) == len(class.Methods) {
			continue // nothing changed
		}

		class.Methods = novel
		reg.ClassIndex[className] = class
		if framework, ok := ownerFramework[className]; ok {
			framework.Classes[className] = class
		}
	}
}

// classMethodIsBridgeable reports whether a parent ObjC method bridges its
// selector to Go. Methods the parent cannot bridge (non-format variadic,
// unavailable, or ObjC runtime lifecycle) do not claim their selector —
// a child redeclaring the same selector with a bridgeable signature is novel.
func classMethodIsBridgeable(method meta.Method) bool {
	if method.Availability.IsUnavailable {
		return false
	}
	if method.IsVariadic && !isSelectorFormatVariadic(method.Selector) {
		return false
	}
	if method.IsClassMethod && (method.Selector == "initialize" || method.Selector == "load") {
		return false
	}
	return true
}

// isSelectorFormatVariadic returns true when any keyword in the ObjC selector
// contains "format" — these variadics are bridgeable via a pre-formatted string.
func isSelectorFormatVariadic(selector string) bool {
	for part := range strings.SplitSeq(selector, ":") {
		if strings.Contains(strings.ToLower(part), "format") {
			return true
		}
	}
	return false
}

// validateSuperchainAcyclic walks every superclass chain in allClasses and
// returns an error if a cycle is detected (e.g. A.Super == B, B.Super == A).
// In valid ObjC metadata this never occurs, but malformed .gometa.json files
// could produce it and would cause infinite loops in the emitter.
func validateSuperchainAcyclic(allClasses map[string]meta.Class) error {
	visited := make(map[string]bool, len(allClasses))
	inStack := make(map[string]bool)

	var dfs func(name string) error
	dfs = func(name string) error {
		if inStack[name] {
			return fmt.Errorf(
				"superclass cycle detected at %q — check metadata for circular inheritance",
				name,
			)
		}
		if visited[name] {
			return nil
		}
		visited[name] = true
		inStack[name] = true
		if cls, ok := allClasses[name]; ok && cls.Super != "" {
			if err := dfs(cls.Super); err != nil {
				return err
			}
		}
		inStack[name] = false
		return nil
	}

	// Iterate in sorted order for deterministic error messages.
	names := make([]string, 0, len(allClasses))
	for name := range allClasses {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := dfs(name); err != nil {
			return err
		}
	}
	return nil
}

// resolvePaths expands directory entries into .gometa.json file lists,
// preferring arm64 when multiple arch variants exist per subdirectory.
func resolvePaths(paths []string) ([]string, error) {
	var out []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", p, err)
		}
		if info.IsDir() {
			// Support both flat dirs (*.gometa.json) and metadata/<fw>/*.gometa.json
			matches, err := filepath.Glob(filepath.Join(p, "*.gometa.json"))
			if err != nil {
				return nil, err
			}
			sub, err := filepath.Glob(filepath.Join(p, "*", "*.gometa.json"))
			if err != nil {
				return nil, err
			}
			all := append(matches, sub...)
			out = append(out, selectBestArch(all)...)
		} else {
			if !strings.HasSuffix(p, ".gometa.json") {
				return nil, fmt.Errorf("%s is not a .gometa.json file", p)
			}
			out = append(out, p)
		}
	}
	return out, nil
}

// selectBestArch groups files by directory and picks one per directory,
// preferring arm64 files over others.
func selectBestArch(files []string) []string {
	byDir := map[string][]string{}
	for _, f := range files {
		dir := filepath.Dir(f)
		byDir[dir] = append(byDir[dir], f)
	}
	var out []string
	for _, group := range byDir {
		chosen := group[0]
		for _, f := range group {
			if strings.Contains(filepath.Base(f), "-arm64-") {
				chosen = f
				break
			}
		}
		out = append(out, chosen)
	}
	return out
}

// lookupStruct returns the struct metadata for `name` from the framework matching
// `fwName`, or nil when absent. Used during ownership resolution to compare a
// candidate entry's completeness against the prior winner without re-scanning.
func lookupStruct(frameworks []*meta.FrameworkMeta, fwName, name string) *meta.Struct {
	for _, framework := range frameworks {
		if framework.Framework != fwName {
			continue
		}
		if s, ok := framework.Structs[name]; ok {
			return &s
		}
	}
	return nil
}

// lookupProtocol mirrors lookupStruct for protocol entries: forward declarations
// across many frameworks would otherwise win the registry by alphabetical
// order. Comparing method counts at registration time keeps the canonical
// definition's framework as the owner.
func lookupProtocol(frameworks []*meta.FrameworkMeta, fwName, name string) *meta.Protocol {
	for _, framework := range frameworks {
		if framework.Framework != fwName {
			continue
		}
		if p, ok := framework.Protocols[name]; ok {
			return &p
		}
	}
	return nil
}

// protocolNameMatchesFramework reports whether `protoName` looks like a
// protocol that originates from `fwName`. Apple's frameworks consistently
// prefix their types with a 2–4 character abbreviation (MTL → Metal, NS →
// Foundation, AV → AVFoundation, CA → CoreAnimation, …). The registry uses
// this only as a tie-breaker when method counts are equal — never to override
// a more-complete entry.
func protocolNameMatchesFramework(protoName, fwName string) bool {
	// Mapping from Apple's ObjC type-prefix abbreviation to the owning framework
	// name (as returned by framework.Framework). When Apple ships a new framework whose
	// protocols appear in multiple header trees, add an entry here so ownership
	// tie-breaking works correctly. Prefix must uniquely identify the framework —
	// check that the two-letter code isn't already used by another entry.
	prefixes := map[string]string{
		"MTL": "Metal",
		"NS":  "Foundation",
		"CA":  "CoreAnimation",
		"AV":  "AVFoundation",
		"CI":  "CoreImage",
		"CG":  "CoreGraphics",
		"CF":  "CoreFoundation",
		"CB":  "CoreBluetooth",
		"CL":  "CoreLocation",
		"CM":  "CoreMedia",
		"SC":  "ScreenCaptureKit",
		"SK":  "SceneKit",
		"SF":  "SecurityInterface",
		"UN":  "UserNotifications",
		"UT":  "UniformTypeIdentifiers",
		"VZ":  "Virtualization",
		"WK":  "WebKit",
	}
	for prefix, owner := range prefixes {
		if strings.HasPrefix(protoName, prefix) {
			// The next character after the prefix must be uppercase or end
			// (NSCopying ✓, NSObject ✓, MTLBuffer ✓; but "NSomething" ×).
			rest := protoName[len(prefix):]
			if rest == "" {
				return owner == fwName
			}
			c := rest[0]
			if c >= 'A' && c <= 'Z' {
				return owner == fwName
			}
		}
	}
	return false
}

// capitaliseFirst uppercases the first ASCII character of s.
func capitaliseFirst(s string) string {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return s
	}
	return string(s[0]-32) + s[1:]
}

// buildProtocolProxyIndex inspects class method return types and records
// any single-protocol id<P> returns in dst. Only protocols already in
// knownProtocols are recorded (forward-declared ones are ignored).
func buildProtocolProxyIndex(classes map[string]meta.Class, knownProtocols, dst map[string]string) {
	for _, cls := range classes {
		for _, method := range cls.Methods {
			recordIDProtocolReturn(method.Return.ObjCType, knownProtocols, dst)
		}
	}
}

// indexProtocolProxiesFromMethods inspects foreign-extension method return
// types for single-protocol id<P> returns.
func indexProtocolProxiesFromMethods(
	extensions map[string][]meta.Method,
	knownProtocols, dst map[string]string,
) {
	for _, methods := range extensions {
		for _, method := range methods {
			recordIDProtocolReturn(method.Return.ObjCType, knownProtocols, dst)
		}
	}
}

// recordIDProtocolReturn extracts the protocol name from a single-protocol
// id<Protocol> ObjC type string and adds it to dst if the protocol is known.
func recordIDProtocolReturn(objcType string, knownProtocols, dst map[string]string) {
	t := strings.TrimSpace(objcType)
	if !strings.HasPrefix(t, "id<") {
		return
	}
	open := strings.Index(t, "<")
	close := strings.Index(t, ">")
	if open < 0 || close <= open {
		return
	}
	inner := strings.TrimSpace(t[open+1 : close])
	// Multi-protocol id<P1,P2> cannot map to a single wrapper type.
	if strings.Contains(inner, ",") {
		return
	}
	proto := inner
	if owner, ok := knownProtocols[proto]; ok {
		dst[proto] = owner
	}
}

// SuperclassIndex returns the set of class names that appear as the Super
// field of at least one other class across all loaded frameworks.
// This identifies classes that Apple itself subclasses — the population
// that SubclassSpec targets.
func (r *Registry) SuperclassIndex() map[string]bool {
	index := make(map[string]bool)
	for _, framework := range r.Frameworks {
		for _, cls := range framework.Classes {
			if cls.Super != "" {
				index[cls.Super] = true
			}
		}
	}
	return index
}
