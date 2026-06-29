package library

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// Specs writes Spec structs and Apply functions for *Configuration classes.
//
// For every class whose name ends in "Configuration" with ≥ 3 non-readonly
// properties, a plain Go struct (FooConfigurationSpec) and an Apply function
// are emitted. The Apply function sets non-zero fields and calls
// ValidateWithError when the class supports it.
//
// The caller is responsible for creating the configuration instance (e.g. via
// a factory function in this package) before calling Apply.
func EmitSpecs(w io.Writer, pkgName, rawImportPath string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) error {
	type specField struct {
		name          string // exported Go field name
		goType        string // resolved Go type (may include raw. prefix)
		setter        string // setter method name on the raw type
		isSlice       bool   // true when backed by NSArray
		isInterface   bool   // true when Go type is an interface (ObjC id<Protocol>)
		isValueStruct bool   // true when Go type is a C value struct (CGRect, CMTime, etc.)
		elemClass     string // element class for slice fields
	}
	type specEntry struct {
		className string
		fields    []specField
	}

	var specs []specEntry
	for _, className := range sortedKeys(framework.Classes) {
		class := framework.Classes[className]
		if class.Availability.IsUnavailable || !strings.HasSuffix(className, "Configuration") {
			continue
		}

		ctx := m.BaseContext(framework.Framework, knownClasses)
		ctx.ClassName = className
		localImports := make(typemap.ImportSet)

		var fields []specField
		seenFields := make(map[string]bool)
		for _, prop := range class.Properties {
			if prop.IsReadOnly || prop.Availability.IsUnavailable {
				continue
			}
			// Skip properties that don't have a real instance setter method.
			// Properties backed by class methods or with no setter are effectively
			// read-only from an instance perspective even if not flagged readonly.
			if !classHasInstanceSetter(class, prop.Name) {
				continue
			}
			fieldName := propertyGoName(prop.Name)
			if seenFields[fieldName] {
				continue
			}
			seenFields[fieldName] = true
			setter := "Set" + fieldName

			if elemClass, ok := extractNSArrayElementType(prop.ObjCType); ok {
				if _, inFW := framework.Classes[elemClass]; inFW {
					fields = append(fields, specField{
						name:      fieldName,
						goType:    "[]*raw." + elemClass,
						setter:    setter,
						isSlice:   true,
						elemClass: elemClass,
					})
					continue
				}
			}

			goType := m.GoType(prop.ObjCType, ctx, localImports)
			if goType == "" || goType == "unsafe.Pointer" {
				continue
			}
			goType = qualifyWithRawPackage(goType, framework)
			cleanObjC := extractObjCBaseTypeName(prop.ObjCType)
			isIface := strings.HasPrefix(strings.TrimSpace(prop.ObjCType), "id<")
			_, sameStruct := framework.Structs[cleanObjC]
			isValStruct := !isIface && (sameStruct || m.StructIndex[cleanObjC] != "")
			fields = append(fields, specField{
				name:          fieldName,
				goType:        goType,
				setter:        setter,
				isSlice:       false,
				isInterface:   isIface,
				isValueStruct: isValStruct,
			})
		}

		if len(fields) < 3 {
			continue
		}
		specs = append(specs, specEntry{className: className, fields: fields})
	}

	if len(specs) == 0 {
		return nil
	}

	usedImports := make(map[string]string)
	needsFoundation := false
	needsObjc := false
	for _, spec := range specs {
		for _, f := range spec.fields {
			recordOpinionatedImports(f.goType, m, usedImports)
			if f.isSlice {
				needsFoundation = true
				needsObjc = true
			}
			if strings.Contains(f.goType, "cgo.") {
				needsObjc = true
			}
		}
	}

	var body bytes.Buffer
	for _, spec := range specs {
		specTypeName := spec.className + "Spec"

		fmt.Fprintf(&body, "// %s holds the fields used to configure a %s.\n", specTypeName, spec.className)
		fmt.Fprintf(&body, "// Pass a zero-value field to leave the corresponding property unchanged.\n")
		fmt.Fprintf(&body, "// Create the %s instance before calling Apply%s.\n", spec.className, spec.className)
		fmt.Fprintf(&body, "type %s struct {\n", specTypeName)
		for _, f := range spec.fields {
			fmt.Fprintf(&body, "\t%s %s\n", f.name, f.goType)
		}
		fmt.Fprintf(&body, "}\n\n")

		fmt.Fprintf(&body, "// Apply%s applies spec to c, setting non-zero fields", spec.className)
		if classHasValidateMethod(framework.Classes[spec.className]) {
			fmt.Fprintf(&body, " and calling ValidateWithError")
		}
		fmt.Fprintf(&body, ".\n")
		fmt.Fprintf(&body, "func Apply%s(c *raw.%s, spec %s) error {\n",
			spec.className, spec.className, specTypeName)
		for _, f := range spec.fields {
			if f.isSlice {
				fmt.Fprintf(&body, "\tif len(spec.%s) > 0 {\n", f.name)
				fmt.Fprintf(&body, "\t\t_objs := make([]cgo.Object, len(spec.%s))\n", f.name)
				fmt.Fprintf(&body, "\t\tfor _i, _d := range spec.%s { _objs[_i] = _d }\n", f.name)
				fmt.Fprintf(&body, "\t\tc.%s(foundation.NSArrayOf[cgo.Object](_objs...))\n", f.setter)
				fmt.Fprintf(&body, "\t}\n")
			} else if strings.HasPrefix(f.goType, "*") || f.isInterface {
				fmt.Fprintf(&body, "\tif spec.%s != nil { c.%s(spec.%s) }\n", f.name, f.setter, f.name)
			} else if f.goType == "bool" {
				fmt.Fprintf(&body, "\tif spec.%s { c.%s(spec.%s) }\n", f.name, f.setter, f.name)
			} else if f.isValueStruct {
				fmt.Fprintf(&body, "\tif spec.%s != (%s{}) { c.%s(spec.%s) }\n", f.name, f.goType, f.setter, f.name)
			} else {
				fmt.Fprintf(&body, "\tif spec.%s != 0 { c.%s(spec.%s) }\n", f.name, f.setter, f.name)
			}
		}
		if classHasValidateMethod(framework.Classes[spec.className]) {
			fmt.Fprintf(&body, "\tif _, err := c.ValidateWithError(); err != nil {\n")
			fmt.Fprintf(&body, "\t\treturn err\n")
			fmt.Fprintf(&body, "\t}\n")
		}
		fmt.Fprintf(&body, "\treturn nil\n")
		fmt.Fprintf(&body, "}\n\n")
	}

	if body.Len() == 0 {
		return nil
	}

	extraImports := make(map[string]string)
	if needsFoundation {
		extraImports["foundation"] = "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
	}
	writeOpinionatedHeader(w, pkgName, rawImportPath, extraImports, usedImports, needsObjc)
	_, err := w.Write(body.Bytes())
	return err
}
