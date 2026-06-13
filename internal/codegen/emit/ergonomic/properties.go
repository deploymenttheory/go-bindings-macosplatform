package ergonomic

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// ErgonomicProperties emits context-free package-level getter/setter function
// pairs for each non-readonly property on every class (PropertyPair pattern).
//
// Generated shape:
//
//	func AlphaValue(o *raw.NSView) float64
//	func SetAlphaValue(o *raw.NSView, v float64)
//
// Properties are simple synchronous reads/writes and rarely need tracing at the
// OTel level. Context-free wrappers inject context.Background() internally so
// callers don't need to thread a context through property access chains. Complex
// methods (blocks, async, NSError) remain in other ergonomic emitters where a
// real context matters.
func EmitProperties(w io.Writer, pkgName, rawImportPath string, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, nt *NameTracker) error {
	type propEntry struct {
		className string
		propName  string // Go-exported name (e.g. "AlphaValue")
		goType    string
		getter    string // raw Go method name
		setter    string // raw Go method name ("" for readonly — excluded)
	}

	var entries []propEntry
	for _, className := range sortedKeys(framework.Classes) {
		cls := framework.Classes[className]
		if cls.Availability.IsUnavailable {
			continue
		}
		ctx := m.BaseContext(framework.Framework, knownClasses)
		ctx.ClassName = className
		localImports := make(typemap.ImportSet)

		// Build a set of instance method selectors so we can verify setters exist.
		classMethods := make(map[string]bool)
		for _, mm := range cls.Methods {
			if !mm.IsClassMethod && !mm.Availability.IsUnavailable {
				classMethods[mm.Selector] = true
			}
		}

		for _, prop := range cls.Properties {
			if prop.Availability.IsUnavailable || prop.IsReadOnly {
				continue
			}
			// Block-typed properties (e.g. completionHandler, readabilityHandler) are
			// handled by ErgonomicAsync, which produces a better wrapper. Skip them here
			// to avoid declaring the same Set<Prop> function in two files.
			// ObjC block types contain "(^" regardless of nullability annotations.
			if strings.Contains(prop.ObjCType, "(^") {
				continue
			}
			goType := resolveOpinionatedArgType(prop.ObjCType, ctx, m, framework, localImports)
			if goType == "" || goType == "unsafe.Pointer" {
				continue
			}
			// Compound protocol constraint types (id<P1, P2>) produce interface { P1; P2 }
			// which may not match the raw method's actual return type (objc.Object). Skip.
			if strings.HasPrefix(goType, "interface {") {
				continue
			}

			getterSel := prop.Name
			if prop.Getter != "" {
				getterSel = prop.Getter
			}
			setterSel := prop.Setter
			if setterSel == "" && len(prop.Name) > 0 {
				setterSel = "set" + strings.ToUpper(prop.Name[:1]) + prop.Name[1:] + ":"
			}

			// Verify both getter and setter methods actually exist in the raw bindings.
			// Some properties appear non-readonly in metadata yet have no setter
			// implementation (e.g. computed value properties, or C-struct returns
			// whose type was degraded to unsafe.Pointer in the raw binding). Without
			// this guard the emitter would generate Set calls that won't compile.
			if !classMethods[setterSel] || !classMethods[getterSel] {
				continue
			}

			// Protocol-typed properties (id<P>) return *PIDProtocol in raw bindings
			// but an interface type in the ergonomic layer — the two are not
			// assignable. The delegates emitter handles these via proper ObjC shims.
			trimmedObjCType := strings.TrimSpace(prop.ObjCType)
			for _, ann := range []string{"_Nullable ", "_Nonnull ", "_Null_unspecified "} {
				trimmedObjCType = strings.TrimPrefix(trimmedObjCType, ann)
			}
			if strings.HasPrefix(trimmedObjCType, "id<") {
				continue
			}

			// Skip properties whose raw getter returns unsafe.Pointer (e.g. C-struct
			// return types degraded by the raw emitter). The ergonomic type resolver
			// may produce a different type, causing an incompatible assignment.
			rawReturnType := m.GoType(prop.ObjCType, ctx, localImports)
			if rawReturnType == "unsafe.Pointer" {
				continue
			}
			// Skip block-typedef properties (e.g. NSItemProviderLoadHandler). The
			// ObjC type is a named typedef whose underlying type is a block. The raw
			// binding always returns unsafe.Pointer for block return values because
			// ObjC blocks cannot be meaningfully round-tripped to Go. The type
			// resolver does NOT degrade typedef names, so we detect by checking if
			// the resolved type is a func — func-typed properties would have been
			// caught by the "(^" check above only when using explicit block syntax.
			if strings.HasPrefix(rawReturnType, "func(") {
				continue
			}
			// SEL-typed properties (nullable) generate *string in raw bindings but the
			// ergonomic layer resolves SEL to string — incompatible assignment.
			if typemap.IsSEL(prop.ObjCType) {
				continue
			}

			entries = append(entries, propEntry{
				className: className,
				propName:  naming.MethodName(getterSel),
				goType:    goType,
				getter:    naming.MethodName(getterSel),
				setter:    naming.MethodName(setterSel),
			})
		}
	}

	if len(entries) == 0 {
		return nil
	}

	usedImports := make(map[string]string)
	for _, e := range entries {
		recordOpinionatedImports(e.goType, m, usedImports)
	}

	var body bytes.Buffer

	for _, e := range entries {
		// Deduplicate by the getter function name across all ergonomic emitters.
		// Also claim the setter name so no other emitter can collide with it.
		if !nt.Claim(e.propName, "properties") {
			continue
		}
		nt.Claim("Set"+e.propName, "properties")

		recvType := buildRawReceiverType(e.className, m)
		fmt.Fprintf(&body, "func %s(o *%s) %s {\n",
			e.propName, recvType, e.goType)
		fmt.Fprintf(&body, "\treturn o.%s(context.Background())\n}\n\n", e.getter)

		fmt.Fprintf(&body, "func Set%s(o *%s, v %s) {\n",
			e.propName, recvType, e.goType)
		fmt.Fprintf(&body, "\to.%s(context.Background(), v)\n}\n\n", e.setter)
	}

	if body.Len() == 0 {
		return nil
	}

	recordSpecialImports(body.Bytes(), usedImports)
	writeErgonomicHeader(w, pkgName, rawImportPath, usedImports, false)
	_, err := w.Write(body.Bytes())
	return err
}
