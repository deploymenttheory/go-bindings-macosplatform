package render

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/codegen/frameworks/emit/idiomatic/view"
)

// update rewrites the golden files instead of comparing against them. Run
// `go test ./.../render/ -update` after an intentional template change, then
// review the diff.
var update = flag.Bool("update", false, "update golden files")

// checkGolden compares got against testdata/<name>.golden, or rewrites it when
// -update is set. It keeps the render templates honest: any change to emitted
// bytes shows up as a failing golden the author must consciously accept.
func checkGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to create)", path, err)
	}
	if string(got) != string(want) {
		t.Errorf("render mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

// TestEnumsGolden locks the enum template across the bitmask and switch forms
// and the doc/deprecation comment rendering.
func TestEnumsGolden(t *testing.T) {
	enums := []view.Enum{
		{
			GoName:       "VirtualMachineState",
			GoType:       "int64",
			CommentBlock: "// VirtualMachineState describes the run state of a virtual machine.\n",
			Members: []view.EnumMember{
				{ConstName: "VirtualMachineStateStopped", Value: "0", IsZeroVal: true},
				{ConstName: "VirtualMachineStateRunning", Value: "1"},
				{ConstName: "VirtualMachineStatePaused", Value: "2", CommentBlock: "\t// VirtualMachineStatePaused is a paused machine.\n"},
			},
			UniqueMembers: []view.EnumMember{
				{ConstName: "VirtualMachineStateStopped", Value: "0", IsZeroVal: true},
				{ConstName: "VirtualMachineStateRunning", Value: "1"},
				{ConstName: "VirtualMachineStatePaused", Value: "2"},
			},
			DefaultFmt: "VirtualMachineState(%d)",
		},
		{
			GoName:    "ActivityOptions",
			GoType:    "uint64",
			IsBitmask: true,
			Members: []view.EnumMember{
				{ConstName: "ActivityIdleDisplaySleepDisabled", Value: "1099511627776"},
				{ConstName: "ActivityIdleSystemSleepDisabled", Value: "1048576"},
			},
			DefaultFmt: "ActivityOptions(%d)",
		},
	}
	got, err := Enums(enums)
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "enums", got)
}

// TestMethodGolden locks the plain method/dispatch_tail template across every
// ResultKind and the with/without-error and out-parameter variants.
func TestMethodGolden(t *testing.T) {
	cases := []struct {
		name string
		m    view.Method
	}{
		{"method_scalar", view.Method{
			Recv: "(x *Array) ", GoName: "Count", RetSig: " int",
			Dispatch: view.Dispatch{Call: `objc.Send[int](objref.IDOf(x), objc.RegisterName("count"))`, RetKind: view.RetScalar},
		}},
		{"method_object_guard", view.Method{
			Recv: "(x *Array) ", GoName: "ObjectAtIndex", ParamStr: "index int", RetSig: " obj.Object",
			Dispatch: view.Dispatch{
				Guards:  []string{"errkit.CheckIndex(index, x.Count())"},
				Call:    `objc.Send[objc.ID](objref.IDOf(x), objc.RegisterName("objectAtIndex:"), index)`,
				RetKind: view.RetObject, RetWrap: "obj.Wrap(%s)", RetZero: "nil",
			},
		}},
		{"method_string", view.Method{
			Recv: "(x *Array) ", GoName: "ComponentsJoinedByString", ParamStr: "separator string", RetSig: " string",
			Dispatch: view.Dispatch{Call: `objc.Send[objc.ID](objref.IDOf(x), objc.RegisterName("componentsJoinedByString:"), purego.NSString(separator))`, RetKind: view.RetString},
		}},
		{"method_void_error", view.Method{
			Recv: "(x *Data) ", GoName: "WriteToURL", ParamStr: "url string", RetSig: " error",
			Dispatch: view.Dispatch{Call: `objc.Send[bool](objref.IDOf(x), objc.RegisterName("writeToURL:error:"), rt.FileURL(url), unsafe.Pointer(&_nsErr))`, Error: true, RetKind: view.RetVoid},
		}},
		{"method_outparam", view.Method{
			Recv: "(x *Serializer) ", GoName: "PropertyList", ParamStr: "data obj.Object", RetSig: " (obj.Object, PropertyListFormat, error)",
			Dispatch: view.Dispatch{
				Call:    `objc.Send[objc.ID](objref.IDOf(x), objc.RegisterName("propertyListWithData:format:error:"), objref.IDOf(data), unsafe.Pointer(&_out0), unsafe.Pointer(&_nsErr))`,
				Error:   true, RetKind: view.RetObject, RetWrap: "obj.Wrap(%s)", RetZero: "nil",
				Outs: []view.DispatchOut{{GoName: "_out0", GoType: "PropertyListFormat", Zero: "0"}},
			},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Method(c.m)
			if err != nil {
				t.Fatal(err)
			}
			checkGolden(t, c.name, got)
		})
	}
}

// TestMainThreadGolden locks the @MainActor wrapping across the method, slice,
// and boolnserror templates: the body is lifted into an inner closure run on the
// main thread via purego.Main, its results captured and returned. The void case
// needs no capture; the value+error case captures into pre-declared locals.
func TestMainThreadGolden(t *testing.T) {
	t.Run("method_object", func(t *testing.T) {
		got, err := Method(view.Method{
			Recv: "(x *NSView) ", GoName: "Window", RetSig: " obj.Object",
			Dispatch: view.Dispatch{
				Call:    `objc.Send[objc.ID](objref.IDOf(x), objc.RegisterName("window"))`,
				RetKind: view.RetObject, RetWrap: "obj.Wrap(%s)", RetZero: "nil",
			},
			MainThread: true, RetVars: []string{"_mainthread0"}, RetTypes: []string{"obj.Object"},
		})
		if err != nil {
			t.Fatal(err)
		}
		checkGolden(t, "method_mainthread_object", got)
	})

	t.Run("method_void", func(t *testing.T) {
		got, err := Method(view.Method{
			Recv: "(x *NSView) ", GoName: "Layout",
			Dispatch:   view.Dispatch{Call: `objc.Send[objc.ID](objref.IDOf(x), objc.RegisterName("layout"))`, RetKind: view.RetVoid},
			MainThread: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkGolden(t, "method_mainthread_void", got)
	})

	t.Run("slice", func(t *testing.T) {
		got, err := SliceMethod(view.SliceMethod{
			Recv: "(x *NSView) ", GoName: "Subviews", RecvExpr: "objref.IDOf(x)",
			Selector: "subviews", ElemGoType: "obj.Object",
			ConvClosure: "func(_id objc.ID) obj.Object { return obj.Wrap(_id) }",
			MainThread:  true,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkGolden(t, "slice_mainthread", got)
	})

	t.Run("boolnserror", func(t *testing.T) {
		got, err := BoolNSErrorMethod(view.BoolNSErrorMethod{
			Recv: "(x *NSDocument) ", GoName: "Save", RecvExpr: "objref.IDOf(x)",
			Selector: "saveWithError:", MainThread: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		checkGolden(t, "boolnserror_mainthread", got)
	})
}

// TestAsyncMethodGolden locks both async shapes (error-only and typed result).
func TestAsyncMethodGolden(t *testing.T) {
	got, err := AsyncMethod(view.AsyncMethod{
		Recv: "(x *Machine) ", GoName: "Start", ParamStr: "ctx context.Context",
		ClosureParams: []string{"_p0 objc.ID"},
		SendCall:      `objc.Send[objc.ID](objref.IDOf(x), objc.RegisterName("startWithCompletionHandler:"), _block)`,
		ErrConvExpr:   "errkit.FromObjC(purego.NSErrorToError(_p0))",
	})
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "async_erroronly", got)

	got, err = AsyncMethod(view.AsyncMethod{
		Recv: "(x *Loader) ", GoName: "Load", ParamStr: "ctx context.Context",
		HasResult: true, ResultGoType: "obj.Object",
		ClosureParams:  []string{"_p0 objc.ID", "_p1 objc.ID"},
		SendCall:       `objc.Send[objc.ID](objref.IDOf(x), objc.RegisterName("loadWithCompletionHandler:"), _block)`,
		ErrConvExpr:    "errkit.FromObjC(purego.NSErrorToError(_p1))",
		ResultConvExpr: "obj.Wrap(_p0)",
	})
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "async_result", got)
}

// TestSliceMethodGolden locks both slice shapes (with and without error).
func TestSliceMethodGolden(t *testing.T) {
	got, err := SliceMethod(view.SliceMethod{
		Recv: "(x *Array) ", GoName: "ComponentsSeparatedByString", RecvExpr: "objref.IDOf(x)",
		Selector: "componentsSeparatedByString:", ElemGoType: "string",
		ConvClosure: "func(_id objc.ID) string { return purego.GoString(_id) }",
	})
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "slice_plain", got)

	got, err = SliceMethod(view.SliceMethod{
		Recv: "(x *Manager) ", GoName: "ContentsOfDirectory", RecvExpr: "objref.IDOf(x)",
		Selector: "contentsOfDirectoryAtPath:error:", ElemGoType: "string", HasError: true,
		ConvClosure: "func(_id objc.ID) string { return purego.GoString(_id) }",
	})
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "slice_error", got)
}

// TestBoolNSErrorMethodGolden locks the BOOL+NSError → error template.
func TestBoolNSErrorMethodGolden(t *testing.T) {
	got, err := BoolNSErrorMethod(view.BoolNSErrorMethod{
		Recv: "(x *Machine) ", GoName: "Validate", RecvExpr: "objref.IDOf(x)",
		Selector: "validateWithError:",
	})
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "boolnserror", got)
}

// TestConstantsGolden locks the constant accessor template across its three
// kinds (CF reference, Foundation *String, and non-Foundation objc.ID string).
func TestConstantsGolden(t *testing.T) {
	constants := []view.Constant{
		{GoName: "KSecClass", ExternName: "kSecClass", Kind: view.ConstCFRef},
		{GoName: "NSStringTransformLatinToKatakana", ExternName: "NSStringTransformLatinToKatakana", Kind: view.ConstNSString},
		{GoName: "NSPasteboardTypeString", ExternName: "NSPasteboardTypeString", CommentBlock: "// NSPasteboardTypeString is the UTF-8 plain-text type.\n", Kind: view.ConstObjcID},
	}
	got, err := Constants(constants)
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "constants", got)
}

// TestSentinelsGolden locks the error-sentinel template.
func TestSentinelsGolden(t *testing.T) {
	sentinels := []view.ErrorSentinel{
		{GoName: "ErrInternalError", CommentBlock: "// ErrInternalError matches the Virtualization error VZErrorInternalError.\n", Domain: "VZErrorDomain", Code: "2"},
	}
	got, err := Sentinels(sentinels)
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "sentinels", got)
}

