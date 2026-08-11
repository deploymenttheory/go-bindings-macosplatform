//go:build darwin

package puregolibs_test

import (
	"testing"

	applearchive "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/applearchive"
	bsm "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/bsm"
	compression "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/compression"
	dispatch "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/dispatch"
	endpointsecurity "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/endpointsecurity"
	ioreport "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/ioreport"
	libproc "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/libproc"
	machhost "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machhost"
	machinit "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machinit"
	machtime "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machtime"
	machvm "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machvm"
	oslog "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/oslog"
	sandbox "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/sandbox"
	xar "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/xar"
	xpc "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/xpc"
)

// TestSymbolsResolve proves each purego-backed library dlopened its dylib and
// bound its key symbols — the direct per-library binding check.
func TestSymbolsResolve(t *testing.T) {
	cases := []struct {
		lib       string
		available func(string) bool
		symbol    string
	}{
		{"machtime", machtime.SymbolAvailable, "mach_absolute_time"},
		{"machinit", machinit.SymbolAvailable, "mach_host_self"},
		{"machvm", machvm.SymbolAvailable, "vm_allocate"},
		{"machhost", machhost.SymbolAvailable, "host_statistics64"},
		{"compression", compression.SymbolAvailable, "compression_encode_buffer"},
		{"sandbox", sandbox.SymbolAvailable, "sandbox_init"},
		{"xar", xar.SymbolAvailable, "xar_open"},
		{"ioreport", ioreport.SymbolAvailable, "IOReportCopyAllChannels"},
		{"bsm", bsm.SymbolAvailable, "audit_token_to_pid"},
		{"libproc", libproc.SymbolAvailable, "proc_pidpath"},
		{"endpointsecurity", endpointsecurity.SymbolAvailable, "es_new_client"},
		{"xpc", xpc.SymbolAvailable, "xpc_int64_create"},
		{"dispatch", dispatch.SymbolAvailable, "dispatch_async"},
		{"oslog", oslog.SymbolAvailable, "os_log_create"},
		{"applearchive", applearchive.SymbolAvailable, "AEAContextDestroy"},
	}
	for _, c := range cases {
		if !c.available(c.symbol) {
			t.Errorf("%s: symbol %q did not bind", c.lib, c.symbol)
		}
	}
}
