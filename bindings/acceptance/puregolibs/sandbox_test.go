//go:build darwin

package puregolibs_test

import (
	"testing"

	sandbox "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/sandbox"
)

// TestSandbox_InvalidProfileFails calls the real sandbox_init with a garbage
// profile name: a non-zero error return proves the string parameter reached
// libsandbox intact (without actually sandboxing the test process).
func TestSandbox_InvalidProfileFails(t *testing.T) {
	if rc := sandbox.Sandbox_init("no-such-profile-8f2d1", 0, nil); rc == 0 {
		t.Error("sandbox_init with a nonexistent named profile returned success")
	}
}
