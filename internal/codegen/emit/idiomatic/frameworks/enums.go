//go:build darwin

package idiofw

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/view"
	rawfw "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emitmanifest"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// recordIdiomaticEnum records an emitted idiomatic enum type and its available
// members into the parity manifest, keyed on the ObjC/C name (identical to the
// raw oracle's key) so the two styles join. It is a no-op on a nil recorder.
func recordIdiomaticEnum(rec *emitmanifest.Recorder, framework, pkgName, objcKey, localName string, enum meta.Enum) {
	if rec == nil {
		return
	}
	rec.Record(emitmanifest.Entry{
		Style:     emitmanifest.StyleIdiomatic,
		Kind:      emitmanifest.KindEnum,
		Framework: framework,
		MetaKey:   emitmanifest.MetaKey(framework, emitmanifest.KindEnum, objcKey, ""),
		GoPkg:     pkgName,
		GoSymbol:  localName,
	})
	for _, member := range enum.Members {
		if member.Availability.IsUnavailable {
			continue
		}
		rec.Record(emitmanifest.Entry{
			Style:     emitmanifest.StyleIdiomatic,
			Kind:      emitmanifest.KindEnumMember,
			Framework: framework,
			MetaKey:   emitmanifest.MetaKey(framework, emitmanifest.KindEnumMember, member.Name, ""),
			GoPkg:     pkgName,
			GoSymbol:  naming.GoTypeName(member.Name),
		})
	}
}

// emitEnums writes <pkgname>_enums_generated.go: a concrete Go re-emission (type
// + typed/valued const block + String) of every raw enum the idiomatic package's
// own generated code references — rendered via templates/enum.tmpl. It mirrors
// the raw bindings rather than aliasing to them, so the constant values, the
// underlying integer width, and the String() behaviour are all visible in the
// idiomatic package. Runs last (after every other *_generated.go file is on disk)
// and scans them for raw.<GoType>, keeping the surface minimal.
func emitEnums(
	outDir, pkgName, rawPkgAlias, rawPkgPath string,
	fc *frameworkContext,
	takenNames map[string]bool,
) error {
	framework := fc.framework
	// Index the framework's exported, available named enums by Go type name,
	// derived exactly like the raw emitter (naming.GoTypeName on the enum key).
	enumsByGoType := make(map[string]meta.Enum)
	// goTypeToKey remembers the ObjC/C metadata key each Go type name came from,
	// so a recorded parity entry can be keyed identically to the raw oracle's.
	goTypeToKey := make(map[string]string)
	// namedMemberNames is every member name a named (tag-bearing) enum owns, so
	// the anonymous-enum pass can skip members a named enum already covers —
	// matching the raw layer, which never emits a constant twice.
	namedMemberNames := make(map[string]bool)
	var anonEnums []meta.Enum
	for key, enum := range framework.Enums {
		if enum.Availability.IsUnavailable {
			continue
		}
		if enum.IsAnon {
			anonEnums = append(anonEnums, enum)
			continue
		}
		for _, member := range enum.Members {
			namedMemberNames[member.Name] = true
		}
		goType := naming.GoTypeName(key)
		if !isExportedGoIdent(goType) {
			continue
		}
		enumsByGoType[goType] = enum
		goTypeToKey[goType] = key
	}
	if len(enumsByGoType) == 0 && len(anonEnums) == 0 {
		return nil
	}

	enumsFile := pkgName + "_enums_generated.go"

	// Emit every one of the framework's own named enums (matching the raw layer's
	// coverage), not just the ones a generated signature referenced. Enums a
	// signature localized are emitted first so they keep their preferred (shared)
	// spelling; the remainder follow in sorted order and take a fallback spelling
	// only on an actual name collision. This ordering keeps signature-referenced
	// enums byte-stable while adding the rest.
	goTypes := make([]string, 0, len(enumsByGoType))
	for goType := range enumsByGoType {
		goTypes = append(goTypes, goType)
	}
	sort.Strings(goTypes)
	sort.SliceStable(goTypes, func(i, j int) bool {
		ri, rj := fc.referenced[goTypes[i]], fc.referenced[goTypes[j]]
		if ri != rj {
			return ri // referenced enums first
		}
		return false // otherwise keep the sorted order
	})

	// Type-name collisions dedup against emittedTypeNames (a fresh set — the enum
	// type names were pre-reserved in takenNames for the enums' own use, so
	// takenNames must not block them). Member const-name collisions are resolved
	// against usedConstNames, seeded from every other claimed package-level name.
	emittedTypeNames := map[string]bool{}
	usedConstNames := map[string]bool{}
	for name := range takenNames {
		usedConstNames[name] = true
	}

	var enums []view.Enum
	needsFmt, needsStrings := false, false
	for _, goType := range goTypes {
		enum := enumsByGoType[goType]
		// Resolve the type name, falling back to the raw (un-deprefixed) spelling
		// and then a numeric suffix if the preferred local spelling is taken.
		localName := fc.localEnumTypeName(goType)
		if emittedTypeNames[localName] {
			localName = naming.GoTypeName(goTypeToKey[goType])
			for emittedTypeNames[localName] {
				localName = uniquifyName(localName, emittedTypeNames)
			}
		}
		v := buildEnumView(localName, enum, fc.prefix, usedConstNames)
		if len(v.Members) == 0 {
			continue
		}
		emittedTypeNames[localName] = true
		usedConstNames[localName] = true
		recordIdiomaticEnum(fc.manifest, framework.Framework, pkgName, goTypeToKey[goType], localName, enum)
		if v.IsBitmask {
			needsStrings = true
		} else {
			needsFmt = true
		}
		enums = append(enums, v)
	}

	// Anonymous (tag-less) enums: their members are emitted as one untyped const
	// block, dropping any a named enum already covers and any whose name is not
	// exportable, with the same first-wins collision resolution.
	anonMembers := buildAnonConstMembers(anonEnums, framework.Framework, pkgName, fc.prefix, namedMemberNames, usedConstNames, fc.manifest)

	if len(enums) == 0 && len(anonMembers) == 0 {
		return nil
	}

	body, err := render.Enums(enums)
	if err != nil {
		return err
	}
	if len(anonMembers) > 0 {
		anonBody, err := render.AnonConsts(anonMembers)
		if err != nil {
			return err
		}
		body = append(body, anonBody...)
	}

	imports := map[string]string{}
	if needsFmt {
		imports["fmt"] = "fmt"
	}
	if needsStrings {
		imports["strings"] = "strings"
	}

	return rawfw.WriteGoFile(filepath.Join(outDir, enumsFile), assembleFile(pkgName, imports, body))
}

