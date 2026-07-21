// Curated acceptance tests for the performance-telemetry surfaces: the mach
// host C libraries (machhost/machinit/machvm/machtime), the IOKit
// PowerSources sub-framework, and the private IOReport library. These are
// hand-maintained, never regenerated.
//
//go:build darwin

package acceptance_test

import (
	"testing"
	"unsafe"

	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/frameworks/powersources"
	machhostraw "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machhost"
	machinitraw "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machinit"
	machtimeraw "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machtime"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/libraries/ioreport"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/libraries/machinit"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/libraries/machtime"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/libraries/machvm"
)

// --- mach host ---------------------------------------------------------------

const (
	hostVMInfo64         = 4 // HOST_VM_INFO64
	hostVMInfo64Count    = 38
	processorCPULoadInfo = 2 // PROCESSOR_CPU_LOAD_INFO
	cpuStateMax          = 4
)

func TestCurated_MachHost_Statistics64(t *testing.T) {
	runIsolated(t, "curated:machhost host_statistics64 returns vm stats", func(t *testing.T) {
		host := machinit.HostSelf()
		if host == 0 {
			t.Fatal("mach_host_self returned 0")
		}
		// vm_statistics64 is 38 natural_t-sized words; free_count is the first,
		// wire_count the fourth.
		var buf [hostVMInfo64Count]int32
		count := uint32(hostVMInfo64Count)
		if rc := machhostraw.Host_statistics64(host, hostVMInfo64, &buf[0], &count); rc != 0 {
			t.Fatalf("host_statistics64 = %d; want 0", rc)
		}
		free := uint32(buf[0])
		wire := uint32(buf[3])
		if free == 0 && wire == 0 {
			t.Error("host_statistics64 returned all-zero free and wired page counts")
		}
	})
}

func TestCurated_MachHost_ProcessorInfo_AndVMDeallocate(t *testing.T) {
	runIsolated(t, "curated:machhost host_processor_info per-core ticks", func(t *testing.T) {
		host := machinit.HostSelf()
		var (
			cpuCount uint32
			info     uintptr
			infoCnt  uint32
		)
		rc := machhostraw.Host_processor_info(
			host, processorCPULoadInfo, &cpuCount,
			unsafe.Pointer(&info), &infoCnt,
		)
		if rc != 0 {
			t.Fatalf("host_processor_info = %d; want 0", rc)
		}
		if cpuCount == 0 || infoCnt < cpuCount*cpuStateMax {
			t.Fatalf("host_processor_info: cpus=%d infoCnt=%d", cpuCount, infoCnt)
		}
		infoPtr := *(*unsafe.Pointer)(unsafe.Pointer(&info))
		ticks := unsafe.Slice((*uint32)(infoPtr), int(infoCnt))
		var total uint64
		for _, v := range ticks {
			total += uint64(v)
		}
		if total == 0 {
			t.Error("host_processor_info returned all-zero tick counts")
		}
		// The out array is vm_allocated in this task and must be released.
		if err := machvm.Deallocate(machinitraw.Mach_task_self_, uint64(info), uint64(infoCnt)*4); err != nil {
			t.Errorf("vm_deallocate: %v", err)
		}
	})
}

func TestCurated_MachTime_AbsoluteTime(t *testing.T) {
	runIsolated(t, "curated:machtime mach_absolute_time > 0", func(t *testing.T) {
		if machtime.AbsoluteTime() == 0 {
			t.Error("mach_absolute_time returned 0")
		}
	})
}

func TestCurated_MachTime_TimebaseInfo(t *testing.T) {
	runIsolated(t, "curated:machtime mach_timebase_info numer/denom > 0", func(t *testing.T) {
		var tb machtimeraw.Mach_timebase_info
		if err := machtime.TimebaseInfo(&tb); err != nil {
			t.Fatalf("mach_timebase_info: %v", err)
		}
		if tb.Numer == 0 || tb.Denom == 0 {
			t.Errorf("timebase = %d/%d; want both > 0", tb.Numer, tb.Denom)
		}
	})
}

// --- IOKit power sources -----------------------------------------------------

func TestCurated_PowerSources_CopyInfo(t *testing.T) {
	runIsolated(t, "curated:IOPSCopyPowerSourcesInfo non-nil blob", func(t *testing.T) {
		blob := powersources.IOPSCopyPowerSourcesInfo()
		if blob == nil {
			t.Fatal("IOPSCopyPowerSourcesInfo returned nil")
		}
		// The list may be empty on desktops, but the call itself must succeed.
		list := powersources.IOPSCopyPowerSourcesList(blob)
		if list == nil {
			t.Fatal("IOPSCopyPowerSourcesList returned nil")
		}
	})
}

// --- IOReport (private) ------------------------------------------------------

func TestCurated_IOReport_CopyAllChannels(t *testing.T) {
	runIsolated(t, "curated:IOReportCopyAllChannels non-nil", func(t *testing.T) {
		channels := ioreport.IOReportCopyAllChannels(0, 0)
		if channels == nil {
			t.Skip("IOReport returned no channels (private API unavailable on this system)")
		}
	})
}
