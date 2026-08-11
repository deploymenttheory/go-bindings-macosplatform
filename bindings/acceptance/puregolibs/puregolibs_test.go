//go:build darwin

package puregolibs_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	compression "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/compression"
	ioreport "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/ioreport"
	machinit "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machinit"
	machtime "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machtime"
	machvm "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machvm"
	sandbox "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/sandbox"
	xar "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/xar"
)

// TestSymbolsResolve proves each migrated library dlopened its dylib and bound
// its key symbols through purego — the direct per-library binding check.
func TestSymbolsResolve(t *testing.T) {
	cases := []struct {
		lib       string
		available func(string) bool
		symbol    string
	}{
		{"machtime", machtime.SymbolAvailable, "mach_absolute_time"},
		{"machinit", machinit.SymbolAvailable, "mach_host_self"},
		{"machvm", machvm.SymbolAvailable, "vm_allocate"},
		{"compression", compression.SymbolAvailable, "compression_encode_buffer"},
		{"sandbox", sandbox.SymbolAvailable, "sandbox_init"},
		{"xar", xar.SymbolAvailable, "xar_open"},
		{"ioreport", ioreport.SymbolAvailable, "IOReportCopyAllChannels"},
	}
	for _, c := range cases {
		if !c.available(c.symbol) {
			t.Errorf("%s: symbol %q did not bind", c.lib, c.symbol)
		}
	}
}

// TestMachtime_Monotonic makes real mach_absolute_time calls and checks the
// clock actually advances monotonically.
func TestMachtime_Monotonic(t *testing.T) {
	t1 := machtime.Mach_absolute_time()
	t2 := machtime.Mach_absolute_time()
	if t1 == 0 || t2 < t1 {
		t.Errorf("mach_absolute_time not monotonic: %d then %d", t1, t2)
	}
}

// TestMachtime_TimebaseInfo checks the out-parameter struct fill: the kernel
// writes numer/denom through the *MachTimebaseInfo pointer.
func TestMachtime_TimebaseInfo(t *testing.T) {
	var tb machtime.MachTimebaseInfo
	if rc := machtime.Mach_timebase_info(&tb); rc != 0 {
		t.Fatalf("mach_timebase_info rc = %d", rc)
	}
	if tb.Numer == 0 || tb.Denom == 0 {
		t.Errorf("timebase = %d/%d; want both > 0", tb.Numer, tb.Denom)
	}
}

// TestCompression_RoundTrip compresses a buffer with zlib and decompresses it
// back — proving pointer, length, and enum marshalling end to end: the output
// is only byte-identical if every parameter crossed the ABI correctly.
func TestCompression_RoundTrip(t *testing.T) {
	src := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog. "), 100)
	dst := make([]byte, len(src)+4096)

	encoded := compression.Compression_encode_buffer(
		&dst[0], uint64(len(dst)),
		&src[0], uint64(len(src)),
		nil, compression.COMPRESSION_ZLIB,
	)
	if encoded == 0 || encoded >= uint64(len(src)) {
		t.Fatalf("encode: %d bytes from %d input; want 0 < n < input (compressible data)", encoded, len(src))
	}

	back := make([]byte, len(src)+16)
	decoded := compression.Compression_decode_buffer(
		&back[0], uint64(len(back)),
		&dst[0], encoded,
		nil, compression.COMPRESSION_ZLIB,
	)
	if decoded != uint64(len(src)) {
		t.Fatalf("decode: %d bytes; want %d", decoded, len(src))
	}
	if !bytes.Equal(back[:decoded], src) {
		t.Fatal("round-trip corrupted the data")
	}
}

// TestCompression_ScratchSizes checks the enum-by-value marshalling on a
// simple scalar-returning call for every documented algorithm.
func TestCompression_ScratchSizes(t *testing.T) {
	for _, alg := range []compression.CompressionAlgorithm{
		compression.COMPRESSION_LZ4, compression.COMPRESSION_ZLIB,
	} {
		if compression.Compression_encode_scratch_buffer_size(alg) == 0 {
			t.Errorf("scratch buffer size for %v = 0; want > 0", alg)
		}
	}
}

