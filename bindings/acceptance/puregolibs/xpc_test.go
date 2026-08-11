//go:build darwin

package puregolibs_test

import (
	"testing"
	"unsafe"

	xpc "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/xpc"
)

// TestXPC_PrimitiveRoundTrips builds real XPC objects and reads their values
// back — scalar and string marshalling through libxpc, plus reference release.
func TestXPC_PrimitiveRoundTrips(t *testing.T) {
	xi := xpc.Xpc_int64_create(0x0D15EA5E)
	if xi == nil {
		t.Fatal("xpc_int64_create returned nil")
	}
	if got := xpc.Xpc_int64_get_value(xi); got != 0x0D15EA5E {
		t.Errorf("xpc int64 round-trip = %#x; want 0xD15EA5E", got)
	}
	xpc.Xpc_release(xi)

	xb := xpc.Xpc_bool_create(true)
	if !xpc.Xpc_bool_get_value(xb) {
		t.Error("xpc bool round-trip lost true")
	}
	xpc.Xpc_release(xb)

	xs := xpc.Xpc_string_create("weave")
	if xs == nil {
		t.Fatal("xpc_string_create returned nil")
	}
	if got := xpc.Xpc_string_get_string_ptr(xs); got != "weave" {
		t.Errorf("xpc string round-trip = %q; want \"weave\"", got)
	}
	xpc.Xpc_release(xs)
}

// TestXPC_DictionaryStoresValues stores an int64 in a dictionary and reads it
// back by key — string key marshalling plus object graph handling.
func TestXPC_DictionaryStoresValues(t *testing.T) {
	d := xpc.Xpc_dictionary_create(nil, nil, 0)
	if d == nil {
		t.Fatal("xpc_dictionary_create returned nil")
	}
	defer xpc.Xpc_release(d)

	xpc.Xpc_dictionary_set_int64(d, "answer", 42)
	if got := xpc.Xpc_dictionary_get_int64(d, "answer"); got != 42 {
		t.Errorf("xpc_dictionary_get_int64 = %d; want 42", got)
	}
}

// TestXPC_ArrayApplyBlock is the block-adapter proof for xpc: xpc_array_apply
// invokes the applier block once per element. The block must be built by
// objc.NewBlock, cross into libxpc, and be called back with each element —
// counting invocations proves the callback path works end to end.
func TestXPC_ArrayApplyBlock(t *testing.T) {
	arr := xpc.Xpc_array_create(nil, 0)
	if arr == nil {
		t.Fatal("xpc_array_create returned nil")
	}
	defer xpc.Xpc_release(arr)

	// Note: an empty array applies the block zero times but still exercises the
	// block construction + ABI crossing; populate it so the callback fires.
	for i := 0; i < 3; i++ {
		xpc.Xpc_array_append_value(arr, xpc.Xpc_int64_create(int64(i)))
	}
	if got := xpc.Xpc_array_get_count(arr); got != 3 {
		t.Fatalf("xpc_array_get_count = %d; want 3", got)
	}

	seen := 0
	ok := xpc.Xpc_array_apply(arr, func(index uint64, value unsafe.Pointer) bool {
		seen++
		return true // continue
	})
	if !ok {
		t.Error("xpc_array_apply returned false (iteration did not complete)")
	}
	if seen != 3 {
		t.Errorf("applier block called %d times; want 3", seen)
	}
}
