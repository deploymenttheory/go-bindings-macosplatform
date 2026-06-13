// Package overrides applies declarative, per-framework corrections to scanned
// metadata at load time.
//
// Scanned data is never perfect: a header misdeclares a type, Clang drops an
// attribute, an API is unusable through the bridge. Without this layer the
// only correction lever is editing scanner or typemap code — a global change
// to fix a local defect. An override file fixes exactly one declaration, is
// reviewable as data, survives re-scans, and documents intent in the diff.
//
// Override files live next to the metadata they correct
// (metadata/frameworks/<name>/overrides.json) and are applied by the pipeline
// loaders after macosplatformmetadata.Read — the committed .gometa.json stays pure observed
// Clang output, with corrections as a visible, separate layer. Entries that
// no longer match anything (stale after an SDK re-scan) produce warnings.
//
// The operation set is deliberately small: exclude, remap a type, force an
// enum to bitmask, fix availability, override the link library.
package overrides

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// FileName is the override file name expected next to a framework's
// .gometa.json (e.g. metadata/frameworks/foundation/overrides.json).
const FileName = "overrides.json"

// MethodRef addresses one method on one class.
type MethodRef struct {
	Class         string `json:"class"`
	Selector      string `json:"selector"`
	IsClassMethod bool   `json:"class_method,omitempty"`
}

// TypeRemap corrects the ObjC type of a single parameter or return value.
// Address a method via Class+Selector or a C function via Function.
// Param is the parameter name to remap, or "return" for the return type.
type TypeRemap struct {
	Class         string `json:"class,omitempty"`
	Selector      string `json:"selector,omitempty"`
	IsClassMethod bool   `json:"class_method,omitempty"`
	Function      string `json:"function,omitempty"`
	Param         string `json:"param"`
	ObjCType      string `json:"objc_type"`
}

// AvailabilityFix corrects availability on a class, enum, or function.
// Only non-empty fields are applied; IsUnavailable uses a pointer so that
// "absent" and "set to false" are distinguishable.
type AvailabilityFix struct {
	Class           string `json:"class,omitempty"`
	Enum            string `json:"enum,omitempty"`
	Function        string `json:"function,omitempty"`
	MacOSIntroduced string `json:"macos_introduced,omitempty"`
	MacOSDeprecated string `json:"macos_deprecated,omitempty"`
	IsUnavailable   *bool  `json:"unavailable,omitempty"`
}

// File is one framework's override set.
type File struct {
	// Comment documents why the overrides exist (JSON has no comment syntax).
	Comment string `json:"comment,omitempty"`

	ExcludeClasses    []string          `json:"exclude_classes,omitempty"`
	ExcludeMethods    []MethodRef       `json:"exclude_methods,omitempty"`
	ExcludeFunctions  []string          `json:"exclude_functions,omitempty"`
	RemapTypes        []TypeRemap       `json:"remap_types,omitempty"`
	ForceBitmaskEnums []string          `json:"force_bitmask_enums,omitempty"`
	AvailabilityFixes []AvailabilityFix `json:"availability_fixes,omitempty"`

	// LinkLib overrides FrameworkMeta.LinkLib (the -l<lib> linker flag).
	LinkLib string `json:"link_lib,omitempty"`
}

// LoadAdjacent loads the override file that sits next to metaPath. found is
// false when no override file exists — not an error.
func LoadAdjacent(metaPath string) (file *File, found bool, err error) {
	path := filepath.Join(filepath.Dir(metaPath), FileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("reading overrides %s: %w", path, err)
	}
	var parsed File
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, false, fmt.Errorf("parsing overrides %s: %w", path, err)
	}
	return &parsed, true, nil
}

// ApplyAdjacent looks for an override file next to metaPath and applies it to
// framework. A missing file is not an error. Returned warnings list override
// entries that matched nothing — stale after an SDK re-scan.
func ApplyAdjacent(metaPath string, framework *macosplatformmetadata.FrameworkMeta) ([]string, error) {
	file, found, err := LoadAdjacent(metaPath)
	if err != nil || !found {
		return nil, err
	}
	return Apply(file, framework), nil
}

// Apply mutates framework according to file and returns a warning for every
// override entry that matched no declaration.
func Apply(file *File, framework *macosplatformmetadata.FrameworkMeta) []string {
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

func describeRemap(remap TypeRemap) string {
	if remap.Function != "" {
		return fmt.Sprintf("function %s param %q", remap.Function, remap.Param)
	}
	return fmt.Sprintf("%s.%s param %q", remap.Class, remap.Selector, remap.Param)
}

// applyRemap rewrites one param/return ObjC type. Reports whether anything matched.
func applyRemap(remap TypeRemap, framework *macosplatformmetadata.FrameworkMeta) bool {
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

func remapParams(remap TypeRemap, params []macosplatformmetadata.Param, retType *macosplatformmetadata.ReturnType) bool {
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

func applyAvailabilityFix(fix AvailabilityFix, framework *macosplatformmetadata.FrameworkMeta) bool {
	patch := func(avail *macosplatformmetadata.Availability) {
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
