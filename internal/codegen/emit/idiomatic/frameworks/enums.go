//go:build darwin

package idiofw

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/idiomatic/frameworks/view"
	rawfw "github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/frameworks"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

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
	for key, enum := range framework.Enums {
		if enum.Availability.IsUnavailable || enum.IsAnon {
			continue
		}
		goType := naming.GoTypeName(key)
		if !isExportedGoIdent(goType) {
			continue
		}
		enumsByGoType[goType] = enum
	}
	if len(enumsByGoType) == 0 {
		return nil
	}

	// The enums the generated signatures localized — and therefore need a
	// concrete local definition — were accumulated into the per-framework
	// referenced set during the build passes (see localizeEnumType). emitEnums
	// runs after those passes, so the set is complete here.
	referenced := fc.referenced
	// Also surface error-code enums (e.g. VZErrorCode) even when no signature
	// references them: callers need their constants for errors.As / error-domain
	// code comparison, but errors cross the boundary as Go error values rather
	// than typed returns, so the referenced-only scan never picks them up.
	for goType := range enumsByGoType {
		if strings.HasSuffix(goType, "ErrorCode") {
			referenced[goType] = true
		}
	}
	if len(referenced) == 0 {
		return nil
	}
	enumsFile := pkgName + "_enums_generated.go"

	refNames := make([]string, 0, len(referenced))
	for goType := range referenced {
		if _, ok := enumsByGoType[goType]; ok {
			refNames = append(refNames, goType)
		}
	}
	sort.Strings(refNames)

	// Enum type names were reserved up front (see EmitFrameworkWrappers), so this
	// pass tracks the names it actually defines locally to avoid emitting two
	// definitions when two enums shorten to the same name.
	emitted := map[string]bool{}
	var enums []view.Enum
	needsFmt, needsStrings := false, false
	for _, goType := range refNames {
		// Use the shared local spelling, matching localizeEnumType so signatures
		// and this definition agree.
		localName := fc.localEnumTypeName(goType)
		if emitted[localName] {
			continue
		}
		v := buildEnumView(localName, enumsByGoType[goType], fc.prefix)
		if len(v.Members) == 0 {
			continue
		}
		emitted[localName] = true
		if v.IsBitmask {
			needsStrings = true
		} else {
			needsFmt = true
		}
		enums = append(enums, v)
	}
	if len(enums) == 0 {
		return nil
	}

	body, err := render.Enums(enums)
	if err != nil {
		return err
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

// buildEnumView populates the template view-model from raw metadata, matching
// the raw emitter's decisions: underlying type via rawfw.MapEnumGoType +
// UpgradeEnumTypeIfOverflow, names via naming.GoTypeName, (name,value) dedup for
// the const block, and value dedup for the String() switch.
func buildEnumView(goName string, enum meta.Enum, prefix string) view.Enum {
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
		constName := deprefixEnumName(naming.GoTypeName(member.Name), prefix)
		if cName, isC := cMemberNames[member.Name]; isC {
			constName = cName
		}
		k := nv{constName, member.Value}
		if seen[k] {
			continue
		}
		seen[k] = true
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
