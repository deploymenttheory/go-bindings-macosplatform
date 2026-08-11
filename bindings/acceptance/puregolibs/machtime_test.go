//go:build darwin

package puregolibs_test

import (
	"testing"

	machtime "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/machtime"
)

// TestMachtime_Monotonic makes real mach_absolute_time calls and checks the
// clock actually advances monotonically.
func TestMachtime_Monotonic(t *testing.T) {
	t1 := machtime.Mach_absolute_time()
	t2 := machtime.Mach_absolute_time()
	if t1 == 0 || t2 < t1 {
		t.Errorf("mach_absolute_time not monotonic: %d then %d", t1, t2)
	}
}

// TestMachtime_TimebaseInfo checks the out-parameter struct fill: the kernel
// writes numer/denom through the *MachTimebaseInfo pointer.
func TestMachtime_TimebaseInfo(t *testing.T) {
	var tb machtime.MachTimebaseInfo
	if rc := machtime.Mach_timebase_info(&tb); rc != 0 {
		t.Fatalf("mach_timebase_info rc = %d", rc)
	}
	if tb.Numer == 0 || tb.Denom == 0 {
		t.Errorf("timebase = %d/%d; want both > 0", tb.Numer, tb.Denom)
	}
}
