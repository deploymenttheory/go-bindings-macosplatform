package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/overrides"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/purecg/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/metadata/objcclasshierarchy"
)

// Registry holds combined metadata from all loaded frameworks. It is the
// single source of truth for cross-framework type resolution.
type Registry struct {
	Frameworks        []*meta.FrameworkMeta
	ClassNameIndex    map[string]bool
	GenericClasses    map[string]bool
	GenericParamIndex map[string][]string
	OwnerIndex        map[string]string // className → framework name
	ProtocolIndex     map[string]string // protocol name → owning framework
	EnumIndex         map[string]string // enum name → owning framework
	EnumGoTypeIndex   map[string]string // enum name → underlying Go type
	TypedefIndex      map[string]string // typedef name → target ObjC qualType
	TypedefOwnerIndex map[string]string
	StructIndex       map[string]string // struct name → owning framework
	CFTypeIndex       map[string]string // CF opaque typedef → owning framework
	ClassIndex        map[string]meta.Class
	// UnavailableClasses is the set of ObjC class names that have IsUnavailable=true.
	// The mapper degrades references to these classes to objc.ID.
	UnavailableClasses map[string]bool
	// UnavailableEnumBaseTypes maps unavailable enum names → underlying Go type.
	// The mapper degrades references to these enums to their base integer type.
	UnavailableEnumBaseTypes map[string]string
	// BlockedImports[src][dst]=true means src must not import dst (cycle break).
	BlockedImports map[string]map[string]bool
	// ModulePrefix is the Go module path prefix for framework packages.
	// e.g. "github.com/deploymenttheory/go-bindings-macosplatform/purego-frameworks"
	ModulePrefix string
	// LibraryModulePrefix is the module path prefix for C library packages.
	LibraryModulePrefix string
}

