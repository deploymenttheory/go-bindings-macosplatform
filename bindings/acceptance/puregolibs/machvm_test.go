//go:build darwin

package puregolibs_test

import (
	"testing"
	"unsafe"

	machinit "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machinit"
	machvm "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machvm"
)

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
