// Package overrides applies the shared declarative override schema
// (internal/overrides) to the purego generator's metadata model
// (internal/codegen/frameworks/meta). The file format, discovery convention, and
// semantics are identical to internal/overrides — only the mutated types
// differ, because the purecg tree mirrors codegen with its own meta package.
package overrides

import (
	"fmt"

	rootoverrides "github.com/deploymenttheory/go-bindings-macosplatform/internal/overrides"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/meta"
)

// ApplyAdjacent looks for an override file next to metaPath and applies it to
// framework. A missing file is not an error. Returned warnings list override
// entries that matched nothing — stale after an SDK re-scan.
func ApplyAdjacent(metaPath string, framework *meta.FrameworkMeta) ([]string, error) {
	file, found, err := rootoverrides.LoadAdjacent(metaPath)
	if err != nil {
		return nil, fmt.Errorf("loading overrides for %s: %w", framework.Framework, err)
	}
	if !found {
		return nil, nil
	}
	return Apply(file, framework), nil
}

// Apply mutates framework according to file and returns a warning for every
// override entry that matched no declaration.
func Apply(file *rootoverrides.File, framework *meta.FrameworkMeta) []string {
	var warnings []string
	stale := func(format string, args ...any) {
		warnings = append(
			warnings,
			fmt.Sprintf(
				"%s overrides: ",
				framework.Framework,
			)+fmt.Sprintf(
				format,
				args...)+" — stale entry?",
		)
	}

	for _, className := range file.ExcludeClasses {
		if _, ok := framework.Classes[className]; !ok {
			stale("exclude_classes: no class %q", className)
			continue
		}
		delete(framework.Classes, className)
	}

	for _, ref := range file.ExcludeMethods {
		class, ok := framework.Classes[ref.Class]
		if !ok {
			stale("exclude_methods: no class %q", ref.Class)
			continue
		}
		kept := class.Methods[:0]
		matched := false
		for _, method := range class.Methods {
			if method.Selector == ref.Selector && method.IsClassMethod == ref.IsClassMethod {
				matched = true
				continue
			}
			kept = append(kept, method)
		}
		if !matched {
			stale(
				"exclude_methods: no method %s on %s (class_method=%v)",
				ref.Selector,
				ref.Class,
				ref.IsClassMethod,
			)
			continue
		}
		class.Methods = kept
		framework.Classes[ref.Class] = class
	}

	for _, name := range file.ExcludeFunctions {
		kept := framework.Functions[:0]
		matched := false
		for _, function := range framework.Functions {
			if function.Name == name {
				matched = true
				continue
			}
			kept = append(kept, function)
		}
		if !matched {
			stale("exclude_functions: no function %q", name)
			continue
		}
		framework.Functions = kept
	}

	for _, remap := range file.RemapTypes {
		if !applyRemap(remap, framework) {
			stale("remap_types: no match for %s", describeRemap(remap))
		}
	}

	for _, enumName := range file.ForceBitmaskEnums {
		enum, ok := framework.Enums[enumName]
		if !ok {
			stale("force_bitmask_enums: no enum %q", enumName)
			continue
		}
		enum.IsBitmask = true
		framework.Enums[enumName] = enum
	}

	for _, fix := range file.AvailabilityFixes {
		if !applyAvailabilityFix(fix, framework) {
			stale(
				"availability_fixes: no match for class=%q enum=%q function=%q",
				fix.Class,
				fix.Enum,
				fix.Function,
			)
		}
	}

	if file.LinkLib != "" {
		framework.LinkLib = file.LinkLib
	}

	return warnings
}

func describeRemap(remap rootoverrides.TypeRemap) string {
	if remap.Function != "" {
		return fmt.Sprintf("function %s param %q", remap.Function, remap.Param)
	}
	return fmt.Sprintf("%s.%s param %q", remap.Class, remap.Selector, remap.Param)
}

func applyRemap(remap rootoverrides.TypeRemap, framework *meta.FrameworkMeta) bool {
	if remap.Function != "" {
		matched := false
		for i := range framework.Functions {
			if framework.Functions[i].Name != remap.Function {
				continue
			}
			if remapParams(remap, framework.Functions[i].Params, &framework.Functions[i].Return) {
				matched = true
			}
		}
		return matched
	}

	class, ok := framework.Classes[remap.Class]
	if !ok {
		return false
	}
	matched := false
	for i := range class.Methods {
		method := &class.Methods[i]
		if method.Selector != remap.Selector || method.IsClassMethod != remap.IsClassMethod {
			continue
		}
		if remapParams(remap, method.Params, &method.Return) {
			matched = true
		}
	}
	if matched {
		framework.Classes[remap.Class] = class
	}
	return matched
}

func remapParams(
	remap rootoverrides.TypeRemap,
	params []meta.Param,
	retType *meta.ReturnType,
) bool {
	if remap.Param == "return" {
		retType.ObjCType = remap.ObjCType
		return true
	}
	for i := range params {
		if params[i].Name == remap.Param {
			params[i].ObjCType = remap.ObjCType
			return true
		}
	}
	return false
}

func applyAvailabilityFix(fix rootoverrides.AvailabilityFix, framework *meta.FrameworkMeta) bool {
	patch := func(avail *meta.Availability) {
		if fix.MacOSIntroduced != "" {
			avail.MacOSIntroduced = fix.MacOSIntroduced
		}
		if fix.MacOSDeprecated != "" {
			avail.MacOSDeprecated = fix.MacOSDeprecated
		}
		if fix.IsUnavailable != nil {
			avail.IsUnavailable = *fix.IsUnavailable
		}
	}

	switch {
	case fix.Class != "":
		class, ok := framework.Classes[fix.Class]
		if !ok {
			return false
		}
		patch(&class.Availability)
		framework.Classes[fix.Class] = class
		return true
	case fix.Enum != "":
		enum, ok := framework.Enums[fix.Enum]
		if !ok {
			return false
		}
		patch(&enum.Availability)
		framework.Enums[fix.Enum] = enum
		return true
	case fix.Function != "":
		matched := false
		for i := range framework.Functions {
			if framework.Functions[i].Name == fix.Function {
				patch(&framework.Functions[i].Availability)
				matched = true
			}
		}
		return matched
	}
	return false
}