// TestFuncsGolden locks the C-function wrapper template across its kinds:
// generic scalar, OSStatus→error, and CFErrorRef→error (comment-first).
func TestFuncsGolden(t *testing.T) {
	funcs := []view.Func{
		{
			GoName: "AudioChannelLayoutTagGetNumberOfChannels", CName: "AudioChannelLayoutTag_GetNumberOfChannels",
			VarName:     "_fnAudioChannelLayoutTagGetNumberOfChannels",
			CommentLine: "// AudioChannelLayoutTagGetNumberOfChannels calls the CoreAudioTypes framework function AudioChannelLayoutTag_GetNumberOfChannels.\n",
			ABIParams:   []string{"uint32"}, ABIRet: "uint32",
			SigParams: []string{"inLayoutTag uint32"}, RetSig: " uint",
			Kind: view.FuncScalar,
			Call: "_fnAudioChannelLayoutTagGetNumberOfChannels(inLayoutTag)",
		},
		{
			GoName: "SecItemDelete", CName: "SecItemDelete", VarName: "_fnSecItemDelete",
			CommentLine: "// SecItemDelete reports an error if the Security framework function SecItemDelete fails.\n",
			ABIParams:   []string{"objc.ID"}, ABIRet: "int32",
			SigParams: []string{"query obj.Object"}, RetSig: " error",
			Kind: view.FuncOSStatus,
			Call: "_fnSecItemDelete(objref.IDOf(query))",
			FailRet: "_err", OkRet: "nil",
		},
		{
			GoName: "CFBundlePreflightExecutable", CName: "CFBundlePreflightExecutable", VarName: "_fnCFBundlePreflightExecutable",
			CommentLine:  "// CFBundlePreflightExecutable reports an error if the CoreFoundation framework function CFBundlePreflightExecutable fails.\n",
			CommentFirst: true,
			ABIParams:    []string{"objc.ID", "unsafe.Pointer"}, ABIRet: "bool",
			SigParams: []string{"bundle obj.Object"}, RetSig: " error",
			Kind:     view.FuncCFErrorBool,
			PreLines: []string{"var _cfErr unsafe.Pointer"},
			Call:     "_fnCFBundlePreflightExecutable(objref.IDOf(bundle), unsafe.Pointer(&_cfErr))",
			Fail:     "!_ok",
		},
	}
	got, err := Funcs(funcs)
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "cfuncs", got)
}

// TestStructsGolden locks the value-struct template, including a documented
// struct and multi-field ordering.
func TestStructsGolden(t *testing.T) {
	structs := []view.Struct{
		{
			GoName: "CGPoint",
			Doc:    "CGPoint is a point in a two-dimensional coordinate system.",
			Fields: []view.Field{
				{GoName: "X", GoType: "float64"},
				{GoName: "Y", GoType: "float64"},
			},
		},
		{
			GoName: "CGRect",
			Fields: []view.Field{
				{GoName: "Origin", GoType: "CGPoint"},
				{GoName: "Size", GoType: "CGSize"},
			},
		},
	}
	got, err := Structs(structs)
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "structs", got)
}
