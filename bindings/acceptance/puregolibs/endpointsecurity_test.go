//go:build darwin

package puregolibs_test

import (
	"reflect"
	"testing"
	"unsafe"

	endpointsecurity "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/endpointsecurity"
)

// Values measured with sizeof/_Alignof/offsetof against Xcode 27.0 arm64
// headers. The two unions must be inline arrays with the C size and alignment.
func TestEndpointSecurity_SDK27Layouts(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		size  uintptr
	}{
		{"es_event_bootstrap_check_in_t", endpointsecurity.EsEventBootstrapCheckInT{}, 56},
		{"es_event_bootstrap_look_up_t", endpointsecurity.EsEventBootstrapLookUpT{}, 120},
		{"es_lightweight_code_requirement_t", endpointsecurity.EsLightweightCodeRequirementT{}, 32},
		{"es_process_t", endpointsecurity.EsProcessT{}, 248},
	} {
		t.Run(tc.name, func(t *testing.T) {
			typ := reflect.TypeOf(tc.value)
			if typ.Size() != tc.size || typ.Align() != 8 {
				t.Errorf("size/alignment = %d/%d, want %d/8", typ.Size(), typ.Align(), tc.size)
			}
		})
	}
	var lookup endpointsecurity.EsEventBootstrapLookUpT
	var process endpointsecurity.EsProcessT
	if unsafe.Offsetof(lookup.Target) != 64 || unsafe.Sizeof(lookup.Target) != 56 {
		t.Error("bootstrap target union must occupy 56 bytes at offset 64")
	}
	if unsafe.Offsetof(process.Field17) != 212 || unsafe.Sizeof(process.Field17) != 16 {
		t.Error("process reserved union must occupy 16 bytes at offset 212")
	}
	if unsafe.Offsetof(process.Cdhash_full) != 232 {
		t.Errorf("full cdhash offset = %d, want 232", unsafe.Offsetof(process.Cdhash_full))
	}
}

// TestEndpointSecurity_NewClientBlock exercises the block adapter (the tier's
// reason to exist): es_new_client takes a Go handler func, which the purego
// backend wraps in an objc.NewBlock before the call. Creating a client needs
// the Endpoint Security entitlement no test process carries, so success is not
// expected — what this proves is that constructing the block, passing it
// across the ABI, and returning a well-defined EsNewClientResultT all work
// without crashing. A missing/broken adapter would segfault here instead.
func TestEndpointSecurity_NewClientBlock(t *testing.T) {
	if !endpointsecurity.SymbolAvailable("es_new_client") {
		t.Skip("es_new_client did not bind")
	}
	called := false
	handler := func(client, message unsafe.Pointer) { called = true }

	var client unsafe.Pointer
	res := endpointsecurity.Es_new_client(unsafe.Pointer(&client), handler)

	// Without the entitlement the kernel refuses the client; any of the
	// well-defined error results is fine — a corrupted block would not return
	// a clean enum.
	switch res {
	case endpointsecurity.ES_NEW_CLIENT_RESULT_SUCCESS:
		// Unexpected in an unentitled process, but valid — clean up.
		endpointsecurity.Es_delete_client(client)
	case endpointsecurity.ES_NEW_CLIENT_RESULT_ERR_NOT_ENTITLED,
		endpointsecurity.ES_NEW_CLIENT_RESULT_ERR_NOT_PERMITTED,
		endpointsecurity.ES_NEW_CLIENT_RESULT_ERR_NOT_PRIVILEGED,
		endpointsecurity.ES_NEW_CLIENT_RESULT_ERR_INVALID_ARGUMENT:
		// Expected: the block crossed the ABI and the call returned cleanly.
	default:
		t.Errorf("es_new_client returned unexpected result %d", res)
	}
	_ = called
}