// LoadAll reads .gometa.json files from the given paths (files or directories)
// and builds a Registry. Directories are scanned one level deep; arm64 files
// are preferred when multiple arch variants exist.
func LoadAll(paths []string, modulePrefix, libraryModulePrefix string) (*Registry, error) {
	reg := &Registry{
		ClassNameIndex:           make(map[string]bool),
		GenericClasses:           make(map[string]bool),
		GenericParamIndex:        make(map[string][]string),
		OwnerIndex:               make(map[string]string),
		ProtocolIndex:            make(map[string]string),
		EnumIndex:                make(map[string]string),
		EnumGoTypeIndex:          make(map[string]string),
		TypedefIndex:             make(map[string]string),
		TypedefOwnerIndex:        make(map[string]string),
		StructIndex:              make(map[string]string),
		CFTypeIndex:              make(map[string]string),
		ClassIndex:               make(map[string]meta.Class),
		UnavailableClasses:       make(map[string]bool),
		UnavailableEnumBaseTypes: make(map[string]string),
		BlockedImports:           make(map[string]map[string]bool),
		ModulePrefix:             modulePrefix,
		LibraryModulePrefix:      libraryModulePrefix,
	}

	resolved, err := resolvePaths(paths)
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("no .gometa.json files found in: %v", paths)
	}

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

	// Infer ParentFramework from path depth.
	fwDirPaths := make(map[string]string)
	for _, lf := range all {
		dir := filepath.Clean(filepath.Dir(lf.path))
		fwDirPaths[dir] = lf.framework.Framework
	}
	for _, lf := range all {
		if lf.framework.ParentFramework != "" {
			continue
		}
		dir := filepath.Dir(lf.path)
		grandpar := filepath.Clean(filepath.Dir(dir))
		if parentFW, ok := fwDirPaths[grandpar]; ok && parentFW != lf.framework.Framework {
			lf.framework.ParentFramework = parentFW
		}
	}

	// Deduplicate: top-level (no parent) wins over sub-framework.
	byName := make(map[string]fmWithPath)
	for _, lf := range all {
		prev, exists := byName[lf.framework.Framework]
		if !exists {
			byName[lf.framework.Framework] = lf
			continue
		}
		prevIsTop := prev.framework.ParentFramework == ""
		curIsTop := lf.framework.ParentFramework == ""
		if curIsTop && !prevIsTop {
			byName[lf.framework.Framework] = lf
		}
	}

	var frameworks []*meta.FrameworkMeta
	for _, lf := range all {
		if byName[lf.framework.Framework].path != lf.path {
			continue
		}
		frameworks = append(frameworks, lf.framework)
		reg.Frameworks = append(reg.Frameworks, lf.framework)
	}

	// Pass 2: canonical class ownership — fewest non-ghost methods wins.
	type classEntry struct {
		framework string
		score     int
		cls       meta.Class
	}
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
		var nonGhost []classEntry
		for _, e := range entries {
			if e.score > 0 {
				nonGhost = append(nonGhost, e)
			}
		}
		candidates := nonGhost
		if len(candidates) == 0 {
			candidates = entries
		}
		winner := candidates[0]
		for _, e := range candidates[1:] {
			if e.score < winner.score ||
				(e.score == winner.score && e.framework < winner.framework) {
				winner = e
			}
		}
		best[name] = bestEntry(winner)
	}

	for name, entry := range best {
		reg.ClassNameIndex[name] = true
		reg.OwnerIndex[name] = entry.framework
		reg.ClassIndex[name] = entry.cls
		if len(entry.cls.GenericParams) > 0 {
			reg.GenericClasses[name] = true
			reg.GenericParamIndex[name] = entry.cls.GenericParams
		}
		if entry.cls.Availability.IsUnavailable {
			reg.UnavailableClasses[name] = true
		}
	}

	// Also mark unavailable classes from all frameworks so cross-framework
	// references to unavailable types degrade to objc.ID.
	for _, framework := range frameworks {
		for name, cls := range framework.Classes {
			if cls.Availability.IsUnavailable {
				reg.UnavailableClasses[name] = true
			}
		}
	}

	// Pass 3: enum registry — first-write-wins.
	for _, framework := range frameworks {
		for enumName, enum := range framework.Enums {
			if enum.Availability.IsUnavailable {
				// Track unavailable enums so the mapper can degrade references
				// to them to their underlying integer type.
				if enum.GoType != "" {
					if _, already := reg.UnavailableEnumBaseTypes[enumName]; !already {
						reg.UnavailableEnumBaseTypes[enumName] = enum.GoType
					}
				}
				continue // don't add to EnumIndex — skip the unavailable enum
			}
			if _, already := reg.EnumIndex[enumName]; !already {
				reg.EnumIndex[enumName] = framework.Framework
			}
			if enum.GoType != "" {
				if _, already := reg.EnumGoTypeIndex[enumName]; !already {
					reg.EnumGoTypeIndex[enumName] = enum.GoType
				}
				goName := capitaliseFirst(enumName)
				if _, already := reg.EnumGoTypeIndex[goName]; !already {
					reg.EnumGoTypeIndex[goName] = enum.GoType
				}
			}
		}
	}

	// Pass 4: protocol registry — highest method+inherited count wins.
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
			curPrefixMatch := protocolNameMatchesFramework(protoName, framework.Framework)
			priorPrefixMatch := protocolNameMatchesFramework(protoName, existing)
			if curPrefixMatch && !priorPrefixMatch {
				reg.ProtocolIndex[protoName] = framework.Framework
				continue
			}
			if priorPrefixMatch && !curPrefixMatch {
				continue
			}
			if framework.Framework < existing {
				reg.ProtocolIndex[protoName] = framework.Framework
			}
		}
	}

	// Pass 5: typedef map — first-write-wins.
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

	// Pass 6: struct registry — prefer non-empty definitions.
	for _, framework := range frameworks {
		for name, s := range framework.Structs {
			existingOwner, has := reg.StructIndex[name]
			if !has {
				if len(s.Fields) > 0 {
					reg.StructIndex[name] = framework.Framework
				}
				continue
			}
			if len(s.Fields) > 0 {
				prior := lookupStruct(frameworks, existingOwner, name)
				if prior == nil || len(prior.Fields) == 0 {
					reg.StructIndex[name] = framework.Framework
					continue
				}
				if framework.Framework < existingOwner {
					reg.StructIndex[name] = framework.Framework
				}
			}
		}
	}

	// Pass 6b: typedef aliases for underscore-prefixed struct tags.
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

	// Pass 6c: framework-specific opaque CF pointer typedefs.
	for _, framework := range frameworks {
		zeroFieldStructs := make(map[string]bool)
		for name, s := range framework.Structs {
			if len(s.Fields) == 0 {
				zeroFieldStructs[name] = true
			}
		}
		for tName, target := range framework.Typedefs {
			if typemap.IsCoreFoundationOpaqueRef(tName) {
				continue
			}
			t := strings.TrimSpace(target)
			t = strings.TrimPrefix(t, "const ")
			t = strings.TrimSpace(t)
			if !strings.HasPrefix(t, "struct ") {
				continue
			}
			bare := strings.TrimSpace(t[7:])
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

	// Warn about sub-frameworks loaded without parent.
	loadedFWs := make(map[string]bool, len(reg.Frameworks))
	for _, framework := range reg.Frameworks {
		loadedFWs[strings.ToLower(framework.Framework)] = true
	}
	for _, framework := range reg.Frameworks {
		if framework.ParentFramework != "" &&
			!loadedFWs[strings.ToLower(framework.ParentFramework)] {
			fmt.Fprintf(os.Stderr,
				"warning: sub-framework %q loaded without parent %q\n",
				framework.Framework, framework.ParentFramework)
		}
	}

	if err := validateSuperchainAcyclic(reg.ClassIndex); err != nil {
		return nil, err
	}

	filterInheritedClassMethods(reg)
	detectAndBreakCycles(reg)

	// Some cycles are hidden behind C typedef chains (e.g. dispatch_queue_t →
	// NSObject from Foundation) that the identifier-based edge scanner can't see.
	// Apply stable overrides for these known pairs: block the lower-level framework
	// from importing the higher-level one.
	applyHardcodedBlocks(reg)

	return reg, nil
}

// applyHardcodedBlocks adds BlockedImports entries for cycles that are hidden
// behind typedef chains and therefore not detectable by identifier scanning.
// The direction blocked is always the "more surprising" one (e.g. CoreFoundation
// importing Foundation) — keeping the more natural dependency direction open.
func applyHardcodedBlocks(reg *Registry) {
	known := [][2]string{
		// dispatch_queue_t in CF functions resolves to foundation.NSObject
		{"CoreFoundation", "Foundation"},
		// NSUserNotification (deprecated) is owned by Foundation but references AppKit
		// types. Foundation is a lower-level framework and should not import AppKit.
		{"Foundation", "AppKit"},
		// Security C functions reference dispatch_queue_t → foundation.NSObject
		{"Security", "Foundation"},
		// audiotoolbox → soundanalysis → avfaudio → audiotoolbox 3-node cycle
		{"AVFAudio", "AudioToolbox"},
		// AppKit NSTextView references id<QLPreviewItem> from QuickLookUI;
		// keep quicklookui→appkit (QLPreviewPanel is NSPanel subclass)
		{"AppKit", "QuickLookUI"},
		// AppKit protocols embed CloudKit protocols; CloudKit depends on Foundation
		// not AppKit → block AppKit from importing CloudKit
		{"AppKit", "CloudKit"},
		// CoreData NSPersistentCloudKitContainer references CloudKit; block
		// CoreData from importing CloudKit to prevent appkit→coredata→cloudkit→foundation cycle
		{"CoreData", "CloudKit"},
		// CoreData NSCoreDataCoreSpotlightDelegate references CoreSpotlight;
		// CoreSpotlight should not depend on CoreData
		{"CoreData", "CoreSpotlight"},
		// IOBluetooth and IOBluetoothUI mutual dependency; UI layer (IOBluetoothUI)
		// wraps core layer (IOBluetooth), not the other way around
		{"IOBluetooth", "IOBluetoothUI"},
		// Photos and PhotosUI mutual dependency; PhotosUI wraps Photos types so
		// Photos (the data layer) should not import PhotosUI (the view layer)
		{"Photos", "PhotosUI"},
		// AVFoundation AVAsynchronousCIImageFilteringRequest references CoreImage;
		// CoreImage is more fundamental than AVFoundation
		{"AVFoundation", "CoreImage"},
		// CloudKit references Contacts types; Contacts should not depend on CloudKit
		{"CloudKit", "Contacts"},
		// CloudKit CK* operations inherit from NSOperation (Foundation); Foundation
		// should not import CloudKit
		{"Foundation", "CloudKit"},
		// IOBluetooth device selector controller is an AppKit panel subclass;
		// AppKit should not depend on IOBluetooth
		{"AppKit", "IOBluetooth"},
		// Photos PHLivePhotoView is an AppKit view subclass; AppKit should not
		// import Photos
		{"AppKit", "Photos"},
		// SpriteKit SK3DNode wraps SceneKit nodes; SpriteKit depends on SceneKit
		{"SceneKit", "SpriteKit"},
		// TCL/TK C libraries: TK functions import TCL; block the reverse
		{"Tcl", "Tk"},
		// CoreFoundation C functions reference CarbonCore struct types (FSRef for
		// CFURL), but CarbonCore is a higher-level legacy layer; block CF from
		// importing CarbonCore so CarbonCore can safely import CF types (CFUUIDBytes).
		{"CoreFoundation", "CarbonCore"},
		// Quartz is a meta-framework that bundles QuartzComposer; the sub-framework
		// should reference Quartz types, not the other way around.
		{"Quartz", "QuartzComposer"},
		// Same meta-framework relationship: Quartz bundles ImageKit.
		{"Quartz", "ImageKit"},
		// MPSCore is the lower-level sublayer of MetalPerformanceShaders; it must
		// not import the higher-level umbrella.
		{"MPSCore", "MetalPerformanceShaders"},
	}
	fwLoaded := make(map[string]bool)
	for _, fw := range reg.Frameworks {
		fwLoaded[fw.Framework] = true
	}
	for _, pair := range known {
		src, dst := pair[0], pair[1]
		if !fwLoaded[src] || !fwLoaded[dst] {
			continue
		}
		// Always apply the specified direction. The cycle detector may have
		// blocked the reverse; that's fine — both being blocked is safe.
		if reg.BlockedImports[src] == nil {
			reg.BlockedImports[src] = make(map[string]bool)
		}
		reg.BlockedImports[src][dst] = true
		// Also unblock the reverse if it was blocked — we want the specified
		// direction to carry, not the detector's guess.
		if reg.BlockedImports[dst] != nil {
			delete(reg.BlockedImports[dst], src)
		}
	}
}

// filterInheritedClassMethods strips inherited instance methods from every
// ObjC class using the canonical ObjCClassSuperclassIndex hierarchy map.
// This is the same approach as the CGo generator's filterInheritedClassMethods.
func filterInheritedClassMethods(reg *Registry) {
	ownerFramework := make(map[string]*meta.FrameworkMeta, len(reg.OwnerIndex))
	for _, framework := range reg.Frameworks {
		for className := range framework.Classes {
			if reg.OwnerIndex[className] == framework.Framework {
				ownerFramework[className] = framework
			}
		}
	}

	for className := range reg.ClassIndex {
		ancestorSelectors := make(map[string]bool)
		// Walk superclass chain via canonical hierarchy — same as CGo generator.
		for super := objcclasshierarchy.ObjCClassSuperclassIndex[className]; super != ""; super = objcclasshierarchy.ObjCClassSuperclassIndex[super] {
			parentClass, ok := reg.ClassIndex[super]
			if !ok {
				break
			}
			for _, method := range parentClass.Methods {
				if isMethodBridgeable(method) {
					ancestorSelectors[method.Selector] = true
				}
			}
		}

		if len(ancestorSelectors) == 0 {
			continue
		}

		class := reg.ClassIndex[className]
		novel := make([]meta.Method, 0, len(class.Methods))
		for _, method := range class.Methods {
			if method.IsInit || method.IsClassMethod {
				novel = append(novel, method)
				continue
			}
			if !ancestorSelectors[method.Selector] {
				novel = append(novel, method)
			}
		}

		if len(novel) == len(class.Methods) {
			continue
		}
		class.Methods = novel
		reg.ClassIndex[className] = class
		if framework, ok := ownerFramework[className]; ok {
			framework.Classes[className] = class
		}
	}
}

// isMethodBridgeable reports whether an ObjC method can be bridged.
func isMethodBridgeable(method meta.Method) bool {
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

// isSelectorFormatVariadic returns true when the selector contains "format".
func isSelectorFormatVariadic(selector string) bool {
	for _, part := range strings.Split(selector, ":") {
		if strings.Contains(strings.ToLower(part), "format") {
			return true
		}
	}
	return false
}

// detectAndBreakCycles computes import edges from exact cross-framework
// class/struct/enum/protocol references in class definitions and method
// signatures, then iteratively breaks cycles by blocking the lowest-weight edge.
func detectAndBreakCycles(reg *Registry) {
	edges := buildAllImportEdges(reg)

	for {
		cycle := findCycle(edges)
		if cycle == nil {
			break
		}
		minWeight := -1
		minSrc, minDst := "", ""
		for i, src := range cycle {
			dst := cycle[(i+1)%len(cycle)]
			w := edges[src][dst]
			if minWeight < 0 || w < minWeight || (w == minWeight && src > minSrc) {
				minWeight = w
				minSrc = src
				minDst = dst
			}
		}
		if minSrc == "" {
			break
		}
		if reg.BlockedImports[minSrc] == nil {
			reg.BlockedImports[minSrc] = make(map[string]bool)
		}
		reg.BlockedImports[minSrc][minDst] = true
		delete(edges[minSrc], minDst)
	}
}

// buildAllImportEdges builds the cross-framework import graph using three
// exact, registry-based sources — no string scanning.
//
//  1. Class superclass edges from ObjCClassSuperclassIndex (canonical hierarchy).
//  2. Class protocol conformance: if class C conforms to protocol P from
//     framework Y, framework(C) → Y.
//  3. Protocol inheritance: if protocol Q inherits from protocol P from
//     framework Y, framework(Q) → Y.
func buildAllImportEdges(reg *Registry) map[string]map[string]int {
	edges := make(map[string]map[string]int)

	add := func(src, dst string) {
		if src == "" || dst == "" || src == dst {
			return
		}
		if edges[src] == nil {
			edges[src] = make(map[string]int)
		}
		edges[src][dst]++
	}

	// Source 1: class superclass edges from canonical hierarchy.
	for className, superName := range objcclasshierarchy.ObjCClassSuperclassIndex {
		if superName == "" {
			continue
		}
		ownerSrc := reg.OwnerIndex[className]
		ownerDst := reg.OwnerIndex[superName]
		if ownerSrc == "" || ownerDst == "" || ownerSrc == ownerDst {
			continue
		}
		add(ownerSrc, ownerDst)
	}

	// Source 2: class protocol conformance edges.
	for _, framework := range reg.Frameworks {
		fw := framework.Framework
		for _, cls := range framework.Classes {
			ownerSrc := reg.OwnerIndex[cls.Super] // skip if class not canonical owner
			_ = ownerSrc
			for _, protoName := range cls.Protocols {
				if ownerDst, ok := reg.ProtocolIndex[protoName]; ok && ownerDst != fw {
					add(fw, ownerDst)
				}
			}
		}

		// Source 3: protocol inheritance edges.
		for protoName, proto := range framework.Protocols {
			ownerSrc, ok := reg.ProtocolIndex[protoName]
			if !ok || ownerSrc != fw {
				continue // only canonical owner emits the protocol
			}
			for _, parentProto := range proto.InheritedProtocols {
				if ownerDst, ok2 := reg.ProtocolIndex[parentProto]; ok2 && ownerDst != fw {
					add(fw, ownerDst)
				}
			}
		}
	}

	// Source 4: function and method type references.
	// Scans ObjC type strings for identifiers that resolve to types owned by
	// other frameworks, catching cycles that are invisible from class/protocol
	// structure alone (e.g. C function params that reference cross-framework structs).
	reTypeName := regexp.MustCompile(`\b([A-Z][A-Za-z0-9_]*)\b`)

	addFromTypeStr := func(src, typeStr string) {
		for _, match := range reTypeName.FindAllStringSubmatch(typeStr, -1) {
			name := match[1]
			for _, idx := range []map[string]string{
				reg.OwnerIndex, reg.StructIndex, reg.EnumIndex, reg.TypedefOwnerIndex,
			} {
				if dst, ok := idx[name]; ok && dst != src {
					add(src, dst)
					break
				}
			}
		}
	}

	for _, framework := range reg.Frameworks {
		fw := framework.Framework
		for _, fn := range framework.Functions {
			for _, p := range fn.Params {
				addFromTypeStr(fw, p.ObjCType)
			}
			addFromTypeStr(fw, fn.Return.ObjCType)
		}
		for className, cls := range framework.Classes {
			if reg.OwnerIndex[className] != fw {
				continue // only scan methods for classes this framework owns
			}
			for _, method := range cls.Methods {
				for _, p := range method.Params {
					addFromTypeStr(fw, p.ObjCType)
				}
				addFromTypeStr(fw, method.Return.ObjCType)
			}
			for _, prop := range cls.Properties {
				addFromTypeStr(fw, prop.ObjCType)
			}
		}
	}

	return edges
}

// findCycle uses iterative DFS with explicit path tracking to find a cycle.
// Returns the nodes forming the cycle, or nil if none exists.
// The path-tracking approach guarantees the returned slice is a valid cycle.
func findCycle(edges map[string]map[string]int) []string {
	// All nodes (keys and values).
	nodeSet := make(map[string]bool)
	for src, dsts := range edges {
		nodeSet[src] = true
		for dst := range dsts {
			nodeSet[dst] = true
		}
	}
	var nodes []string
	for n := range nodeSet {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)

	visited := make(map[string]bool)

	var path []string
	inPath := make(map[string]bool)

	var dfs func(n string) []string
	dfs = func(n string) []string {
		if inPath[n] {
			// Found a cycle: extract the cycle portion of path starting at n.
			for i, node := range path {
				if node == n {
					cycle := make([]string, len(path)-i)
					copy(cycle, path[i:])
					return cycle
				}
			}
			return []string{n}
		}
		if visited[n] {
			return nil
		}
		visited[n] = true
		inPath[n] = true
		path = append(path, n)

		// Sort destinations for determinism.
		dsts := make([]string, 0, len(edges[n]))
		for d := range edges[n] {
			dsts = append(dsts, d)
		}
		sort.Strings(dsts)

		for _, d := range dsts {
			if cycle := dfs(d); cycle != nil {
				path = path[:len(path)-1]
				inPath[n] = false
				return cycle
			}
		}

		path = path[:len(path)-1]
		inPath[n] = false
		return nil
	}

	for _, n := range nodes {
		if !visited[n] {
			if cycle := dfs(n); cycle != nil {
				return cycle
			}
		}
	}
	return nil
}

// SortByDependency returns frameworks in topological order (dependencies first).
func SortByDependency(reg *Registry) []*meta.FrameworkMeta {
	// Build adjacency: framework → set of frameworks it depends on (superclass).
	deps := make(map[string]map[string]bool)
	for _, framework := range reg.Frameworks {
		deps[framework.Framework] = make(map[string]bool)
		for _, cls := range framework.Classes {
			if cls.Super != "" {
				if owner, ok := reg.OwnerIndex[cls.Super]; ok && owner != framework.Framework {
					deps[framework.Framework][owner] = true
				}
			}
		}
	}

	// Kahn's algorithm.
	inDegree := make(map[string]int)
	for _, framework := range reg.Frameworks {
		if _, ok := inDegree[framework.Framework]; !ok {
			inDegree[framework.Framework] = 0
		}
		for dep := range deps[framework.Framework] {
			inDegree[dep] = inDegree[dep] // ensure key exists
			inDegree[framework.Framework]++
		}
	}

	// Recalculate in-degree correctly.
	inDeg := make(map[string]int)
	for _, framework := range reg.Frameworks {
		for dep := range deps[framework.Framework] {
			inDeg[dep]++
		}
	}

	queue := []string{}
	for _, framework := range reg.Frameworks {
		if inDeg[framework.Framework] == 0 {
			queue = append(queue, framework.Framework)
		}
	}
	sort.Strings(queue)

	fwByName := make(map[string]*meta.FrameworkMeta)
	for _, framework := range reg.Frameworks {
		fwByName[framework.Framework] = framework
	}

	var result []*meta.FrameworkMeta
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		if fw, ok := fwByName[n]; ok {
			result = append(result, fw)
		}
		// Find all frameworks that depend on n and decrement their in-degree.
		var next []string
		for _, framework := range reg.Frameworks {
			if deps[framework.Framework][n] {
				inDeg[framework.Framework]--
				if inDeg[framework.Framework] == 0 {
					next = append(next, framework.Framework)
				}
			}
		}
		sort.Strings(next)
		queue = append(queue, next...)
	}

	// Append any remaining (cycles broken by BlockedImports).
	added := make(map[string]bool)
	for _, fw := range result {
		added[fw.Framework] = true
	}
	var remaining []*meta.FrameworkMeta
	for _, fw := range reg.Frameworks {
		if !added[fw.Framework] {
			remaining = append(remaining, fw)
		}
	}
	sort.Slice(remaining, func(i, j int) bool {
		return remaining[i].Framework < remaining[j].Framework
	})
	return append(result, remaining...)
}

// validateSuperchainAcyclic checks for circular inheritance.
func validateSuperchainAcyclic(allClasses map[string]meta.Class) error {
	visited := make(map[string]bool, len(allClasses))
	inStack := make(map[string]bool)

	var dfs func(name string) error
	dfs = func(name string) error {
		if inStack[name] {
			return fmt.Errorf("superclass cycle detected at %q", name)
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

func resolvePaths(paths []string) ([]string, error) {
	var out []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", p, err)
		}
		if info.IsDir() {
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

func protocolNameMatchesFramework(protoName, fwName string) bool {
	prefixes := map[string]string{
		"MTL": "Metal", "NS": "Foundation", "CA": "CoreAnimation",
		"AV": "AVFoundation", "CI": "CoreImage", "CG": "CoreGraphics",
		"CF": "CoreFoundation", "CB": "CoreBluetooth", "CL": "CoreLocation",
		"CM": "CoreMedia", "SC": "ScreenCaptureKit", "SK": "SceneKit",
		"SF": "SecurityInterface", "UN": "UserNotifications",
		"UT": "UniformTypeIdentifiers", "VZ": "Virtualization", "WK": "WebKit",
	}
	for prefix, owner := range prefixes {
		if strings.HasPrefix(protoName, prefix) {
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

func capitaliseFirst(s string) string {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return s
	}
	return string(s[0]-32) + s[1:]
}
