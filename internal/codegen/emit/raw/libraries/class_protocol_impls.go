package rawlib

import (
	"bytes"
	"fmt"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/render"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/emit/raw/libraries/view"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/naming"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/libraries/typemap"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// EmitProtocolImpls writes per-protocol callback files for all protocols in framework.
//
// For each protocol NSFooDelegate it writes:
//   - outDir/NSFooDelegate_protocol_callback.go  — NSFooDelegateCallbacks struct + NewNSFooDelegateProtocolCallback
//   - outDir/bridge/NSFooDelegate_protocol_callback.h
//   - outDir/bridge/NSFooDelegate_protocol_callback.m
func EmitProtocolImpls(outDir string, framework *macosplatformmetadata.FrameworkMeta, m *typemap.Mapper) error {
	packageName := strings.ToLower(framework.Framework)
	bridgeDir := filepath.Join(outDir, "bridge")

	names := make([]string, 0, len(framework.Protocols))
	for name := range framework.Protocols {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, protoName := range names {
		proto := framework.Protocols[protoName]
		if proto.Availability.IsUnavailable {
			continue
		}
		methods := collectProtocolMethods(proto, m)
		if len(methods) == 0 {
			continue
		}

		var goBuf bytes.Buffer
		if err := emitProtocolImplGo(&goBuf, packageName, framework.Framework, protoName, methods, m); err != nil {
			return fmt.Errorf("protocol impl go %s: %w", protoName, err)
		}
		if goBuf.Len() > 0 {
			goPath := filepath.Join(outDir, protoName+"_protocol_callback.go")
			if err := os.WriteFile(goPath, goBuf.Bytes(), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", goPath, err)
			}
		}

		var hBuf bytes.Buffer
		emitProtocolImplHeader(&hBuf, protoName, methods)
		hPath := filepath.Join(bridgeDir, protoName+"_protocol_callback.h")
		if err := os.WriteFile(hPath, hBuf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", hPath, err)
		}

		var mBuf bytes.Buffer
		emitProtocolImplImpl(&mBuf, framework.Framework, protoName, methods)
		mPath := filepath.Join(bridgeDir, protoName+"_protocol_callback.m")
		if err := os.WriteFile(mPath, mBuf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", mPath, err)
		}
	}
	return nil
}

// collectProtocolMethods returns all IMP-safe instance methods from proto.
func collectProtocolMethods(proto macosplatformmetadata.Protocol, m *typemap.Mapper) []overrideMethod {
	seen := make(map[string]bool)
	var result []overrideMethod

	for _, method := range proto.Methods {
		if !isMethodIMPSafe(method, m) {
			continue
		}
		sig, ok := methodSigFromMethod(method, m)
		if !ok {
			continue
		}
		goName := naming.MethodName(method.Selector)
		if seen[goName] {
			continue
		}
		seen[goName] = true
		result = append(result, overrideMethod{
			GoName:   goName,
			Selector: method.Selector,
			Sig:      sig,
		})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].GoName < result[j].GoName })
	return result
}

// emitProtocolImplGo writes the Go factory for a protocol implementation object.
func emitProtocolImplGo(w *bytes.Buffer, packageName, framework, protoName string, methods []overrideMethod, m *typemap.Mapper) error {
	return render.Execute(w, "protocol_impl_go_file", buildProtocolImplGoFileModel(packageName, framework, protoName, methods, m))
}

// emitProtocolImplHeader writes the .h file for a protocol callback.
func emitProtocolImplHeader(w *bytes.Buffer, protoName string, methods []overrideMethod) {
	goClassName := "goBridge_Proto_" + protoName
	_ = render.Execute(w, "protocol_impl_h_file", view.ProtocolImplHFileModel{
		GoClassName: goClassName,
		IMPSigs:     uniqueIMPSigsFor(methods),
	})
}

// emitProtocolImplImpl writes the .m file for a protocol callback.
func emitProtocolImplImpl(w *bytes.Buffer, framework, protoName string, methods []overrideMethod) {
	goClassName := "goBridge_Proto_" + protoName
	_ = render.Execute(w, "protocol_impl_m_file", view.ProtocolImplMFileModel{
		Framework:   framework,
		ProtoName:   protoName,
		GoClassName: goClassName,
		HeaderFile:  protoName + "_protocol_callback.h",
		IMPSigs:     uniqueIMPSigsFor(methods),
	})
}

