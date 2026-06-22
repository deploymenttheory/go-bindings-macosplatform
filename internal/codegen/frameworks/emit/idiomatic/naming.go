//go:build darwin

package idiomatic

import (
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// trialNameMap maps raw ObjC class names to their trial Go type names.
// Used to substitute trial types for raw types in method signatures.
type trialNameMap map[string]string

func buildTrialNameMap(fw *meta.FrameworkMeta, prefix string) trialNameMap {
	m := make(trialNameMap, len(fw.Classes))
	for className := range fw.Classes {
		m[className] = trialTypeName(className, prefix)
	}
	return m
}

// abstractBaseIndex maps ObjC raw class names (that have at least one direct subclass
// in the same framework) to their trial Go type name.
type abstractBaseIndex map[string]string

// buildAbstractBaseIndex returns a map from ObjC class name to trial Go type name for
// every class in fw that has at least one direct subclass also in fw.
func buildAbstractBaseIndex(fw *meta.FrameworkMeta, prefix string) abstractBaseIndex {
	bases := make(abstractBaseIndex)
	for _, cls := range fw.Classes {
		if cls.Availability.IsUnavailable || cls.Super == "" {
			continue
		}
		if superCls, ok := fw.Classes[cls.Super]; ok && !superCls.Availability.IsUnavailable {
			bases[cls.Super] = trialTypeName(cls.Super, prefix)
		}
	}
	return bases
}

// providerInterfaceName returns the provider interface name for a trial Go type, e.g.
// "BootLoader" → "BootLoaderProvider".
func providerInterfaceName(trialGoTypeName string) string {
	return trialGoTypeName + "Provider"
}

// markerMethodName returns the unexported sealing method a base class defines so
// only it and its subclasses satisfy its provider interface, e.g. "BootLoader" →
// "isBootLoader". The method is promoted to every subclass through embedding, so
// a setter typed as the provider accepts only real members of the hierarchy.
func markerMethodName(trialGoTypeName string) string {
	return "is" + trialGoTypeName
}

// adoptHelperName is the unexported helper that wraps an already-+1-owned object
// without an extra retain, e.g. "VirtualMachine" -> "virtualMachineAdopt".
func adoptHelperName(goTypeName string) string {
	return lowerFirst(goTypeName) + "Adopt"
}

// safeParamName renames a parameter that would shadow one of the package
// aliases the generated bodies use (e.g. a parameter literally named "obj"
// would hide the obj package). The trailing underscore keeps it a valid,
// distinct identifier.
func safeParamName(name string) string {
	switch name {
	case "obj", "rt", "errkit", "objref", "purego", "objc", "ebipurego", "unsafe", "context":
		return name + "_"
	}
	return name
}

// safeReturnName returns a named-return identifier that does not collide with the
// receiver, the signature parameters, or an earlier named return.
func safeReturnName(name string, taken map[string]bool) string {
	if name == "" {
		name = "v"
	}
	for taken[name] {
		name += "_"
	}
	return name
}

func detectClassPrefix(fw *meta.FrameworkMeta) string {
	names := sortedKeys(fw.Classes)
	if len(names) == 0 {
		return ""
	}
	prefix := names[0]
	for _, name := range names[1:] {
		i := 0
		for i < len(prefix) && i < len(name) && prefix[i] == name[i] {
			i++
		}
		prefix = prefix[:i]
		if prefix == "" {
			return ""
		}
	}
	trimAt := len(prefix)
	for i, r := range prefix {
		if r >= 'a' && r <= 'z' {
			trimAt = i
			break
		}
	}
	// When the common prefix was cut at a lowercase letter, the preceding
	// uppercase letter starts the first word of the class name proper
	// (e.g. "CHHaptic" → trim at 'a' gives "CHH"; the final 'H' belongs to
	// "Haptic"), so drop it from the prefix.
	if trimAt < len(prefix) && trimAt > 0 {
		trimAt--
	}
	prefix = prefix[:trimAt]
	if len(prefix) < 2 {
		return ""
	}
	return prefix
}

func trialTypeName(className, prefix string) string {
	if prefix != "" && strings.HasPrefix(className, prefix) {
		stripped := className[len(prefix):]
		// A stripped name starting with a digit is an invalid Go identifier,
		// and one starting lowercase would be unexported. Keep the full class
		// name in those cases (e.g. AVB17221ACMPInterface stays as-is).
		if len(stripped) > 0 &&
			(stripped[0] >= '0' && stripped[0] <= '9' || stripped[0] >= 'a' && stripped[0] <= 'z') {
			return className
		}
		return stripped
	}
	return className
}

func asyncGoMethodName(selector string) string {
	base := naming.MethodName(selector)
	if strings.HasSuffix(base, "WithCompletionHandler") {
		return base[:len(base)-len("WithCompletionHandler")]
	}
	if strings.HasSuffix(base, "CompletionHandler") {
		return base[:len(base)-len("CompletionHandler")]
	}
	return base
}

func boolNSErrorGoMethodName(selector string) string {
	base := naming.MethodName(selector)
	if strings.HasSuffix(base, "WithError") {
		return base[:len(base)-len("WithError")]
	}
	return base
}

// deprefixEnumName removes the framework prefix from an enum type or member name
// (VZVirtualMachineState -> VirtualMachineState), so the idiomatic names read
// without the framework's letters repeated. It keeps the original name when
// stripping would leave an invalid or lower-case identifier.
func deprefixEnumName(name, prefix string) string {
	if prefix == "" || !strings.HasPrefix(name, prefix) {
		return name
	}
	stripped := name[len(prefix):]
	if stripped == "" || stripped[0] < 'A' || stripped[0] > 'Z' {
		return name
	}
	return stripped
}

func sortedKeys(m map[string]meta.Class) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
