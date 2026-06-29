package raw

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// Enums writes all enum types and their constants to w.
func EmitEnums(w io.Writer, framework *macosplatformmetadata.FrameworkMeta) error {
	names := make([]string, 0, len(framework.Enums))
	for k := range framework.Enums {
		names = append(names, k)
	}
	sort.Strings(names)

	skipAnon := anonEnumsCoveredByNamed(framework.Enums)

	for _, name := range names {
		if skipAnon[name] {
			continue
		}
		e := framework.Enums[name]
		if err := writeEnum(w, name, e); err != nil {
			return err
		}
	}
	return nil
}

// writeEnum dispatches to the appropriate template depending on whether the
// enum is anonymous (const-only) or named (type + const + helper methods).
func writeEnum(w io.Writer, name string, e macosplatformmetadata.Enum) error {
	if e.IsAnon {
		return writeAnonEnum(w, e)
	}
	return executeTemplate(w, "named_enum", buildEnumModel(name, e))
}

// buildEnumModel populates an enumModel from raw metadata.
// All string-formatting decisions (comment rendering, name conversion, dedup)
// live here so the template itself stays a pure structural description of the output.
func buildEnumModel(name string, e macosplatformmetadata.Enum) enumModel {
	goName := naming.GoTypeName(name)

	// Deduplicate members for the String() switch: skip constants whose Go name
	// or formatted value has already appeared (ObjC allows aliased values).
	seenNames := make(map[string]bool)
	seenValues := make(map[string]bool)
	var uniqueMembers []enumMemberModel

	members := make([]enumMemberModel, 0, len(e.Members))
	for _, m := range e.Members {
		constName := naming.GoTypeName(m.Name)
		val := formatValue(m.Value)

		mm := enumMemberModel{
			ConstName:    constName,
			Value:        val,
			ObjCName:     m.Name,
			CommentBlock: renderCommentBlock(m.Doc, "", 0, m.Availability, "\t"),
			IsZeroVal:    val == "0",
		}
		members = append(members, mm)

		if !seenNames[constName] && !seenValues[val] {
			seenNames[constName] = true
			seenValues[val] = true
			uniqueMembers = append(uniqueMembers, mm)
		}
	}

	firstConst := ""
	if len(members) > 0 {
		firstConst = members[0].ConstName
	}

	return enumModel{
		GoName:        goName,
		GoType:        e.GoType,
		IsBitmask:     e.IsBitmask,
		HasMembers:    len(e.Members) > 0,
		CommentBlock:  renderCommentBlock(e.Doc, e.SDKFile, e.SDKLine, e.Availability, ""),
		Members:       members,
		UniqueMembers: uniqueMembers,
		FirstConst:    firstConst,
		DefaultFmt:    goName + "(%d)",
	}
}

// writeAnonEnum writes a const block for an anonymous ObjC enum (no type declaration).
func writeAnonEnum(w io.Writer, e macosplatformmetadata.Enum) error {
	if len(e.Members) == 0 {
		return nil
	}
	members := make([]enumMemberModel, len(e.Members))
	for i, m := range e.Members {
		members[i] = enumMemberModel{
			ConstName: naming.GoTypeName(m.Name),
			Value:     formatValue(m.Value),
		}
	}
	return executeTemplate(w, "anon_enum", members)
}

// anonEnumsCoveredByNamed identifies "_anon_*" enum entries whose member names
// are entirely covered by a separate named enum in the same framework. Clang
// represents `typedef enum { A, B } Name;` as two AST siblings (anonymous
// EnumDecl + TypedefDecl); the scanner records both, producing an "_anon_A"
// duplicate of the typedef-named entry. Emitting both yields "redeclared in
// this block" errors at file scope.
//
// The dedup is name-only — C/ObjC forbid two enum declarations from sharing
// constant names within a translation unit, so identical member-name sets are
// a reliable signal that both metadata entries describe the same underlying
// declaration. The named (typedef-promoted) form is authoritative. Genuine
// standalone anonymous enums (no typedef twin) are unaffected and retain
// their correctly auto-incremented values from extractAnonymousEnum.
func anonEnumsCoveredByNamed(enums map[string]macosplatformmetadata.Enum) map[string]bool {
	namedMemberSets := make([]map[string]bool, 0, len(enums))
	for name, e := range enums {
		if strings.HasPrefix(name, "_anon_") {
			continue
		}
		members := make(map[string]bool, len(e.Members))
		for _, m := range e.Members {
			members[m.Name] = true
		}
		namedMemberSets = append(namedMemberSets, members)
	}

	skip := make(map[string]bool)
	for name, e := range enums {
		if !strings.HasPrefix(name, "_anon_") || len(e.Members) == 0 {
			continue
		}
		for _, ns := range namedMemberSets {
			covered := true
			for _, m := range e.Members {
				if !ns[m.Name] {
					covered = false
					break
				}
			}
			if covered {
				skip[name] = true
				break
			}
		}
	}
	return skip
}

