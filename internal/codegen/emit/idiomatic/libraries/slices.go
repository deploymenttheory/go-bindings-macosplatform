package idiolib

import (
	"bytes"
	"fmt"
	"io"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// Slices writes typed slice accessors for NSArray-typed properties.
//
// For every property whose ObjC type is NSArray<ClassName *> where ClassName
// belongs to the same framework, two functions are emitted:
//
//   - FooList(recv) []*raw.Foo  — typed getter
//   - SetFooList(recv, items)   — typed setter (non-readonly properties only)
func EmitSlices(w io.Writer, pkgName, rawImportPath string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool) error {
	type sliceEntry struct {
		className string
		propName  string // capitalised Go name (e.g. "AudioDevices")
		elemClass string // element class in this framework (e.g. "VZAudioDeviceConfiguration")
		readonly  bool
	}

	var entries []sliceEntry
	for _, className := range sortedKeys(framework.Classes) {
		class := framework.Classes[className]
		if class.Availability.IsUnavailable {
			continue
		}
		for _, prop := range class.Properties {
			if prop.Availability.IsUnavailable {
				continue
			}
			elemClass, ok := extractNSArrayElementType(prop.ObjCType)
			if !ok {
				continue
			}
			if _, inFW := framework.Classes[elemClass]; !inFW {
				continue
			}
			// Skip properties where the getter is a class method — the raw
			// framework emits these as free functions with no instance receiver.
			getterSel := prop.Name
			if prop.Getter != "" {
				getterSel = prop.Getter
			}
			if classGetterIsClassMethod(class, getterSel) {
				continue
			}
			// Only emit slice accessors for properties with a real instance setter.
			// Getter-only (readonly-in-practice) properties on live objects are less
			// commonly needed through a typed accessor.
			if !classHasInstanceSetter(class, prop.Name) {
				continue
			}
			entries = append(entries, sliceEntry{
				className: className,
				propName:  propertyGoName(prop.Name),
				elemClass: elemClass,
				readonly:  prop.IsReadOnly,
			})
		}
	}

	if len(entries) == 0 {
		return nil
	}

	var body bytes.Buffer
	seen := make(map[string]bool)
	needsFoundation := false
	needsObjc := false

	for _, e := range entries {
		// Deduplicate by getter function name: when two classes share the same
		// property name (e.g. VZMacGraphicsDeviceConfiguration and
		// VZVirtioGraphicsDeviceConfiguration both have "displays"), only emit
		// the first one encountered in sorted order.
		getterFnName := e.propName + "List"
		if seen[getterFnName] {
			continue
		}
		getterKey := e.className + ".Get" + e.propName
		if !seen[getterKey] {
			seen[getterKey] = true
			seen[getterFnName] = true
			fmt.Fprintf(&body, "// %sList returns the %s of %s as a typed Go slice.\n",
				e.propName, e.propName, e.className)
			fmt.Fprintf(&body, "func %sList(recv *raw.%s) []*raw.%s {\n",
				e.propName, e.className, e.elemClass)
			fmt.Fprintf(&body, "\ta := recv.%s()\n", e.propName)
			fmt.Fprintf(&body, "\tif a == nil {\n\t\treturn nil\n\t}\n")
			fmt.Fprintf(&body, "\tn := int(a.Count())\n")
			fmt.Fprintf(&body, "\tout := make([]*raw.%s, n)\n", e.elemClass)
			fmt.Fprintf(&body, "\tfor i := range n {\n")
			fmt.Fprintf(&body, "\t\tout[i] = raw.Cast%s(a.ObjectAtIndex(uint64(i)))\n", e.elemClass)
			fmt.Fprintf(&body, "\t}\n")
			fmt.Fprintf(&body, "\treturn out\n")
			fmt.Fprintf(&body, "}\n\n")
		}

		if !e.readonly {
			setterKey := e.className + ".Set" + e.propName
			if !seen[setterKey] {
				seen[setterKey] = true
				needsFoundation = true
				needsObjc = true
				fmt.Fprintf(&body, "// Set%sList sets the %s of %s from a typed Go slice.\n",
					e.propName, e.propName, e.className)
				fmt.Fprintf(&body, "func Set%sList(recv *raw.%s, items []*raw.%s) {\n",
					e.propName, e.className, e.elemClass)
				fmt.Fprintf(&body, "\tobjects := make([]objptr.Object, len(items))\n")
				fmt.Fprintf(&body, "\tfor i, item := range items {\n\t\tobjects[i] = item\n\t}\n")
				fmt.Fprintf(&body, "\trecv.Set%s(foundation.NSArrayOf[objptr.Object](objects...))\n", e.propName)
				fmt.Fprintf(&body, "}\n\n")
			}
		}
	}

	if body.Len() == 0 {
		return nil
	}

	extraImports := make(map[string]string)
	if needsFoundation {
		extraImports["foundation"] = "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
	}
	if err := writeOpinionatedHeader(w, pkgName, rawImportPath, extraImports, nil, needsObjc); err != nil {
		return err
	}
	_, err := w.Write(body.Bytes())
	return err
}
