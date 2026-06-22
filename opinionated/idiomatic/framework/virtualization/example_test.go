//go:build darwin

package virtualization_test

import (
	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/virtualization"
)

// ExampleVirtualMachineConfiguration builds a Linux virtual-machine configuration
// with the fluent API. Each With* setter returns the receiver, so configuration
// reads as a single expression.
//
// The boot loader is passed to WithBootLoader, which accepts a BootLoaderProvider
// — the sealed interface that only real members of the BootLoader hierarchy
// satisfy. Passing a value that is not a boot loader is a compile error, not a
// runtime surprise.
//
// This example is compile-only (it has no Output line), so it doubles as a
// smoke test that the generated API is usable without dispatching into the
// Virtualization framework.
func ExampleVirtualMachineConfiguration() {
	bootLoader := virtualization.NewLinuxBootLoaderWithKernelURL("/var/vm/vmlinuz").
		WithCommandLine("console=hvc0 root=/dev/vda").
		WithInitialRamdiskURL("/var/vm/initrd")

	config := virtualization.NewVirtualMachineConfiguration().
		WithBootLoader(bootLoader).
		WithCPUCount(2).
		WithMemorySize(2 * 1024 * 1024 * 1024)

	_ = config
}

// ExampleBootLoaderProvider shows that every concrete boot loader satisfies the
// sealed BootLoaderProvider interface, so any of them can be handed to a setter
// typed as the provider.
func ExampleBootLoaderProvider() {
	loaders := []virtualization.BootLoaderProvider{
		virtualization.NewLinuxBootLoader(),
		virtualization.NewEFIBootLoader(),
		virtualization.NewMacOSBootLoader(),
	}

	config := virtualization.NewVirtualMachineConfiguration()
	for _, loader := range loaders {
		config.WithBootLoader(loader)
	}
}