func buildProtocolImplGoFileModel(packageName, framework, protoName string, methods []overrideMethod, m *typemap.Mapper) view.ProtocolImplGoFileModel {
	goClassName := "goBridge_Proto_" + protoName
	callbacksStructName := protoName + "Callbacks"
	factoryName := "New" + protoName + "ProtocolCallback"
	impSigs := uniqueIMPSigsFor(methods)

	goProtoName := naming.ProtocolGoTypeName(protoName, m.OwnerIndex)
	_, hasProxy := m.ProtocolProxyIndex[protoName]

	var retType string
	if hasProxy {
		retType = "*" + goProtoName + "IDProtocol"
	} else {
		retType = "cgo.Object"
	}

	// Pre-render the callbacks struct fields.
	var implFields strings.Builder
	for _, method := range methods {
		funcType := method.Sig.goCallbackFuncType()
		fmt.Fprintf(&implFields, "\t// %s implements -[%s %s].\n", method.GoName, protoName, method.Selector)
		fmt.Fprintf(&implFields, "\t%s %s\n", method.GoName, funcType)
	}

	// Pre-render the factory function doc comment.
	var factoryComment strings.Builder
	fmt.Fprintf(&factoryComment, "// %s allocates an ObjC object conforming to %s and routes all\n", factoryName, protoName)
	fmt.Fprintf(&factoryComment, "// protocol messages to the provided Go callbacks.\n")
	if hasProxy {
		fmt.Fprintf(&factoryComment, "// The returned *%sIDProtocol satisfies the %s Go interface and\n", goProtoName, protoName)
		fmt.Fprintf(&factoryComment, "// can be passed directly to any method that accepts a %s delegate.\n", protoName)
	} else {
		fmt.Fprintf(&factoryComment, "// The returned cgo.Object wraps the +1-retained ObjC delegate and\n")
		fmt.Fprintf(&factoryComment, "// can be passed to any id<%s> parameter.\n", protoName)
	}
	fmt.Fprintf(&factoryComment, "// Callers are responsible for retaining the object as long as the delegate is needed.\n")

	// Pre-render the addMethod if-blocks (at 1-tab indent, no outer wrapper).
	var addMethodBody strings.Builder
	for _, method := range methods {
		enc := method.Sig.ObjCEnc
		getterFn := impGetterName(method.Sig.Name)
		fmt.Fprintf(&addMethodBody, "\tif fns.%s != nil {\n", method.GoName)
		fmt.Fprintf(&addMethodBody, "\t\t_sel%s := C.CString(%q)\n", method.GoName, method.Selector)
		fmt.Fprintf(&addMethodBody, "\t\t_enc%s := C.CString(%q)\n", method.GoName, enc)
		fmt.Fprintf(&addMethodBody, "\t\tC.%s_addMethod(cls, _sel%s, C.%s(), _enc%s)\n",
			goClassName, method.GoName, getterFn, method.GoName)
		fmt.Fprintf(&addMethodBody, "\t\tC.free(unsafe.Pointer(_sel%s))\n", method.GoName)
		fmt.Fprintf(&addMethodBody, "\t\tC.free(unsafe.Pointer(_enc%s))\n", method.GoName)
		fmt.Fprintf(&addMethodBody, "\t}\n")
	}

	// Pre-render the BindMethod if-blocks.
	var bindMethodBody strings.Builder
	for _, method := range methods {
		fmt.Fprintf(&bindMethodBody, "\tif fns.%s != nil {\n", method.GoName)
		fmt.Fprintf(&bindMethodBody, "\t\tcallbacks.BindMethod(obj, %q, fns.%s)\n", method.Selector, method.GoName)
		fmt.Fprintf(&bindMethodBody, "\t}\n")
	}

	// Pre-render the return statement(s). No trailing \n — the template's \n}
	// line provides the final newline before the closing brace.
	var returnBody strings.Builder
	if hasProxy {
		fmt.Fprintf(&returnBody, "\treturn New%sIDProtocol(obj)", goProtoName)
	} else {
		fmt.Fprintf(&returnBody, "\twrapped := cgo.WrapObject(obj)\n")
		fmt.Fprintf(&returnBody, "\tif wrapped != nil {\n")
		fmt.Fprintf(&returnBody, "\t\tcgo.Track(wrapped, wrapped.Ptr)\n")
		fmt.Fprintf(&returnBody, "\t}\n")
		fmt.Fprintf(&returnBody, "\treturn wrapped")
	}

	return view.ProtocolImplGoFileModel{
		PackageName:         packageName,
		GoClassName:         goClassName,
		ProtoName:           protoName,
		CallbacksStructName: callbacksStructName,
		FactoryName:         factoryName,
		IMPSigs:             impSigs,
		NeedsObjc:           !hasProxy,
		ImplFields:          implFields.String(),
		FactoryComment:      factoryComment.String(),
		ReturnType:          retType,
		AddMethodBody:       addMethodBody.String(),
		BindMethodBody:      bindMethodBody.String(),
		ReturnBody:          returnBody.String(),
	}
}
