//go:build darwin

package puregolibs_test

import (
	"testing"
	"unsafe"

	endpointsecurity "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/endpointsecurity"
)

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
