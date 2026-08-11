// Curated vmnet + block-bridge smoke tests — hand-maintained, never regenerated.
//
// These validate the two codegen capabilities introduced for C-function
// frameworks: exported snake_case functions and real block-object bridging
// (objc.NewBlock adapters) for completion handlers.
//
// The full interface start/stop test needs the com.apple.vm.networking
// entitlement or root, so it is double-gated. Run it manually with:
//
//	sudo VMNET_SMOKE=1 go test ./acceptance/ -run TestCurated_Vmnet_StartStopInterface -v
//
//go:build darwin

package acceptance_test

import (
	"os"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/frameworks/foundation"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/frameworks/vmnet"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/dispatch"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/xpc"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

// TestCurated_BlockAdapter_CompletionHandler proves the generated block
// adapters work at runtime: the Go closure is wrapped in a real ObjC block
// (objc.NewBlock with the leading objc.Block parameter) and invoked by the
// framework. Before the adapter fix every generated block-taking method
// panicked with "A Block implementation must take a Block as its first
// argument".
func TestCurated_BlockAdapter_CompletionHandler(t *testing.T) {
	runIsolated(t, "curated:block-adapter NSOperationQueue.addOperationWithBlock", func(t *testing.T) {
		queueID := objc.ID(objc.GetClass("NSOperationQueue")).Send(objc.RegisterName("new"))
		if queueID == 0 {
			t.Fatal("could not create NSOperationQueue")
		}
		queue := foundation.NSOperationQueueFromID(queueID)

		done := make(chan struct{})
		queue.AddOperationWith(func() {
			close(done)
		})

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("block was never invoked")
		}
	})
}

// TestCurated_Vmnet_NetworkConfigurationCreate exercises Dlopen, symbol
// registration, enum params, and pointer out-params for the vmnet framework
// without needing entitlements: creating a network configuration object does
// not touch the network stack.
func TestCurated_Vmnet_NetworkConfigurationCreate(t *testing.T) {
	runIsolated(t, "curated:vmnet.vmnet_network_configuration_create non-nil", func(t *testing.T) {
		if !vmnet.SymbolAvailable("vmnet_network_configuration_create") {
			t.Skip("vmnet_network_configuration_create not available on this macOS release")
		}
		var status vmnet.Vmnet_return_t
		config := vmnet.VmnetNetworkConfigurationCreate(vmnet.VMNET_SHARED_MODE, &status)
		if config == nil {
			t.Fatalf("vmnet_network_configuration_create returned nil (status %v)", status)
		}
	})
}

// TestCurated_Vmnet_StartStopInterface starts a shared-mode vmnet interface,
// asserts the async completion handler fires with VMNET_SUCCESS, and stops it
// again. Requires root (or the com.apple.vm.networking entitlement) — see the
// file header for the manual invocation.
func TestCurated_Vmnet_StartStopInterface(t *testing.T) {
	if os.Getenv("VMNET_SMOKE") != "1" {
		t.Skip("set VMNET_SMOKE=1 (and run as root) to enable the vmnet interface smoke test")
	}
	if os.Geteuid() != 0 {
		t.Skip("vmnet_start_interface requires root or the com.apple.vm.networking entitlement")
	}
	if !vmnet.SymbolAvailable("vmnet_start_interface") {
		t.Skip("vmnet_start_interface not available on this macOS release")
	}

	// Interface description: { vmnet_operation_mode_key: VMNET_SHARED_MODE }.
	dict := xpc.Xpc_dictionary_create(nil, nil, 0)
	if dict == nil {
		t.Fatal("xpc_dictionary_create returned nil")
	}
	modeKey := derefCStringVar(vmnet.Vmnet_operation_mode_key())
	if modeKey == "" {
		t.Fatal("could not read vmnet_operation_mode_key")
	}
	xpc.Xpc_dictionary_set_uint64(dict, modeKey, uint64(vmnet.VMNET_SHARED_MODE))

	queuePtr := dispatch.Dispatch_queue_create("vmnet.smoke", nil)
	if queuePtr == nil {
		t.Fatal("dispatch_queue_create returned nil")
	}

	desc := foundation.NSObjectFromID(objc.ID(uintptr(dict)))
	queue := foundation.NSObjectFromID(objc.ID(uintptr(queuePtr)))

	started := make(chan vmnet.Vmnet_return_t, 1)
	iface := vmnet.VmnetStartInterface(desc, queue, func(status vmnet.Vmnet_return_t, params *foundation.NSObject) {
		started <- status
	})

	select {
	case status := <-started:
		if status != vmnet.VMNET_SUCCESS {
			t.Fatalf("vmnet_start_interface completion status = %v, want VMNET_SUCCESS", status)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("vmnet_start_interface completion handler never fired")
	}
	if iface == nil {
		t.Fatal("vmnet_start_interface returned nil interface_ref")
	}

	stopped := make(chan vmnet.Vmnet_return_t, 1)
	if rc := vmnet.VmnetStopInterface(iface, queue, func(status vmnet.Vmnet_return_t) {
		stopped <- status
	}); rc != vmnet.VMNET_SUCCESS {
		t.Fatalf("vmnet_stop_interface = %v, want VMNET_SUCCESS", rc)
	}
	select {
	case status := <-stopped:
		if status != vmnet.VMNET_SUCCESS {
			t.Fatalf("vmnet_stop_interface completion status = %v, want VMNET_SUCCESS", status)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("vmnet_stop_interface completion handler never fired")
	}

	runtime.KeepAlive(desc)
	runtime.KeepAlive(queue)
}

// derefCStringVar reads a `const char *` global through the address returned
// by a generated extern accessor.
func derefCStringVar(addr uintptr) string {
	if addr == 0 {
		return ""
	}
	return purego.GoCString(*(*uintptr)(unsafe.Pointer(addr))) //nolint:govet // dlsym-provided address of a global
}
