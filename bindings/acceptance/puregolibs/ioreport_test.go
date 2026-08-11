//go:build darwin

package puregolibs_test

import (
	"testing"

	ioreport "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/ioreport"
)

// TestIOReport_CopyAllChannels calls the private IOReport library live. Some
// virtualised runners expose no channels; nil is a skip there, matching the
// curated acceptance test.
func TestIOReport_CopyAllChannels(t *testing.T) {
	if ptr := ioreport.IOReportCopyAllChannels(0, 0); ptr == nil {
		t.Skip("IOReportCopyAllChannels returned nil (no channels on this host)")
	}
}
