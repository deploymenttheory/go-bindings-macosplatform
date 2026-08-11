//go:build darwin

package puregolibs_test

import (
	"testing"
	"unsafe"

	machhost "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machhost"
	machinit "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machinit"
)

// TestMachhost_Statistics64 reads live VM statistics from the kernel — the
// same call the curated acceptance test makes, now through the purego
// backend.
func TestMachhost_Statistics64(t *testing.T) {
	const hostVMInfo64 = 4
	host := machinit.Mach_host_self()
	if host == 0 {
		t.Fatal("mach_host_self() = 0")
	}
	var out [38]int32
	count := uint32(len(out))
	if rc := machhost.Host_statistics64(host, hostVMInfo64, &out[0], &count); rc != 0 {
		t.Fatalf("host_statistics64 rc = %d", rc)
	}
	freePages, wirePages := out[0], out[6]
	if freePages == 0 && wirePages == 0 {
		t.Error("host_statistics64 returned all-zero free and wired page counts")
	}
}

// TestMachhost_ZoneInfoByValue passes the 80-byte mach_zone_name_t by value
// (as the raw surface's pointer form, which IS the arm64 ABI representation
// of a >16-byte composite). The call needs host privileges most setups lack —
// what this asserts is that the crossing itself is sound: a well-defined
// kern_return_t comes back instead of a corrupted-stack crash.
func TestMachhost_ZoneInfoByValue(t *testing.T) {
	// mach_zone_name_t is char mzn_name[80]; mach_zone_info_t is a run of
	// uint64 counters — a generous raw buffer stands in for both.
	var name [80]byte
	copy(name[:], "kalloc")
	var info [128]byte
	rc := machhost.Mach_zone_info_for_zone(
		machinit.Mach_host_self(),
		unsafe.Pointer(&name),
		unsafe.Pointer(&info),
	)
	t.Logf("mach_zone_info_for_zone rc = %d (non-zero expected without host privileges)", rc)
}
