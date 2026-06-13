package emit

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// EmitEnums writes all enum declarations for the framework to w.
func EmitEnums(w io.Writer, framework *meta.FrameworkMeta) error {
	names := sortedEnumNames(framework.Enums)
	if len(names) == 0 {
		return nil
	}

	// Track which named enums cover which anon members so we can skip
	// pure-duplicate anon enums.
	namedMemberNames := make(map[string]bool)
	for _, name := range names {
		e := framework.Enums[name]
		if !e.IsAnon {
			for _, m := range e.Members {
				namedMemberNames[m.Name] = true
			}
		}
	}

	for _, name := range names {
		e := framework.Enums[name]
		if e.Availability.IsUnavailable {
			continue
		}
		if e.IsAnon {
			if err := emitAnonEnum(w, e, namedMemberNames); err != nil {
				return err
			}
		} else {
			if err := emitNamedEnum(w, name, e); err != nil {
				return err
			}
		}
	}
	return nil
}

func emitNamedEnum(w io.Writer, name string, e meta.Enum) error {
	goName := naming.GoTypeName(name)
	goType := e.GoType
	if goType == "" {
		goType = "int64"
	}

	// Map ObjC integer types to Go types.
	goType = mapEnumGoType(goType)

	// Upgrade signed integer type to unsigned if any member value overflows.
	goType = upgradeEnumTypeIfOverflow(goType, e.Members)

	if e.Doc != "" {
		fmt.Fprintf(w, "// %s\n", e.Doc)
	}
	emitDeprecatedComment(w, e.Availability)
	fmt.Fprintf(w, "type %s %s\n\n", goName, goType)

	// Deduplicate members by name and value.
	type memberKey struct{ name, value string }
	seen := make(map[memberKey]bool)
	var unique []meta.EnumMember
	for _, m := range e.Members {
		k := memberKey{m.Name, m.Value}
		if !seen[k] {
			seen[k] = true
			unique = append(unique, m)
		}
	}

	if len(unique) > 0 {
		fmt.Fprintf(w, "const (\n")
		for _, m := range unique {
			if m.Availability.IsUnavailable {
				continue
			}
			constName := naming.GoTypeName(m.Name)
			if m.Doc != "" {
				fmt.Fprintf(w, "\t// %s\n", m.Doc)
			}
			fmt.Fprintf(w, "\t%s %s = %s\n", constName, goName, m.Value)
		}
		fmt.Fprintf(w, ")\n\n")
	}

	// String() method
	fmt.Fprintf(w, "func (e %s) String() string {\n", goName)
	if e.IsBitmask {
		fmt.Fprintf(w, "\tvar parts []string\n")
		for _, m := range unique {
			if m.Availability.IsUnavailable || m.Value == "0" {
				continue
			}
			constName := naming.GoTypeName(m.Name)
			fmt.Fprintf(w, "\tif e&%s != 0 { parts = append(parts, %q) }\n", constName, constName)
		}
		fmt.Fprintf(w, "\tif len(parts) == 0 { return %q }\n", "0")
		fmt.Fprintf(w, "\treturn strings.Join(parts, \"|\")\n")
		fmt.Fprintf(w, "}\n\n")
	} else {
		fmt.Fprintf(w, "\tswitch e {\n")
		emittedValues := make(map[string]bool)
		for _, m := range unique {
			if m.Availability.IsUnavailable {
				continue
			}
			if emittedValues[m.Value] {
				continue
			}
			emittedValues[m.Value] = true
			constName := naming.GoTypeName(m.Name)
			fmt.Fprintf(w, "\tcase %s:\n\t\treturn %q\n", constName, constName)
		}
		fmt.Fprintf(w, "\tdefault:\n\t\treturn fmt.Sprintf(\"%s(%%d)\", int64(e))\n", goName)
		fmt.Fprintf(w, "\t}\n}\n\n")
	}

	return nil
}

func emitAnonEnum(w io.Writer, e meta.Enum, namedMemberNames map[string]bool) error {
	// Skip if all members are already covered by a named enum.
	allCovered := true
	for _, m := range e.Members {
		if !namedMemberNames[m.Name] {
			allCovered = false
			break
		}
	}
	if allCovered {
		return nil
	}

	// Emit as untyped const block.
	var members []meta.EnumMember
	for _, m := range e.Members {
		if !namedMemberNames[m.Name] && !m.Availability.IsUnavailable {
			members = append(members, m)
		}
	}
	if len(members) == 0 {
		return nil
	}

	sort.Slice(members, func(i, j int) bool {
		return members[i].Name < members[j].Name
	})

	fmt.Fprintf(w, "const (\n")
	for _, m := range members {
		constName := naming.GoTypeName(m.Name)
		fmt.Fprintf(w, "\t%s = %s\n", constName, m.Value)
	}
	fmt.Fprintf(w, ")\n\n")
	return nil
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