// buildAnonConstMembers resolves the members of the framework's anonymous enums
// into untyped constants, skipping members a named enum already emits
// (namedMemberNames), members whose exported spelling is empty, and exact
// duplicates. Collisions are resolved against usedConstNames (first spelling
// wins, then the raw spelling, then a numeric suffix). Each emitted member is
// recorded in the parity manifest keyed on its ObjC/C name.
func buildAnonConstMembers(
	anonEnums []meta.Enum,
	framework, pkgName, prefix string,
	namedMemberNames, usedConstNames map[string]bool,
	rec *emitmanifest.Recorder,
) []view.EnumMember {
	// Deterministic order: sort members by name across all anon enums.
	seenName := make(map[string]bool)
	var cands []meta.EnumMember
	for _, enum := range anonEnums {
		for _, member := range enum.Members {
			if member.Availability.IsUnavailable || namedMemberNames[member.Name] || seenName[member.Name] {
				continue
			}
			seenName[member.Name] = true
			cands = append(cands, member)
		}
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].Name < cands[j].Name })

	var out []view.EnumMember
	for _, member := range cands {
		preferred := deprefixEnumName(naming.GoTypeName(member.Name), prefix)
		if !isExportedGoIdent(preferred) {
			continue // an unexported constant name is not usable by callers
		}
		constName := preferred
		if usedConstNames[constName] {
			if raw := naming.GoTypeName(member.Name); isExportedGoIdent(raw) && !usedConstNames[raw] {
				constName = raw
			} else {
				constName = uniquifyName(constName, usedConstNames)
			}
		}
		usedConstNames[constName] = true
		rec.Record(emitmanifest.Entry{
			Style:     emitmanifest.StyleIdiomatic,
			Kind:      emitmanifest.KindEnumMember,
			Framework: framework,
			MetaKey:   emitmanifest.MetaKey(framework, emitmanifest.KindEnumMember, member.Name, ""),
			GoPkg:     pkgName,
			GoSymbol:  constName,
		})
		out = append(out, view.EnumMember{
			ConstName:    constName,
			Value:        member.Value,
			CommentBlock: renderEnumComment(member.Doc, member.Availability, "\t"),
		})
	}
	return out
}

// uniquifyName appends the smallest numeric suffix that makes name unused. It is
// the last-resort disambiguator when neither the preferred idiomatic spelling nor
// the raw spelling is free.
func uniquifyName(name string, used map[string]bool) string {
	for i := 2; ; i++ {
		candidate := name + strconv.Itoa(i)
		if !used[candidate] {
			return candidate
		}
	}
}

