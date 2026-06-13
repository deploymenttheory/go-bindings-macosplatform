//go:build darwin

package main

import (
	"strings"
)

// Category describes how a function can be acceptance-tested automatically.
type Category int

const (
	// CatZeroArgFactory — zero-arg class method returning a non-nullable pointer.
	// Call it and assert non-nil. Examples: NSArray.+[array], NSString.+[string].
	CatZeroArgFactory Category = iota

	// CatSingleton — zero-arg class method whose return is nullable or whose
	// selector name suggests a shared/default instance. nil is acceptable.
	// Examples: NSApplication.+[sharedApplication], NSScreen.+[mainScreen].
	CatSingleton

	// CatZeroArgScalar — zero-arg method or function returning a scalar or void.
	// Call it and assert it does not crash. Void returns are included here.
	CatZeroArgScalar

	// CatSkip — not auto-testable (instance methods, tuple returns, etc.).
	CatSkip
)

// RetKind classifies the return type of a generated function.
type RetKind int

const (
	RetVoid    RetKind = iota
	RetPointer         // *T, unsafe.Pointer, id, instancetype
	RetScalar          // int, uint, float, bool, string, named enum
	RetTuple           // (T, error) — not auto-testable without unwrapping
)

// FuncRecord holds everything the emitter needs to generate one acceptance test.
type FuncRecord struct {
	ID        string // "ID: objc-sym Foundation.NSArray.+[array]"
	Framework string // "Foundation"
	Selector  string // "array" / "CFNullGetTypeID"

	GoFuncName      string // "NSArrayArray"
	ZeroArgs        bool   // always true (we only sample zero-arg functions)
	RetKind         RetKind
	GoPackage       string // "foundation"
	GoImportPath    string // "github.com/.../bindings/frameworks/foundation"
	NeedsMainThread bool   // true for AppKit and other UI frameworks
	IsCFunction     bool   // free C function (Selector holds the C symbol)

	Category Category
	TestName string // derived from ID
}

// mainThreadFrameworks is the set of frameworks whose zero-arg class methods
// must be called on the OS main thread.
var mainThreadFrameworks = map[string]bool{
	"AppKit": true,
}

// skippedFrameworks is the set of frameworks excluded from test generation.
// Each framework is excluded for a specific environmental reason:
//
//   - Tcl/Tk: destructive void functions (Tcl_FinalizeThread, Tcl_Finalize)
//     tear down global interpreter state and cause SIGSEGV in test processes.
//   - BackgroundAssets: BADownloadManager.sharedManager requires Background
//     Assets entitlements and kills the process without them.
//   - DiscRecordingUI: requires a disc burner and a running AppKit app context.
//   - DriverKit: OSArray/OSDictionary require kernel-extension entitlements.
//   - Photos: PHAssetCreationRequest.creationRequestForAsset requires being
//     called inside a PHPhotoLibrary.performChanges block; segfaults otherwise.
//   - QuickLookUI: QLPreviewPanel.sharedPreviewPanel requires a running AppKit
//     application with a display connection.
//   - SafetyKit: SACrashDetectionManager.isAvailable requires iPhone hardware.
//   - SecurityInterface: SFKeychainSettingsPanel requires a live display/UI context.
//   - UserNotifications: UNUserNotificationCenter.currentNotificationCenter
//     requires a running app/daemon notification context; segfaults otherwise.
//   - WebKit: WKContentWorld.pageWorld requires the web content process to be running.
//   - OpenCL: gcl_* dispatch functions require a live OpenCL context/queue;
//     calling them blind segfaults (framework is also deprecated).
//   - GLUT: glut* functions require glutInit and a live GL context/display;
//     calling them blind segfaults.
//   - Ruby: rb_* functions require an initialized Ruby VM (ruby_init);
//     calling them blind segfaults or blocks forever (rb_thread_sleep_deadly).
var skippedFrameworks = map[string]bool{
	"Tcl":               true,
	"Tk":                true,
	"BackgroundAssets":  true,
	"DiscRecordingUI":   true,
	"DriverKit":         true,
	"Photos":            true,
	"QuickLookUI":       true,
	"SafetyKit":         true,
	"SecurityInterface": true,
	"UserNotifications": true,
	"WebKit":            true,
	"OpenCL":            true,
	"Ruby":              true,
	"GLUT":              true,
}

// isSkippedFunction excludes individual C functions that crash when called
// blind even though their symbol binds. The deprecated Security Transforms
// API (SecTransformGetTypeID, SecEncryptTransformGetTypeID, …) dereferences
// internal state that no longer exists on macOS 26 and SIGSEGVs.
func isSkippedFunction(framework, name string) bool {
	return framework == "Security" && strings.Contains(name, "Transform")
}

// singletonPrefixes: if the selector starts with any of these, the return is
// treated as potentially nil (t.Skip rather than t.Error on nil return).
var singletonPrefixes = []string{
	"shared", "default", "current", "standard", "main",
	"system", "local", "global",
}

// isSingleton returns true when the selector name matches a singleton pattern.
func isSingleton(selector string) bool {
	lower := strings.ToLower(selector)
	for _, pfx := range singletonPrefixes {
		if strings.HasPrefix(lower, pfx) {
			return true
		}
	}
	return false
}

// testName converts an ID string to a valid Go test function name.
//
//	"ID: objc-sym Foundation.NSArray.+[array]" → "TestAccept_Foundation_NSArray_array"
//	"ID: objc-sym CoreFoundation.CFNullGetTypeID" → "TestAccept_CoreFoundation_CFNullGetTypeID"
func testName(id string) string {
	s := strings.TrimPrefix(id, "ID: objc-sym ")
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '.', '+', '-', '[', ']', ':', ' ':
			sb.WriteByte('_')
		default:
			sb.WriteRune(r)
		}
	}
	name := sb.String()
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	return "TestAccept_" + strings.Trim(name, "_")
}
