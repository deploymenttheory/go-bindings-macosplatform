package emitmanifest

import (
	"path/filepath"
	"testing"
)

func TestNilRecorderIsNoOp(t *testing.T) {
	var r *Recorder
	r.Record(Entry{Kind: KindEnum, MetaKey: "X:enum:Y"}) // must not panic
	if got := r.Entries(); got != nil {
		t.Fatalf("nil recorder Entries() = %v, want nil", got)
	}
}

func TestCompareReportsMissingAndRename(t *testing.T) {
	raw := []Entry{
		{Style: StyleRaw, Kind: KindEnum, Framework: "Foundation", MetaKey: "Foundation:enum:NSComparisonResult", GoSymbol: "NSComparisonResult"},
		{Style: StyleRaw, Kind: KindStruct, Framework: "Foundation", MetaKey: "Foundation:struct:_NSRange", GoSymbol: "NSRange"},
		{Style: StyleRaw, Kind: KindStruct, Framework: "Foundation", MetaKey: "Foundation:struct:NSZone", GoSymbol: "NSZone"},
		// derived entries never count toward parity
		{Style: StyleRaw, Kind: KindProtocol, Framework: "Foundation", MetaKey: "Foundation:protocol:NSFooProxy", GoSymbol: "NSFooProxy", Derived: true},
	}
	idio := []Entry{
		// renamed, still present
		{Style: StyleIdiomatic, Kind: KindEnum, Framework: "Foundation", MetaKey: "Foundation:enum:NSComparisonResult", GoSymbol: "ComparisonResult"},
		// same name, present
		{Style: StyleIdiomatic, Kind: KindStruct, Framework: "Foundation", MetaKey: "Foundation:struct:_NSRange", GoSymbol: "NSRange"},
		// NSZone absent → missing
	}

	rep := Compare(raw, idio)

	if len(rep.Missing) != 1 || rep.Missing[0].MetaKey != "Foundation:struct:NSZone" {
		t.Fatalf("Missing = %+v, want only NSZone", rep.Missing)
	}
	if len(rep.Renames) != 1 || rep.Renames[0].RawSymbol != "NSComparisonResult" || rep.Renames[0].IdioSymbol != "ComparisonResult" {
		t.Fatalf("Renames = %+v, want NSComparisonResult→ComparisonResult", rep.Renames)
	}
	if rep.RawTotal != 3 { // derived excluded
		t.Fatalf("RawTotal = %d, want 3", rep.RawTotal)
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	r := New()
	r.Record(Entry{Style: StyleRaw, Kind: KindEnum, Framework: "F", MetaKey: "F:enum:E", GoPkg: "f", GoSymbol: "E"})
	path := filepath.Join(t.TempDir(), "m.json")
	if err := r.Write(path); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].MetaKey != "F:enum:E" {
		t.Fatalf("round trip = %+v", got)
	}
}

func TestMetaKeyAndFrameworkOf(t *testing.T) {
	k := MetaKey("Foundation", KindStructField, "_NSRange", "location")
	if k != "Foundation:struct-field:_NSRange.location" {
		t.Fatalf("MetaKey = %q", k)
	}
	if FrameworkOf(k) != "Foundation" {
		t.Fatalf("FrameworkOf = %q", FrameworkOf(k))
	}
}