// buildEnumView populates the template view-model from raw metadata, matching
// the raw emitter's decisions: underlying type via rawfw.MapEnumGoType +
// UpgradeEnumTypeIfOverflow, names via naming.GoTypeName, (name,value) dedup for
// the const block, and value dedup for the String() switch.
//
// usedConstNames is the package-wide set of already-claimed identifiers. Emitting
// every enum (not just the referenced few) puts many more constants in one
// namespace, so a member whose preferred idiomatic spelling is taken falls back
// to its raw (un-deprefixed) spelling, then to a numeric suffix — first spelling
// wins. Each emitted member's final name is added to usedConstNames.
func buildEnumView(goName string, enum meta.Enum, prefix string, usedConstNames map[string]bool) view.Enum {
	goType := enum.GoType
	if goType == "" {
		goType = "int64"
	}
	goType = rawfw.MapEnumGoType(goType)
	goType = rawfw.UpgradeEnumTypeIfOverflow(goType, enum.Members)

	// SCREAMING_SNAKE C constants get idiomatic const names prefixed with the
	// enum's own Go type name (HV_EXIT_REASON_CANCELED → ExitReasonCanceled);
	// ObjC-style members keep the class-prefix strip.
	cMemberNames := cEnumMemberNames(goName, enum.Members)

	type nv struct{ name, value string }
	seen := map[nv]bool{}
	var members []view.EnumMember
	for _, member := range enum.Members {
		if member.Availability.IsUnavailable {
			continue
		}
		preferred := deprefixEnumName(naming.GoTypeName(member.Name), prefix)
		if cName, isC := cMemberNames[member.Name]; isC {
			preferred = cName
		}
		// Drop an alias/duplicate: the same preferred spelling with the same value
		// already emitted in this enum (matches the raw layer's per-enum dedup).
		if seen[nv{preferred, member.Value}] {
			continue
		}
		seen[nv{preferred, member.Value}] = true
		// Resolve a package-level name collision (another enum's member, a type, a
		// function, or an earlier member here with a different value): keep the
		// preferred spelling when free, else the raw (un-deprefixed) spelling, else
		// a numeric suffix. First spelling wins.
		constName := preferred
		if usedConstNames[constName] {
			if raw := naming.GoTypeName(member.Name); !usedConstNames[raw] {
				constName = raw
			} else {
				constName = uniquifyName(constName, usedConstNames)
			}
		}
		usedConstNames[constName] = true
		members = append(members, view.EnumMember{
			ConstName:    constName,
			Value:        member.Value,
			CommentBlock: renderEnumComment(member.Doc, member.Availability, "\t"),
			IsZeroVal:    member.Value == "0",
		})
	}

	seenVal := map[string]bool{}
	var unique []view.EnumMember
	for _, member := range members {
		if seenVal[member.Value] {
			continue
		}
		seenVal[member.Value] = true
		unique = append(unique, member)
	}

	return view.Enum{
		GoName:        goName,
		GoType:        goType,
		IsBitmask:     enum.IsBitmask,
		CommentBlock:  renderEnumComment(enum.Doc, enum.Availability, ""),
		Members:       members,
		UniqueMembers: unique,
		DefaultFmt:    goName + "(%d)",
	}
}

// renderEnumComment renders the doc + deprecation comment block for an enum or
// member. prefix is "\t" inside the const block, "" at top level.
func renderEnumComment(doc string, avail meta.Availability, prefix string) string {
	var sb strings.Builder
	if doc = cleanDoc(doc); doc != "" {
		for _, line := range strings.Split(strings.TrimRight(doc, "\n"), "\n") {
			fmt.Fprintf(&sb, "%s// %s\n", prefix, line)
		}
	}
	if avail.MacOSDeprecated != "" {
		if doc != "" {
			fmt.Fprintf(&sb, "%s//\n", prefix)
		}
		if avail.DeprecationMsg != "" {
			fmt.Fprintf(&sb, "%s// Deprecated: %s\n", prefix, avail.DeprecationMsg)
		} else {
			fmt.Fprintf(&sb, "%s// Deprecated: since macOS %s.\n", prefix, avail.MacOSDeprecated)
		}
	}
	return sb.String()
}

// isExportedGoIdent reports whether name's first character is an upper-case
// ASCII letter (exported when emitted).
func isExportedGoIdent(name string) bool {
	if name == "" {
		return false
	}
	c := name[0]
	return c >= 'A' && c <= 'Z'
}
