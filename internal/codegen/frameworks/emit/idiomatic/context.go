//go:build darwin

package idiomatic

import (
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// frameworkContext holds the per-framework derived data used during emission. It
// is created once at the start of EmitFrameworkWrappers and discarded when that
// function returns, so the emitter keeps no package-level mutable state (P6):
// what used to live in five maps keyed by *meta.FrameworkMeta now lives here for
// one framework's lifetime only.
//
// The first four sets are pure functions of the framework, computed up front. The
// last, referenced, is an accumulator: resolving a signature that names one of
// the framework's own enums records it here (see localizeEnumType), and emitEnums
// later emits a concrete definition for exactly this set. Because the context is
// local to one EmitFrameworkWrappers call, this accumulation is no longer order-
// dependent shared state.
type frameworkContext struct {
	fw     *meta.FrameworkMeta
	prefix string // common class-name prefix (e.g. "VZ"); from detectClassPrefix

	ownTypes    map[string]bool // exported Go type names the raw package defines
	ownEnums    map[string]bool // Go type names of the framework's own enums
	localStruct map[string]bool // value structs re-declared locally (plain fields)
	// classGoNames is the set of Go wrapper type names the framework's available
	// classes map to. It gates same-framework embedding: a base is embedded only
	// when its Go name is actually emitted here (and distinct from the subclass),
	// so a de-prefix collision or a class outside the framework never produces an
	// embed of an undefined type.
	classGoNames map[string]bool

	referenced map[string]bool // enum names a resolved signature localized
}

// newFrameworkContext computes the per-framework derived data once. The four
// pure sets mirror the former cache builders exactly; referenced starts empty
// and is filled as signatures are resolved.
func newFrameworkContext(fw *meta.FrameworkMeta) *frameworkContext {
	prefix := detectClassPrefix(fw)
	classGoNames := make(map[string]bool)
	for className, cls := range fw.Classes {
		if cls.Availability.IsUnavailable {
			continue
		}
		classGoNames[trialTypeName(className, prefix)] = true
	}
	return &frameworkContext{
		fw:           fw,
		prefix:       prefix,
		ownTypes:     buildOwnTypeNames(fw),
		ownEnums:     buildOwnEnumNames(fw),
		localStruct:  buildLocalValueStructNames(fw),
		classGoNames: classGoNames,
		referenced:   map[string]bool{},
	}
}

// buildOwnTypeNames builds the set of Go type names the raw package exports for
// fw, in both their metadata and exported spellings.
func buildOwnTypeNames(fw *meta.FrameworkMeta) map[string]bool {
	set := make(map[string]bool)
	for className := range fw.Classes {
		set[className] = true
	}
	for enumName := range fw.Enums {
		set[enumName] = true
		set[naming.GoTypeName(enumName)] = true
	}
	// Protocols that share a name with a class are emitted with a "Protocol"
	// suffix (e.g. protocol NSObject → type NSObjectProtocol interface).
	for protocolName := range fw.Protocols {
		goName := naming.GoTypeName(protocolName)
		set[goName] = true
		set[goName+"Protocol"] = true
	}
	for structName := range fw.Structs {
		set[structName] = true
		set[naming.ExportedTypeName(structName)] = true
	}
	// Struct typedefs (e.g. NSRange → struct _NSRange) are emitted as Go type
	// aliases in the raw package.
	for typedefName, target := range fw.Typedefs {
		if !strings.HasPrefix(target, "struct ") {
			continue
		}
		bare := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(target, "struct "), "*"))
		if _, ok := fw.Structs[bare]; ok {
			set[naming.ExportedTypeName(typedefName)] = true
		}
	}
	return set
}

// buildOwnEnumNames is the set of Go type names of fw's own enums, derived
// exactly like emitEnums' index (naming.GoTypeName on each exported, available,
// non-anon enum key with at least one available member).
func buildOwnEnumNames(fw *meta.FrameworkMeta) map[string]bool {
	set := make(map[string]bool)
	for key, e := range fw.Enums {
		if e.Availability.IsUnavailable || e.IsAnon {
			continue
		}
		goType := naming.GoTypeName(key)
		if !isExportedGoIdent(goType) {
			continue
		}
		// emitEnums only re-emits enums that have at least one available member.
		if !enumHasAvailableMember(e) {
			continue
		}
		set[goType] = true
	}
	return set
}

// buildLocalValueStructNames returns the set of exported names of value structs
// the framework re-declares locally — those with at least one field where every
// field is a plain value (a number or another such struct).
func buildLocalValueStructNames(fw *meta.FrameworkMeta) map[string]bool {
	set := make(map[string]bool)
	for name, s := range fw.Structs {
		if s.Availability.IsUnavailable || len(s.Fields) == 0 {
			continue
		}
		plain := true
		for _, f := range s.Fields {
			if f.GoType == "" || strings.ContainsAny(f.GoType, ".*[]") {
				plain = false
				break
			}
		}
		if plain {
			set[naming.ExportedTypeName(name)] = true
		}
	}
	return set
}
