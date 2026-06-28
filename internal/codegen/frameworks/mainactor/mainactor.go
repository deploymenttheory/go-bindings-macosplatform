// Package mainactor applies the @MainActor isolation sidecar
// (internal/mainactor) to the purego generator's metadata model so the
// idiomatic emitter can wrap main-thread-bound calls in purego.Main.
//
// Two steps, mirroring how Apple actually models isolation:
//
//   - ApplyAdjacent stamps the facts a framework's own sidecar carries: whole
//     classes that are @MainActor, individually-isolated selectors, and the
//     `nonisolated` opt-outs.
//   - Propagate then pushes class-level isolation DOWN the Objective-C class
//     hierarchy across all frameworks, because Apple isolates a small root set
//     (NSView, NSWindow, …) and every subclass inherits it implicitly. This is
//     what makes MKMapView / PDFView / SCNView main-thread-bound even though
//     their own frameworks declare no @MainActor classes.
//
// The file format and discovery convention are identical to the appledocs
// sidecar; only the mutated fields differ.
package mainactor

import (
	"fmt"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	rootactor "github.com/deploymenttheory/go-bindings-macosplatform/internal/mainactor"
)

// ApplyAdjacent looks for a mainactor.json sidecar next to metaPath and applies
// its directly-stated facts to framework. A missing sidecar is not an error.
// Hierarchy propagation is a separate, cross-framework pass (see Propagate).
func ApplyAdjacent(metaPath string, framework *meta.FrameworkMeta) error {
	iso, found, err := rootactor.LoadAdjacent(metaPath)
	if err != nil {
		return fmt.Errorf("load mainactor sidecar for %s: %w", metaPath, err)
	}
	if !found {
		return nil
	}
	Apply(iso, framework)
	return nil
}

// Apply stamps iso's directly-stated isolation onto framework in place:
// whole-class flags, per-selector isolation, and `nonisolated` exemptions.
func Apply(iso *rootactor.Isolation, framework *meta.FrameworkMeta) {
	if iso == nil {
		return
	}

	mainClasses := toSet(iso.MainActorClasses)

	for name, cls := range framework.Classes {
		changed := false

		if mainClasses[name] {
			cls.IsMainThreadRequired = true
			changed = true
		}

		// `nonisolated` opt-outs win over everything, so mark them first.
		exempt := selectorSet(iso.NonisolatedSelectors[name])
		// Individually-isolated selectors on a class that is not wholly isolated.
		isolated := selectorSet(iso.MainActorSelectors[name])

		for i := range cls.Methods {
			key := selectorKey(cls.Methods[i].Selector, cls.Methods[i].IsClassMethod)
			if exempt[key] {
				cls.Methods[i].IsMainThreadExempt = true
				changed = true
			}
			if isolated[key] {
				cls.Methods[i].IsMainThreadRequired = true
				changed = true
			}
		}

		if changed {
			framework.Classes[name] = cls
		}
	}
}

// Propagate pushes class-level @MainActor isolation down the class hierarchy
// across all loaded frameworks, then stamps the affected instance methods so
// the emitter (which keys off Method.IsMainThreadRequired) wraps them. Call once
// after every framework's sidecar has been applied.
func Propagate(frameworks []*meta.FrameworkMeta) {
	// Global super-chain (child → super) and the seed set of classes already
	// known to be main-thread, built from the loaded metadata itself (more
	// current than any committed hierarchy snapshot).
	super := map[string]string{}
	seed := map[string]bool{}
	for _, fw := range frameworks {
		for name, cls := range fw.Classes {
			if cls.Super != "" {
				super[name] = cls.Super
			}
			if cls.IsMainThreadRequired {
				seed[name] = true
			}
		}
	}

	// memo caches the resolved verdict per class name so each chain is walked
	// once even when many subclasses share ancestors.
	memo := map[string]bool{}
	var isMain func(name string) bool
	isMain = func(name string) bool {
		if v, ok := memo[name]; ok {
			return v
		}
		// Guard against the (theoretical) cyclic super chain by marking in
		// progress as false before recursing.
		memo[name] = false
		result := seed[name]
		if !result {
			if parent, ok := super[name]; ok {
				result = isMain(parent)
			}
		}
		memo[name] = result
		return result
	}

	for _, fw := range frameworks {
		for name, cls := range fw.Classes {
			if !isMain(name) {
				continue
			}
			cls.IsMainThreadRequired = true
			// Stamp instance methods so the emitter wraps them. Skip class
			// (factory) methods — Apple leaves those nonisolated even on
			// @MainActor classes — and honour explicit `nonisolated` exemptions.
			for i := range cls.Methods {
				if cls.Methods[i].IsClassMethod || cls.Methods[i].IsMainThreadExempt {
					continue
				}
				cls.Methods[i].IsMainThreadRequired = true
			}
			fw.Classes[name] = cls
		}
	}
}

func toSet(xs []string) map[string]bool {
	s := make(map[string]bool, len(xs))
	for _, x := range xs {
		s[x] = true
	}
	return s
}

// selectorSet keys sidecar selectors (class methods carry a leading "+") so they
// match a scanned Method via selectorKey.
func selectorSet(xs []string) map[string]bool {
	return toSet(xs)
}

func selectorKey(selector string, isClassMethod bool) string {
	if isClassMethod {
		return "+" + selector
	}
	return selector
}
