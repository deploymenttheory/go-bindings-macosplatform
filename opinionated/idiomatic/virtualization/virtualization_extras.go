//go:build darwin

// Hand-crafted additions to the trial virtualization package.
// These cover accessors, class-method wrappers, and convenience constructors
// that the code generator does not yet produce automatically.
package virtualization

import (
	"context"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	raw "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/virtualization"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

// ── VirtualMachine state ───────────────────────────────────────────────────

// VirtualMachineState is a type alias for the underlying ObjC state enum so that
// callers can work with VM states without a direct raw package import.
type VirtualMachineState = raw.VZVirtualMachineState

// Re-exported state constants matching VZVirtualMachineState values.
const (
	VirtualMachineStateStopped  = raw.VZVirtualMachineStateStopped
	VirtualMachineStateRunning  = raw.VZVirtualMachineStateRunning
	VirtualMachineStatePaused   = raw.VZVirtualMachineStatePaused
	VirtualMachineStateError    = raw.VZVirtualMachineStateError
	VirtualMachineStateStarting = raw.VZVirtualMachineStateStarting
	VirtualMachineStateResuming = raw.VZVirtualMachineStateResuming
	VirtualMachineStatePausing  = raw.VZVirtualMachineStatePausing
	VirtualMachineStateStopping = raw.VZVirtualMachineStateStopping
)

// State returns the current lifecycle state of the virtual machine.
func (x *VirtualMachine) State() VirtualMachineState {
	return x.inner.State()
}

// CanStart reports whether the virtual machine can be started.
func (x *VirtualMachine) CanStart() bool { return x.inner.CanStart() }

// CanStop reports whether the virtual machine can be forcefully stopped.
func (x *VirtualMachine) CanStop() bool { return x.inner.CanStop() }

// CanPause reports whether the virtual machine can be paused.
func (x *VirtualMachine) CanPause() bool { return x.inner.CanPause() }

// CanResume reports whether the virtual machine can be resumed from pause.
func (x *VirtualMachine) CanResume() bool { return x.inner.CanResume() }

// CanRequestStop reports whether the virtual machine can be gracefully stopped.
func (x *VirtualMachine) CanRequestStop() bool { return x.inner.CanRequestStop() }

// ── MacHardwareModel accessors ────────────────────────────────────────────

// DataRepresentation returns the hardware model as an opaque byte slice
// suitable for serialisation and later reconstruction via NewMacHardwareModelFromBytes.
func (x *MacHardwareModel) DataRepresentation() []byte {
	ns := x.inner.DataRepresentation()
	if ns == nil {
		return nil
	}
	return nsDataToBytes(ns)
}

// IsSupported reports whether this hardware model is supported on the current host.
func (x *MacHardwareModel) IsSupported() bool { return x.inner.IsSupported() }

// ── MacMachineIdentifier accessors ────────────────────────────────────────

// DataRepresentation returns the machine identifier as an opaque byte slice
// suitable for serialisation and later reconstruction via NewMacMachineIdentifierFromBytes.
func (x *MacMachineIdentifier) DataRepresentation() []byte {
	ns := x.inner.DataRepresentation()
	if ns == nil {
		return nil
	}
	return nsDataToBytes(ns)
}

// ── MacOSRestoreImage accessors ───────────────────────────────────────────

// MostFeaturefulSupportedConfiguration returns the configuration requirements
// for the most capable hardware model this restore image supports on the current host.
// Returns nil if no supported configuration exists for the current Mac.
func (x *MacOSRestoreImage) MostFeaturefulSupportedConfiguration() *MacOSConfigurationRequirements {
	r := x.inner.MostFeaturefulSupportedConfiguration()
	if r == nil {
		return nil
	}
	return &MacOSConfigurationRequirements{inner: r}
}

// ── MacOSConfigurationRequirements accessors ──────────────────────────────

// HardwareModel returns the hardware model required by this configuration.
func (x *MacOSConfigurationRequirements) HardwareModel() *MacHardwareModel {
	r := x.inner.HardwareModel()
	if r == nil {
		return nil
	}
	return &MacHardwareModel{inner: r}
}

// MinimumSupportedCPUCount returns the minimum number of CPUs required.
func (x *MacOSConfigurationRequirements) MinimumSupportedCPUCount() uint {
	return x.inner.MinimumSupportedCPUCount()
}

// MinimumSupportedMemorySize returns the minimum memory in bytes required.
func (x *MacOSConfigurationRequirements) MinimumSupportedMemorySize() uint64 {
	return x.inner.MinimumSupportedMemorySize()
}

// ── MacOSInstaller accessors ──────────────────────────────────────────────

// Progress returns the installation progress as a fraction in [0.0, 1.0].
// Poll this from a separate goroutine while Install(ctx) is blocking.
func (x *MacOSInstaller) Progress() float64 {
	p := x.inner.Progress()
	if p == nil {
		return 0
	}
	return p.FractionCompleted()
}

// ── Convenience constructors ──────────────────────────────────────────────

// LoadMacOSRestoreImage loads a macOS restore image from a local IPSW file.
// Wraps the asynchronous +[VZMacOSRestoreImage loadFileURL:completionHandler:]
// class method and blocks until loading completes or ctx is cancelled.
func LoadMacOSRestoreImage(ctx context.Context, path string) (*MacOSRestoreImage, error) {
	type result struct {
		img *raw.VZMacOSRestoreImage
		err error
	}
	ch := make(chan result, 1)
	nsURL := foundation.NSURLFileURLWithPath(foundation.NSStringStringWithUTF8String(path))
	raw.VZMacOSRestoreImageLoadFileURLCompletionHandler(nsURL, func(img *raw.VZMacOSRestoreImage, errPtr unsafe.Pointer) {
		if errPtr != nil {
			ch <- result{err: purego.NSErrorToError(objc.ID(uintptr(errPtr)))}
			return
		}
		ch <- result{img: img}
	})
	select {
	case r := <-ch:
		if r.err != nil {
			return nil, r.err
		}
		return &MacOSRestoreImage{inner: r.img}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// NewMacHardwareModelFromBytes reconstructs a MacHardwareModel from its serialised
// representation (as returned by MacHardwareModel.DataRepresentation).
func NewMacHardwareModelFromBytes(data []byte) *MacHardwareModel {
	if len(data) == 0 {
		return nil
	}
	ns := foundation.NSDataDataWithBytesLength(unsafe.Pointer(&data[0]), uint(len(data)))
	return NewMacHardwareModelWithDataRepresentation(ns)
}

// NewMacMachineIdentifierFromBytes reconstructs a MacMachineIdentifier from its
// serialised representation (as returned by MacMachineIdentifier.DataRepresentation).
func NewMacMachineIdentifierFromBytes(data []byte) *MacMachineIdentifier {
	if len(data) == 0 {
		return nil
	}
	ns := foundation.NSDataDataWithBytesLength(unsafe.Pointer(&data[0]), uint(len(data)))
	return NewMacMachineIdentifierWithDataRepresentation(ns)
}

// NewMacAuxiliaryStorageCreating creates new auxiliary storage at path initialised
// with the given hardware model. Used when setting up a fresh macOS VM.
func NewMacAuxiliaryStorageCreating(path string, hwModel *MacHardwareModel) (*MacAuxiliaryStorage, error) {
	return NewMacAuxiliaryStorageCreatingStorageAtURLHardwareModelOptionsError(
		path, hwModel.Unwrap(), 0,
	)
}

// NewVirtualMachineFromConfig creates a VirtualMachine from a trial
// VirtualMachineConfiguration, using the main dispatch queue.
func NewVirtualMachineFromConfig(cfg *VirtualMachineConfiguration) *VirtualMachine {
	return NewVirtualMachineWithConfigurationQueue(cfg.Unwrap(), nil)
}

// NewMacOSInstallerFor creates a MacOSInstaller from trial types.
func NewMacOSInstallerFor(vm *VirtualMachine, restoreImagePath string) *MacOSInstaller {
	return NewMacOSInstallerWithVirtualMachineRestoreImageURL(vm.Unwrap(), restoreImagePath)
}

// ── Internal helpers ──────────────────────────────────────────────────────

// nsDataToBytes converts an NSData object to a Go byte slice by copying its bytes.
func nsDataToBytes(ns *foundation.NSData) []byte {
	n := ns.Length()
	if n == 0 {
		return nil
	}
	ptr := ns.Bytes()
	if ptr == nil {
		return nil
	}
	return append([]byte(nil), unsafe.Slice((*byte)(ptr), n)...)
}

