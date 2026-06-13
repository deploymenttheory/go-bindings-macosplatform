package naming

import (
	"fmt"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// ResolveGoMethodName resolves the final Go name for a method when its base
// name (from MethodName) collides with another method on the same class
// (goNameCount[gn] > 1).
//
// Three rules in priority order:
//
//  1. Zero-arg priority: the zero-arg form (-[Foo bar]) owns the clean base
//     name. The one-arg overload (-[Foo bar:]) is resolved by rules 2 or 3.
//     Rationale: in Cocoa the zero-arg form is always the direct programmatic
//     call; any n-arg overload is a specialisation.
//
//  2. IBAction omission: a one-arg method whose sole argument is the bare `id`
//     type AND whose zero-arg sibling exists is a Cocoa IBAction overload used
//     only by Interface Builder — never from code. Returns skip=true so the
//     caller omits the method entirely.
//     Example: -[WKWebView goBack:(id)sender] vs -[WKWebView goBack]
//
//  3. Last resort — genuine selector collision: two ObjC selectors produce the
//     same Go name (e.g. transformContextForBox: and transformContext:forBox:).
//     The colon count is appended for a deterministic, stable suffix.
//
// Class methods are excluded from rules 1–2 because they occupy a separate Go
// namespace (ClassName-prefixed package-level functions). The rare collision
// between two class methods falls directly to rule 3.
func ResolveGoMethodName(gn string, method meta.Method, classMethods []meta.Method, goNameCount map[string]int) (resolved string, skip bool) {
	if goNameCount[gn] <= 1 || method.IsClassMethod {
		return gn, false
	}

	nColons := strings.Count(method.Selector, ":")

	switch nColons {
	case 0:
		// Rule 1 (zero-arg side): if a one-arg sibling (selector + ":") exists,
		// this is the primary form — own the base name unchanged.
		oneSel := method.Selector + ":"
		if m1, ok := FindSiblingMethod(classMethods, oneSel, false); ok && !m1.Availability.IsUnavailable {
			return gn, false
		}

	case 1:
		// Rule 1 (one-arg side) + rule 2: check against the zero-arg sibling.
		// Only applies when the selector is a simple "name:" form.
		baseSel := strings.TrimSuffix(method.Selector, ":")
		if !strings.Contains(baseSel, ":") {
			if base, ok := FindSiblingMethod(classMethods, baseSel, false); ok && !base.Availability.IsUnavailable {
				if len(method.Params) == 1 && IsIBActionSender(method.Params[0].ObjCType) {
					// Rule 2: IBAction — skip entirely.
					return gn, true
				}
				// Zero-arg sibling exists but this is not an IBAction (e.g.
				// drawKnob / drawKnob:(NSRect)). Apply rule 3 with suffix so
				// the zero-arg form keeps the clean name.
			}
		}
	}

	// Rule 3: genuine collision — append colon count as a deterministic suffix.
	nArgs := strings.Count(method.Selector, ":")
	return fmt.Sprintf("%s%d", gn, nArgs), false
}

// IsIBActionSender reports whether an ObjC type string is the bare `id` sender
// type used exclusively in Interface Builder action wiring.
func IsIBActionSender(objcType string) bool {
	t := strings.TrimSpace(objcType)
	switch t {
	case "id", "id _Nullable", "id _Nonnull", "_Nullable id", "_Nonnull id":
		return true
	}
	return false
}

// FindSiblingMethod returns the first method in methods that matches both
// selector and isClassMethod. Used to locate the zero-arg counterpart when
// classifying a one-arg method under disambiguation rules 1 and 2.
func FindSiblingMethod(methods []meta.Method, selector string, isClassMethod bool) (meta.Method, bool) {
	for _, m := range methods {
		if m.Selector == selector && m.IsClassMethod == isClassMethod {
			return m, true
		}
	}
	return meta.Method{}, false
}
