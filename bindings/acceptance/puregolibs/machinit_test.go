//go:build darwin

package puregolibs_test

import (
	"testing"

	machinit "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machinit"
)

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