// EnumsNeedImports reports which stdlib imports the enum file will need.
func EnumsNeedImports(framework *macosplatformmetadata.FrameworkMeta) (needsFmt, needsStrings bool) {
	for _, e := range framework.Enums {
		if e.IsAnon || len(e.Members) == 0 {
			continue
		}
		if e.IsBitmask {
			needsStrings = true
		} else {
			needsFmt = true
		}
	}
	return
}

func formatValue(v string) string {
	return v
}

func availabilityComment(av macosplatformmetadata.Availability) string {
	if av.MacOSIntroduced != "" {
		return fmt.Sprintf("Introduced: macOS %s", av.MacOSIntroduced)
	}
	return ""
}

// deprecationComment returns the "Deprecated: ..." text in the format recognised
// by the Go toolchain (Go 1.21+) and gopls. The paragraph must start exactly
// with "Deprecated:" so that IDEs show strike-through on call sites.
func deprecationComment(av macosplatformmetadata.Availability) string {
	if av.MacOSDeprecated == "" {
		return ""
	}
	var parts []string
	intro := "Deprecated in macOS " + av.MacOSDeprecated
	if av.MacOSObsoleted != "" {
		intro += ", removed in macOS " + av.MacOSObsoleted
	}
	parts = append(parts, intro+".")
	if av.DeprecationMsg != "" {
		parts = append(parts, av.DeprecationMsg)
	}
	if av.ReplacedBy != "" {
		parts = append(parts, "Use "+av.ReplacedBy+" instead.")
	}
	return "Deprecated: " + strings.Join(parts, " ")
}

// writeContextComments emits optional doc text, SDK location, availability, and
// deprecation lines. prefix is "\t" inside var blocks, "" at top-level.
func writeContextComments(w io.Writer, doc, sdkFile string, sdkLine int, av macosplatformmetadata.Availability, prefix string) {
	if doc != "" {
		for _, line := range strings.Split(doc, "\n") {
			fmt.Fprintf(w, "%s// %s\n", prefix, line)
		}
	}
	if sdkFile != "" {
		if sdkLine > 0 {
			fmt.Fprintf(w, "%s// [%s:%d]\n", prefix, sdkFile, sdkLine)
		} else {
			fmt.Fprintf(w, "%s// [%s]\n", prefix, sdkFile)
		}
	}
	if c := availabilityComment(av); c != "" {
		fmt.Fprintf(w, "%s// %s\n", prefix, c)
	}
	if d := deprecationComment(av); d != "" {
		// Blank comment line separates the deprecation into its own paragraph so
		// gopls and go vet recognise the "Deprecated:" marker for IDE warnings.
		fmt.Fprintf(w, "%s//\n", prefix)
		fmt.Fprintf(w, "%s// %s\n", prefix, d)
	}
	for _, k := range av.Entitlements {
		fmt.Fprintf(w, "%s// Requires entitlement: %s\n", prefix, k)
	}
}

// renderCommentBlock returns the pre-rendered comment block string for use in
// template models. It calls writeContextComments and captures the output.
// Returns "" when there are no comments to emit.
func renderCommentBlock(doc, sdkFile string, sdkLine int, av macosplatformmetadata.Availability, prefix string) string {
	var sb strings.Builder
	writeContextComments(&sb, doc, sdkFile, sdkLine, av, prefix)
	return sb.String()
}
