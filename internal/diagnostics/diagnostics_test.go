package diagnostics

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestNormaliseDeduplicatesAndSorts(t *testing.T) {
	in := []string{"b", "a", "b", "c", "a"}
	got := Normalise(in)
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Normalise: got %v, want %v", got, want)
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	entries := []string{"AppKit: \"FooRef\" → unsafe.Pointer (unresolved named type)", "Foundation: bar"}
	if err := Write(path, entries); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !reflect.DeepEqual(got, Normalise(entries)) {
		t.Errorf("round-trip: got %v, want %v", got, Normalise(entries))
	}
}

func TestCheckBaselineDetectsNewAndFixed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	if err := Write(path, []string{"known-1", "known-2", "fixed-since"}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	newEntries, fixedEntries, err := CheckBaseline(path, []string{"known-1", "known-2", "brand-new"})
	if err != nil {
		t.Fatalf("CheckBaseline: %v", err)
	}
	if want := []string{"brand-new"}; !reflect.DeepEqual(newEntries, want) {
		t.Errorf("newEntries: got %v, want %v", newEntries, want)
	}
	if want := []string{"fixed-since"}; !reflect.DeepEqual(fixedEntries, want) {
		t.Errorf("fixedEntries: got %v, want %v", fixedEntries, want)
	}
}

func TestCheckBaselineCleanMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.json")
	if err := Write(path, []string{"a", "b"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	newEntries, fixedEntries, err := CheckBaseline(path, []string{"b", "a", "a"})
	if err != nil {
		t.Fatalf("CheckBaseline: %v", err)
	}
	if len(newEntries) != 0 || len(fixedEntries) != 0 {
		t.Errorf("expected clean match; got new=%v fixed=%v", newEntries, fixedEntries)
	}
}

func TestCheckBaselineMissingFile(t *testing.T) {
	_, _, err := CheckBaseline("/no/such/baseline.json", []string{"a"})
	if err == nil {
		t.Fatal("expected error for missing baseline file, got nil")
	}
}
