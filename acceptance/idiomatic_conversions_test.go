// Curated acceptance tests for the idiomatic layer's boundary conversions —
// hand-maintained, never regenerated. Each conversion the idiomatic emitter
// wires at signature boundaries (NSURL↔string, NSDate↔time.Time,
// NSData↔[]byte, NSDictionary↔map, NSSet↔slice) is round-tripped through real
// Foundation objects, plus one generated end-to-end call per direction so a
// regression in either the rt helpers or the emitted call sites fails here.
//
//go:build darwin

package acceptance_test

import (
	"bytes"
	"sort"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"

	idiofoundation "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/foundation"
	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/rt"

	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
)

func TestCurated_Idiomatic_URLStringRoundTrip(t *testing.T) {
	runIsolated(t, "curated:idiomatic URL <-> string round trip", func(t *testing.T) {
		const path = "/private/tmp/idiomatic-url-roundtrip"
		if got := rt.URLString(rt.FileURL(path)); got != path {
			t.Errorf("file URL round trip = %q; want %q", got, path)
		}

		// A non-file URL surfaces as its absolute string.
		const remote = "https://example.com/a/b?q=1"
		urlID := objc.ID(objc.GetClass("NSURL")).Send(
			objc.RegisterName("URLWithString:"), purego.NSString(remote))
		if got := rt.URLString(urlID); got != remote {
			t.Errorf("remote URL string = %q; want %q", got, remote)
		}

		if got := rt.URLString(0); got != "" {
			t.Errorf("nil URL string = %q; want \"\"", got)
		}
	})
}

func TestCurated_Idiomatic_DateTimeRoundTrip(t *testing.T) {
	runIsolated(t, "curated:idiomatic NSDate <-> time.Time round trip", func(t *testing.T) {
		want := time.Date(2026, 7, 9, 12, 34, 56, 789_000_000, time.UTC)
		got := rt.NSDateToTime(rt.TimeToNSDate(want))
		// NSDate stores a float64 of seconds, so allow sub-microsecond error.
		if diff := got.Sub(want); diff < -time.Microsecond || diff > time.Microsecond {
			t.Errorf("date round trip drifted %v (got %v, want %v)", diff, got, want)
		}

		if rt.TimeToNSDate(time.Time{}) != 0 {
			t.Error("zero time.Time must convert to a nil NSDate")
		}
		if !rt.NSDateToTime(0).IsZero() {
			t.Error("nil NSDate must convert to the zero time.Time")
		}

		// End-to-end through a generated signature: a *Date wrapper compares
		// equal to the same instant passed as time.Time.
		date := idiofoundation.NewDateWithTimeIntervalSince1970(1_600_000_000)
		if !date.IsEqualToDate(time.Unix(1_600_000_000, 0)) {
			t.Error("generated IsEqualToDate(time.Time) reported false for the same instant")
		}
	})
}

func TestCurated_Idiomatic_DataBytesRoundTrip(t *testing.T) {
	runIsolated(t, "curated:idiomatic NSData <-> []byte round trip", func(t *testing.T) {
		want := []byte{0x00, 0x01, 0xFE, 0xFF, 'g', 'o'}
		if got := rt.NSDataToBytes(rt.BytesToNSData(want)); !bytes.Equal(got, want) {
			t.Errorf("data round trip = %v; want %v", got, want)
		}
		if got := rt.NSDataToBytes(rt.BytesToNSData(nil)); got != nil {
			t.Errorf("empty data round trip = %v; want nil", got)
		}
		if got := rt.NSDataToBytes(0); got != nil {
			t.Errorf("nil NSData bytes = %v; want nil", got)
		}
	})
}

func TestCurated_Idiomatic_DictMapRoundTrip(t *testing.T) {
	runIsolated(t, "curated:idiomatic NSDictionary <-> map round trip", func(t *testing.T) {
		want := map[string]string{"alpha": "a", "beta": "b", "gamma": ""}
		dict := rt.MapToDict(want,
			func(k string) objc.ID { return purego.NSString(k) },
			func(v string) objc.ID { return purego.NSString(v) })
		got := rt.DictToMap(dict,
			func(id objc.ID) string { return purego.GoString(id) },
			func(id objc.ID) string { return purego.GoString(id) })
		if len(got) != len(want) {
			t.Fatalf("dict round trip has %d entries; want %d (%v)", len(got), len(want), got)
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("dict[%q] = %q; want %q", k, got[k], v)
			}
		}
	})
}

func TestCurated_Idiomatic_SetSliceRoundTrip(t *testing.T) {
	runIsolated(t, "curated:idiomatic NSSet <-> slice round trip", func(t *testing.T) {
		want := []string{"one", "two", "three"}
		set := rt.SliceToNSSet(want, func(s string) objc.ID { return purego.NSString(s) })
		got := rt.NSSetToSlice(set, func(id objc.ID) string { return purego.GoString(id) })
		sort.Strings(got)
		wantSorted := append([]string(nil), want...)
		sort.Strings(wantSorted)
		if len(got) != len(wantSorted) {
			t.Fatalf("set round trip has %d elements; want %d (%v)", len(got), len(wantSorted), got)
		}
		for i := range wantSorted {
			if got[i] != wantSorted[i] {
				t.Errorf("set element %d = %q; want %q", i, got[i], wantSorted[i])
			}
		}
	})
}
