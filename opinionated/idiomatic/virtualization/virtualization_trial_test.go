//go:build darwin

package virtualization_test

import (
	"errors"
	"testing"

	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego/objcerrors"
	trial "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/virtualization"
)

// T1: Plain +new constructor creates a non-nil VirtualMachineConfiguration.
func TestTrialNewVirtualMachineConfiguration(t *testing.T) {
	cfg := trial.NewVirtualMachineConfiguration()
	if cfg == nil {
		t.Fatal("NewVirtualMachineConfiguration returned nil")
	}
	if cfg.Unwrap() == nil {
		t.Fatal("Unwrap returned nil inner")
	}
}

// T2: Scalar fluent setters chain and round-trip through the raw object.
func TestTrialWithMemorySizeAndCPUCount(t *testing.T) {
	const wantMem = 2 * 1024 * 1024 * 1024
	const wantCPU = uint(2)

	cfg := trial.NewVirtualMachineConfiguration().
		WithMemorySize(wantMem).
		WithCPUCount(wantCPU)

	if cfg == nil {
		t.Fatal("fluent chain returned nil")
	}

	inner := cfg.Unwrap()
	if got := inner.MemorySize(); got != wantMem {
		t.Errorf("MemorySize: got %d, want %d", got, wantMem)
	}
	if got := inner.CPUCount(); got != wantCPU {
		t.Errorf("CPUCount: got %d, want %d", got, wantCPU)
	}
}

// T3: Validate returns a non-nil error when no boot loader is configured.
func TestTrialValidate_FailsWithoutBootLoader(t *testing.T) {
	err := trial.NewVirtualMachineConfiguration().
		WithMemorySize(2 * 1024 * 1024 * 1024).
		WithCPUCount(2).
		Validate()

	if err == nil {
		t.Fatal("expected validation error with no boot loader, got nil")
	}
	t.Logf("validation error (expected): %v", err)
}

// T4: alloc+initWithKernelURL: bundles correctly; NSURL is constructed from Go string.
func TestTrialNewLinuxBootLoaderWithKernelURL(t *testing.T) {
	bl := trial.NewLinuxBootLoaderWithKernelURL("/nonexistent/vmlinuz").
		WithCommandLine("root=/dev/vda console=hvc0").
		WithInitialRamdiskURL("/nonexistent/initrd.img")

	if bl == nil {
		t.Fatal("NewLinuxBootLoaderWithKernelURL returned nil")
	}
	if bl.Unwrap() == nil {
		t.Fatal("Unwrap returned nil inner")
	}
	// The ObjC NSURL object was constructed — KernelURL must be non-nil.
	if bl.Unwrap().KernelURL() == nil {
		t.Error("KernelURL is nil; NSURL was not constructed from the Go string")
	}
}

// T5: Plain +new constructors work for classes with no explicit init.
func TestTrialPlainNewConstructors(t *testing.T) {
	efi := trial.NewEFIBootLoader()
	if efi == nil || efi.Unwrap() == nil {
		t.Error("NewEFIBootLoader: got nil")
	}

	nat := trial.NewNATNetworkDeviceAttachment()
	if nat == nil || nat.Unwrap() == nil {
		t.Error("NewNATNetworkDeviceAttachment: got nil")
	}

	net := trial.NewVirtioNetworkDeviceConfiguration()
	if net == nil || net.Unwrap() == nil {
		t.Error("NewVirtioNetworkDeviceConfiguration: got nil")
	}
}

// T6: WithNetworkDevices variadic setter builds an NSArray; NetworkDevices reads it back.
func TestTrialWithNetworkDevices_Roundtrip(t *testing.T) {
	// Trial types implement NetworkDeviceConfigurationProvider — no raw unwrapping needed.
	netDev := trial.NewVirtioNetworkDeviceConfiguration()

	cfg := trial.NewVirtualMachineConfiguration().
		WithNetworkDevices(netDev)

	devices := cfg.NetworkDevices()
	if len(devices) != 1 {
		t.Fatalf("NetworkDevices: got %d items, want 1", len(devices))
	}
}

// T7: Error-returning alloc+init path: non-existent file produces a structured ObjCError.
func TestTrialNewDiskImageStorageDeviceAttachment_FileNotFound(t *testing.T) {
	_, err := trial.NewDiskImageStorageDeviceAttachmentWithURLReadOnlyError(
		"/nonexistent/disk.img", true,
	)
	if err == nil {
		t.Fatal("expected error for non-existent disk image, got nil")
	}

	// The error string must now include domain and code — not just a bare message.
	t.Logf("error string: %v", err)

	// Structured access via errors.As — no information loss.
	var vzErr *objcerrors.ObjCError
	if !errors.As(err, &vzErr) {
		t.Fatal("expected *objcerrors.ObjCError, errors.As returned false")
	}
	if vzErr.Domain == "" {
		t.Error("ObjCError.Domain is empty")
	}
	if vzErr.Description == "" {
		t.Error("ObjCError.Description is empty")
	}
	t.Logf("domain=%q code=%d description=%q failureReason=%q recoverySuggestion=%q",
		vzErr.Domain, vzErr.Code, vzErr.Description, vzErr.FailureReason, vzErr.RecoverySuggestion)
}

// T8: Full idiomatic fluent chain — construction only, no VM start.
func TestTrialFullFluentChain(t *testing.T) {
	bl := trial.NewLinuxBootLoaderWithKernelURL("/boot/vmlinuz").
		WithCommandLine("root=/dev/vda")

	// Trial types implement BootLoaderProvider — pass directly without unwrapping.
	cfg := trial.NewVirtualMachineConfiguration().
		WithBootLoader(bl).
		WithMemorySize(2 * 1024 * 1024 * 1024).
		WithCPUCount(2)

	if cfg == nil {
		t.Fatal("full fluent chain returned nil configuration")
	}

	// Validate will fail because /boot/vmlinuz doesn't exist, but the ObjC
	// objects were all constructed and wired together correctly.
	err := cfg.Validate()
	if err == nil {
		t.Log("validate passed (unexpected but not fatal — kernel file exists)")
	} else {
		t.Logf("validate error (expected in test env): %v", err)
	}

	// Verify the scalar properties round-tripped.
	inner := cfg.Unwrap()
	if got := inner.MemorySize(); got != 2*1024*1024*1024 {
		t.Errorf("MemorySize: got %d, want 2GiB", got)
	}
	if got := inner.CPUCount(); got != 2 {
		t.Errorf("CPUCount: got %d, want 2", got)
	}
	// Boot loader was set — should be non-nil.
	if inner.BootLoader() == nil {
		t.Error("BootLoader is nil after WithBootLoader")
	}

}
