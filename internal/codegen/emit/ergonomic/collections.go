package ergonomic

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/classify"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

// ErgonomicCollections emits package-level functions that bridge CollectionReturn and
// CollectionParam methods to idiomatic Go slices / maps via the shared package helpers.
//
// CollectionReturn methods return an NSArray/NSSet/NSDictionary — the ergonomic wrapper
// converts the result to a typed Go slice ([]T).
// CollectionParam methods accept NS collection args — the ergonomic wrapper accepts a
// Go slice and converts it before calling the raw method.
func EmitCollections(w io.Writer, pkgName, rawImportPath string, framework *meta.FrameworkMeta, m *typemap.Mapper, knownClasses map[string]bool, nt *NameTracker) error {
	type entry struct {
		className  string
		methodName string
		params     []argInfo
		retGoType  string // non-empty for CollectionReturn
		isMutable  bool   // return is NSMutableArray/Set
		hasError   bool   // raw method returns (*NSArray, error)
	}

	var entries []entry
	for _, className := range sortedKeys(framework.Classes) {
		cls := framework.Classes[className]
		if cls.Availability.IsUnavailable {
			continue
		}
		ctx := m.BaseContext(framework.Framework, knownClasses)
		ctx.ClassName = className
		localImports := make(typemap.ImportSet)

		for _, method := range cls.Methods {
			if method.Availability.IsUnavailable || method.IsClassMethod {
				continue
			}
			tags := classify.ClassifyMethod(method, cls, framework)
			if !containsPatternTag(tags, classify.CollectionReturn) {
				continue
			}

			// Only wrap methods that return a typed NSArray — other collection types
			// (NSDictionary, NSSet) require caller-supplied key/value converters which
			// are too open-ended for a generated wrapper.
			retObjC := method.Return.ObjCType
			if !looksLikeNSArray(retObjC) {
				continue
			}

			elemGoType := extractNSArrayElementGoType(retObjC, ctx, m, framework, localImports)
			if elemGoType == "" {
				continue
			}
			// Only wrap arrays whose element type satisfies objc.Object (pointer types or
			// objc.Object itself). func types, scalars, etc. cannot be type-asserted.
			if !isObjcObjectType(elemGoType) {
				continue
			}
			// Skip NSMutableArray returns: in Go, *NSMutableArray[T] is not assignable to
			// *NSArray[T], so shared.NSArrayToSlice (which expects *NSArray[T]) won't compile.
			if strings.Contains(retObjC, "NSMutableArray") {
				continue
			}

			var params []argInfo
			for i, arg := range method.Params {
				goType := resolveOpinionatedArgType(arg.ObjCType, ctx, m, framework, localImports)
				argName := naming.ParamName(arg.Name)
				if argName == "" {
					argName = fmt.Sprintf("arg%d", i)
				}
				params = append(params, argInfo{name: argName, goType: goType})
			}

			entries = append(entries, entry{
				className:  className,
				methodName: naming.MethodName(method.Selector),
				params:     params,
				retGoType:  elemGoType,
				isMutable:  strings.Contains(retObjC, "NSMutableArray"),
				hasError:   method.IsNSError,
			})
		}
	}

	if len(entries) == 0 {
		return nil
	}

	usedImports := make(map[string]string)
	needsObjc := false
	needsUnsafe := false
	for _, e := range entries {
		for _, a := range e.params {
			recordOpinionatedImports(a.goType, m, usedImports)
			if strings.Contains(a.goType, "objc.") {
				needsObjc = true
			}
			if strings.Contains(a.goType, "unsafe.") {
				needsUnsafe = true
			}
		}
		recordOpinionatedImports(e.retGoType, m, usedImports)
		if strings.Contains(e.retGoType, "objc.") {
			needsObjc = true
		}
		if strings.Contains(e.retGoType, "unsafe.") {
			needsUnsafe = true
		}
	}
	sharedPkg := "shared"
	sharedPath := "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/ergonomic/shared"
	usedImports[sharedPkg] = sharedPath
	if needsObjc {
		usedImports["objc"] = "github.com/deploymenttheory/go-bindings-macosplatform/internal/objc"
	}
	if needsUnsafe {
		usedImports["unsafe"] = "unsafe"
	}

	var body bytes.Buffer

	for _, e := range entries {
		funcName := e.methodName + "Slice"
		if !nt.Claim(funcName, "collections") {
			continue
		}

		params := []string{
			"ctx context.Context",
			fmt.Sprintf("o *%s", buildRawReceiverType(e.className, m)),
		}
		var callArgs []string
		for _, a := range e.params {
			params = append(params, a.name+" "+a.goType)
			callArgs = append(callArgs, a.name)
		}
		allCallArgs := append([]string{"ctx"}, callArgs...)

		if e.hasError {
			fmt.Fprintf(&body, "func %s(%s) ([]%s, error) {\n", funcName, strings.Join(params, ", "), e.retGoType)
			fmt.Fprintf(&body, "\t_arr, _err := o.%s(%s)\n", e.methodName, strings.Join(allCallArgs, ", "))
			fmt.Fprintf(&body, "\tif _err != nil {\n\t\treturn nil, _err\n\t}\n")
			fmt.Fprintf(&body, "\treturn shared.NSArrayToSlice[%s](ctx, _arr), nil\n", e.retGoType)
			fmt.Fprintf(&body, "}\n\n")
		} else {
			fmt.Fprintf(&body, "func %s(%s) []%s {\n", funcName, strings.Join(params, ", "), e.retGoType)
			fmt.Fprintf(&body, "\treturn shared.NSArrayToSlice[%s](ctx, o.%s(%s))\n",
				e.retGoType,
				e.methodName,
				strings.Join(allCallArgs, ", "),
			)
			fmt.Fprintf(&body, "}\n\n")
		}
	}

	if body.Len() == 0 {
		return nil
	}

	recordSpecialImports(body.Bytes(), usedImports)
	writeErgonomicHeader(w, pkgName, rawImportPath, usedImports, false)
	_, err := w.Write(body.Bytes())
	return err
}