// TestMachinit_TaskHostAndPageSize exercises the dereferenced-extern
// (mach_task_self_) and two live mach calls, checking against the only two
// page sizes macOS ships.
func TestMachinit_TaskHostAndPageSize(t *testing.T) {
	if machinit.Mach_task_self_ == 0 {
		t.Error("mach_task_self_ extern = 0; extern init did not run")
	}
	if rc := machinit.Mach_task_is_self(machinit.Mach_task_self_); rc != 1 {
		t.Errorf("mach_task_is_self(own task) = %d; want 1", rc)
	}
	host := machinit.Mach_host_self()
	if host == 0 {
		t.Fatal("mach_host_self() = 0")
	}
	var pageSize uint64
	if rc := machinit.Host_page_size(host, &pageSize); rc != 0 {
		t.Fatalf("host_page_size rc = %d", rc)
	}
	if pageSize != 4096 && pageSize != 16384 {
		t.Errorf("page size = %d; want 4096 or 16384", pageSize)
	}
}

// TestMachvm_AllocateWriteDeallocate allocates a real page in this task,
// writes through it, reads it back, and deallocates — out-parameter, write
// access, and rc handling all live.
func TestMachvm_AllocateWriteDeallocate(t *testing.T) {
	const vmFlagsAnywhere = 1
	task := machinit.Mach_task_self_
	var addr uint64
	if rc := machvm.Vm_allocate(task, &addr, 16384, vmFlagsAnywhere); rc != 0 {
		t.Fatalf("vm_allocate rc = %d", rc)
	}
	if addr == 0 {
		t.Fatal("vm_allocate returned address 0")
	}
	p := (*byte)(unsafe.Pointer(uintptr(addr)))
	*p = 0xAB
	if *p != 0xAB {
		t.Error("write through vm_allocate'd page did not stick")
	}
	if rc := machvm.Vm_deallocate(task, addr, 16384); rc != 0 {
		t.Errorf("vm_deallocate rc = %d", rc)
	}
}

// TestSandbox_InvalidProfileFails calls the real sandbox_init with a garbage
// profile name: a non-zero error return proves the string parameter reached
// libsandbox intact (without actually sandboxing the test process).
func TestSandbox_InvalidProfileFails(t *testing.T) {
	if rc := sandbox.Sandbox_init("no-such-profile-8f2d1", 0, nil); rc == 0 {
		t.Error("sandbox_init with a nonexistent named profile returned success")
	}
}

// TestXar_CreateAndReopen writes a real xar archive to disk, closes it,
// reopens it read-only, and checks the negative path — string params, handle
// returns, and int32 rc handling live.
func TestXar_CreateAndReopen(t *testing.T) {
	const (
		xarRead  = 0
		xarWrite = 1
	)
	path := filepath.Join(t.TempDir(), "t.xar")

	w := xar.Xar_open(path, xarWrite)
	if w == nil {
		t.Fatal("xar_open(WRITE) returned nil")
	}
	if rc := xar.Xar_close(w); rc != 0 {
		t.Fatalf("xar_close(write) rc = %d", rc)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("archive was not created: %v", err)
	}

	r := xar.Xar_open(path, xarRead)
	if r == nil {
		t.Fatal("xar_open(READ) on the just-written archive returned nil")
	}
	if rc := xar.Xar_close(r); rc != 0 {
		t.Errorf("xar_close(read) rc = %d", rc)
	}

	if bad := xar.Xar_open(filepath.Join(t.TempDir(), "missing", "no.xar"), xarRead); bad != nil {
		xar.Xar_close(bad)
		t.Error("xar_open on a nonexistent path returned a handle")
	}
}

// TestIOReport_CopyAllChannels calls the private IOReport library live. Some
// virtualised runners expose no channels; nil is a skip there, matching the
// curated acceptance test.
func TestIOReport_CopyAllChannels(t *testing.T) {
	if ptr := ioreport.IOReportCopyAllChannels(0, 0); ptr == nil {
		t.Skip("IOReportCopyAllChannels returned nil (no channels on this host)")
	}
}
