//go:build darwin

package idiomatic

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// enumMemberView is template data for one enum constant.
type enumMemberView struct {
	ConstName    string
	Value        string
	CommentBlock string
	IsZeroVal    bool
}

// enumView is template data for one concrete idiomatic enum (templates/enum.tmpl).
type enumView struct {
	GoName        string
	GoType        string
	IsBitmask     bool
	CommentBlock  string
	Members       []enumMemberView // const block (deduped by name+value)
	UniqueMembers []enumMemberView // String() switch (deduped by value)
	DefaultFmt    string
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
	fw *meta.FrameworkMeta,
	takenNames map[string]bool,
) error {
	// Index the framework's exported, available named enums by Go type name,
	// derived exactly like the raw emitter (naming.GoTypeName on the enum key).
	enumsByGoType := make(map[string]meta.Enum)
	for key, e := range fw.Enums {
		if e.Availability.IsUnavailable || e.IsAnon {
			continue
		}
		goType := naming.GoTypeName(key)
		if !isExportedGoIdent(goType) {
			continue
		}
		enumsByGoType[goType] = e
	}
	if len(enumsByGoType) == 0 {
		return nil
	}

	// The enums the generated signatures localized — and therefore need a
	// concrete local definition — were accumulated into the per-framework
	// referenced set during the build passes (see localizeEnumType). emitEnums
	// runs after those passes, so the set is complete here.
	referenced := referencedEnums(fw)
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

	var body bytes.Buffer
	needsFmt, needsStrings := false, false
	for _, goType := range refNames {
		if takenNames[goType] {
			continue
		}
		view := buildEnumView(goType, enumsByGoType[goType])
		if len(view.Members) == 0 {
			continue
		}
		takenNames[goType] = true
		if view.IsBitmask {
			needsStrings = true
		} else {
			needsFmt = true
		}
		if err := executeTemplate(&body, "named_enum", view); err != nil {
			return err
		}
	}
	if body.Len() == 0 {
		return nil
	}

	imports := map[string]string{}
	if needsFmt {
		imports["fmt"] = "fmt"
	}
	if needsStrings {
		imports["strings"] = "strings"
	}

	return emit.WriteGoFile(filepath.Join(outDir, enumsFile), assembleFile(pkgName, imports, body.Bytes()))
}

// buildEnumView populates the template view-model from raw metadata, matching
// the raw emitter's decisions: underlying type via emit.MapEnumGoType +
// UpgradeEnumTypeIfOverflow, names via naming.GoTypeName, (name,value) dedup for
// the const block, and value dedup for the String() switch.
func buildEnumView(goName string, e meta.Enum) enumView {
	goType := e.GoType
	if goType == "" {
		goType = "int64"
	}
	goType = emit.MapEnumGoType(goType)
	goType = emit.UpgradeEnumTypeIfOverflow(goType, e.Members)

	type nv struct{ name, value string }
	seen := map[nv]bool{}
	var members []enumMemberView
	for _, m := range e.Members {
		if m.Availability.IsUnavailable {
			continue
		}
		constName := naming.GoTypeName(m.Name)
		k := nv{constName, m.Value}
		if seen[k] {
			continue
		}
		seen[k] = true
		members = append(members, enumMemberView{
			ConstName:    constName,
			Value:        m.Value,
			CommentBlock: renderEnumComment(m.Doc, m.Availability, "\t"),
			IsZeroVal:    m.Value == "0",
		})
	}

	seenVal := map[string]bool{}
	var unique []enumMemberView
	for _, m := range members {
		if seenVal[m.Value] {
			continue
		}
		seenVal[m.Value] = true
		unique = append(unique, m)
	}

	return enumView{
		GoName:        goName,
		GoType:        goType,
		IsBitmask:     e.IsBitmask,
		CommentBlock:  renderEnumComment(e.Doc, e.Availability, ""),
		Members:       members,
		UniqueMembers: unique,
		DefaultFmt:    goName + "(%d)",
	}
}

// renderEnumComment renders the doc + deprecation comment block for an enum or
// member. prefix is "\t" inside the const block, "" at top level.
func renderEnumComment(doc string, avail meta.Availability, prefix string) string {
	var sb strings.Builder
	if doc != "" {
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
