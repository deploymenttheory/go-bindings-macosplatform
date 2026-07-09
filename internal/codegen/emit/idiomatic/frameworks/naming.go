//go:build darwin

package idiofw

import (
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/naming"
)

// trialNameMap maps raw ObjC class names to their trial Go type names.
// Used to substitute trial types for raw types in method signatures.
type trialNameMap map[string]string

func buildTrialNameMap(framework *meta.FrameworkMeta, prefix string) trialNameMap {
	names := make(trialNameMap, len(framework.Classes))
	for className := range framework.Classes {
		names[className] = trialTypeName(className, prefix)
	}
	return names
}

// abstractBaseIndex maps ObjC raw class names (that have at least one direct subclass
// in the same framework) to their trial Go type name.
type abstractBaseIndex map[string]string

// buildAbstractBaseIndex returns a map from ObjC class name to trial Go type name for
// every class in framework that has at least one direct subclass also in framework.
func buildAbstractBaseIndex(framework *meta.FrameworkMeta, prefix string) abstractBaseIndex {
	bases := make(abstractBaseIndex)
	for _, class := range framework.Classes {
		if class.Availability.IsUnavailable || class.Super == "" {
			continue
		}
		if superCls, ok := framework.Classes[class.Super]; ok &&
			!superCls.Availability.IsUnavailable {
			bases[class.Super] = trialTypeName(class.Super, prefix)
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
// would hide the obj package), and substitutes a readable name for the
// ugliest reserved-word escapes ParamName produces (string_ / bytes_ / len_ /
// id_). A collision with another parameter of the same method is handled by
// the caller's usedParamNames dedup, so a substitution never duplicates a
// name.
func safeParamName(name string) string {
	switch name {
	case "string_":
		return "str"
	case "bytes_":
		return "data"
	case "len_":
		return "length"
	case "id_":
		return "identifier"
	case "error_":
		return "err"
	case "obj":
		return "object"
	case "rt", "errkit", "objref", "purego", "objc", "ebipurego", "unsafe", "context":
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

func detectClassPrefix(framework *meta.FrameworkMeta) string {
	names := sortedKeys(framework.Classes)
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

func sortedKeys(classes map[string]meta.Class) []string {
	keys := make([]string, 0, len(classes))
	for k := range classes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
