package emit

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/view"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// EmitEnums writes all enum declarations for the framework to w. It gathers each
// enum into a resolved view and renders the whole set through the enum template.
func EmitEnums(w io.Writer, framework *meta.FrameworkMeta) error {
	names := sortedEnumNames(framework.Enums)
	if len(names) == 0 {
		return nil
	}

	// Track which named enums cover which anon members so we can skip
	// pure-duplicate anon enums.
	namedMemberNames := make(map[string]bool)
	for _, name := range names {
		enum := framework.Enums[name]
		if !enum.IsAnon {
			for _, member := range enum.Members {
				namedMemberNames[member.Name] = true
			}
		}
	}

	var enums []view.Enum
	for _, name := range names {
		enum := framework.Enums[name]
		if enum.Availability.IsUnavailable {
			continue
		}
		if enum.IsAnon {
			if built, ok := buildAnonEnumView(enum, namedMemberNames); ok {
				enums = append(enums, built)
			}
		} else {
			enums = append(enums, buildNamedEnumView(name, enum))
		}
	}
	if len(enums) == 0 {
		return nil
	}

	out, err := render.Enums(enums)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

// buildNamedEnumView resolves a named enum into its renderable view: the Go type
// name and underlying integer type, the comment block, the deduplicated typed
// constants, and the members the String method dispatches on.
func buildNamedEnumView(name string, enum meta.Enum) view.Enum {
	goName := naming.GoTypeName(name)
	goType := enum.GoType
	if goType == "" {
		goType = "int64"
	}
	goType = mapEnumGoType(goType)
	goType = upgradeEnumTypeIfOverflow(goType, enum.Members)

	built := view.Enum{
		GoName:       goName,
		GoType:       goType,
		CommentBlock: enumCommentBlock(enum),
		IsBitmask:    enum.IsBitmask,
	}

	// Deduplicate members by name and value.
	type memberKey struct{ name, value string }
	seen := make(map[memberKey]bool)
	var unique []meta.EnumMember
	for _, member := range enum.Members {
		key := memberKey{member.Name, member.Value}
		if !seen[key] {
			seen[key] = true
			unique = append(unique, member)
		}
	}
	built.HasConstBlock = len(unique) > 0

	for _, member := range unique {
		if member.Availability.IsUnavailable {
			continue
		}
		built.Members = append(built.Members, view.EnumMember{
			ConstName:    naming.GoTypeName(member.Name),
			Value:        member.Value,
			CommentBlock: memberCommentBlock(member),
		})
	}

	if enum.IsBitmask {
		for _, member := range unique {
			if member.Availability.IsUnavailable || member.Value == "0" {
				continue
			}
			built.StringMembers = append(built.StringMembers, view.EnumMember{ConstName: naming.GoTypeName(member.Name)})
		}
	} else {
		emittedValues := make(map[string]bool)
		for _, member := range unique {
			if member.Availability.IsUnavailable || emittedValues[member.Value] {
				continue
			}
			emittedValues[member.Value] = true
			built.StringMembers = append(built.StringMembers, view.EnumMember{ConstName: naming.GoTypeName(member.Name)})
		}
	}
	return built
}

// buildAnonEnumView resolves an anonymous enum into an untyped const block,
// dropping members already covered by a named enum. It reports false when no
// members remain to emit.
func buildAnonEnumView(enum meta.Enum, namedMemberNames map[string]bool) (view.Enum, bool) {
	allCovered := true
	for _, member := range enum.Members {
		if !namedMemberNames[member.Name] {
			allCovered = false
			break
		}
	}
	if allCovered {
		return view.Enum{}, false
	}

	var members []meta.EnumMember
	for _, member := range enum.Members {
		if !namedMemberNames[member.Name] && !member.Availability.IsUnavailable {
			members = append(members, member)
		}
	}
	if len(members) == 0 {
		return view.Enum{}, false
	}

	sort.Slice(members, func(i, j int) bool {
		return members[i].Name < members[j].Name
	})

	built := view.Enum{IsAnon: true}
	for _, member := range members {
		built.Members = append(built.Members, view.EnumMember{
			ConstName: naming.GoTypeName(member.Name),
			Value:     member.Value,
		})
	}
	return built, true
}

// enumCommentBlock renders the doc + deprecation comment for an enum type, or ""
// when undocumented and not deprecated.
func enumCommentBlock(enum meta.Enum) string {
	var sb strings.Builder
	if enum.Doc != "" {
		fmt.Fprintf(&sb, "// %s\n", enum.Doc)
	}
	if enum.Availability.MacOSDeprecated != "" {
		if enum.Availability.DeprecationMsg != "" {
			fmt.Fprintf(&sb, "// Deprecated: %s\n", enum.Availability.DeprecationMsg)
		} else {
			fmt.Fprintf(&sb, "// Deprecated: since macOS %s.\n", enum.Availability.MacOSDeprecated)
		}
	}
	return sb.String()
}

// memberCommentBlock renders a member's doc comment, or "" when undocumented.
// The leading tab matches the original string-builder's output: gofmt's
// doc-comment reformatter rewrites a column-0 comment (smart-quoting “…“) but
// leaves an already-indented comment verbatim, so the tab keeps the output
// byte-identical.
func memberCommentBlock(member meta.EnumMember) string {
	if member.Doc != "" {
		return "\t// " + member.Doc + "\n"
	}
	return ""
}

// upgradeEnumTypeIfOverflow upgrades a signed int type to unsigned if any
// member value would overflow a signed integer (e.g. very large bitmask values).
func upgradeEnumTypeIfOverflow(goType string, members []meta.EnumMember) string {
	var signed, bitWidth int
	switch goType {
	case "int":
		signed, bitWidth = 1, 64
	case "int64":
		signed, bitWidth = 1, 64
	case "int32":
		signed, bitWidth = 1, 32
	case "int16":
		signed, bitWidth = 1, 16
	case "int8":
		signed, bitWidth = 1, 8
	default:
		return goType
	}
	_ = bitWidth
	if signed == 0 {
		return goType
	}
	for _, m := range members {
		v := strings.TrimSpace(m.Value)
		if strings.HasPrefix(v, "-") {
			continue // negative — fine for signed
		}
		// Check if value string represents a number larger than int64 max.
		// Numbers >= 2^63 are represented as unsigned hex or large decimals.
		if len(v) >= 19 && !strings.HasPrefix(v, "0x") {
			// Decimal >= 10^18 likely overflows int64
			return unsignedVariant(goType)
		}
		if strings.HasPrefix(v, "0x") || strings.HasPrefix(v, "0X") {
			// Hex value: if more than 16 hex digits or starts with 8-f in high nibble
			hex := v[2:]
			if len(hex) > 16 {
				return unsignedVariant(goType)
			}
			if len(hex) == 16 && hex[0] >= '8' {
				return unsignedVariant(goType)
			}
		}
	}
	return goType
}

func unsignedVariant(signed string) string {
	switch signed {
	case "int":
		return "uint"
	case "int64":
		return "uint64"
	case "int32":
		return "uint32"
	case "int16":
		return "uint16"
	case "int8":
		return "uint8"
	}
	return signed
}

// MapEnumGoType maps an ObjC/C integer type string to the Go integer type used
// as an enum's underlying type. Exported so the idiomatic emitter can derive the
// exact same underlying type for its re-emitted concrete enums.
func MapEnumGoType(t string) string { return mapEnumGoType(t) }

// UpgradeEnumTypeIfOverflow upgrades a signed Go integer type to its unsigned
// variant when a member value would overflow it. Exported for reuse alongside
// MapEnumGoType.
func UpgradeEnumTypeIfOverflow(goType string, members []meta.EnumMember) string {
	return upgradeEnumTypeIfOverflow(goType, members)
}

// mapEnumGoType maps ObjC/C integer type strings to Go integer types.
func mapEnumGoType(t string) string {
	switch strings.TrimSpace(t) {
	case "int", "int32_t", "signed int":
		return "int32"
	case "unsigned int", "uint32_t":
		return "uint32"
	case "long", "int64_t", "NSInteger":
		return "int"
	case "unsigned long", "uint64_t", "NSUInteger":
		return "uint"
	case "long long":
		return "int64"
	case "unsigned long long":
		return "uint64"
	case "short", "int16_t":
		return "int16"
	case "unsigned short", "uint16_t":
		return "uint16"
	case "char", "int8_t":
		return "int8"
	case "unsigned char", "uint8_t":
		return "uint8"
	default:
		return t
	}
}
