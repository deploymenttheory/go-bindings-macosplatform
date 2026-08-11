//go:build darwin

package puregolibs_test

import (
	"testing"

	oslog "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/oslog"
)

// TestOSLog_CreateAndQuery creates a real os_log_t via os_log_create (two
// string params marshalled into libsystem_trace), then queries its enabled
// state and per-type gating. os_log_create never returns nil — an unconfigured
// subsystem yields the shared OS_LOG_DEFAULT — so a non-nil handle plus a
// coherent FAULT-enabled answer proves the calls crossed correctly.
func TestOSLog_CreateAndQuery(t *testing.T) {
	log := oslog.Os_log_create("com.weave.puregolibs.test", "acceptance")
	if log == nil {
		t.Fatal("os_log_create returned nil")
	}

	// Faults are always enabled; if this call mis-marshalled the enum or handle
	// it would not report the documented always-on level.
	if !oslog.Os_log_type_enabled(log, oslog.OS_LOG_TYPE_FAULT) {
		t.Error("os_log_type_enabled(FAULT) = false; faults are always enabled")
	}

	// os_log_is_enabled must agree it is a usable log object (never panics /
	// crashes on the returned handle).
	_ = oslog.Os_log_is_enabled(log)
	_ = oslog.Os_log_is_debug_enabled(log)
}
