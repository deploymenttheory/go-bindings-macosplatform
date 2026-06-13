// Package diagnostics persists generator type-degradation diagnostics and
// enforces a committed baseline over them.
//
// The ratchet model: the baseline file records every currently-known
// degradation (unsafe.Pointer fallbacks, cycle-forced objc.ID substitutions).
// CI regenerates the bindings, collects fresh diagnostics, and fails when an
// entry appears that is not in the baseline — so quality can only improve.
// Fixing a degradation shrinks the baseline; introducing one requires an
// explicit, reviewed baseline edit.
package diagnostics

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// Normalise deduplicates and sorts diagnostic entries so that baseline files
// and comparisons are stable across runs regardless of emission order.
func Normalise(entries []string) []string {
	seen := make(map[string]bool, len(entries))
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !seen[entry] {
			seen[entry] = true
			out = append(out, entry)
		}
	}
	sort.Strings(out)
	return out
}

// Write serialises the normalised entries to path as an indented JSON array,
// one entry per line — the committed baseline format.
func Write(path string, entries []string) error {
	data, err := json.MarshalIndent(Normalise(entries), "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling diagnostics: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing diagnostics %s: %w", path, err)
	}
	return nil
}

// Read loads a baseline file written by Write.
func Read(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading diagnostics baseline %s: %w", path, err)
	}
	var entries []string
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing diagnostics baseline %s: %w", path, err)
	}
	return entries, nil
}

// CheckBaseline compares fresh diagnostics against the baseline at path.
// newEntries are diagnostics absent from the baseline (regressions — these
// should fail CI). fixedEntries are baseline entries that no longer occur
// (improvements — the baseline can be shrunk by rewriting it).
func CheckBaseline(path string, entries []string) (newEntries, fixedEntries []string, err error) {
	baseline, err := Read(path)
	if err != nil {
		return nil, nil, err
	}
	baselineSet := make(map[string]bool, len(baseline))
	for _, entry := range baseline {
		baselineSet[entry] = true
	}
	current := Normalise(entries)
	currentSet := make(map[string]bool, len(current))
	for _, entry := range current {
		currentSet[entry] = true
		if !baselineSet[entry] {
			newEntries = append(newEntries, entry)
		}
	}
	for _, entry := range Normalise(baseline) {
		if !currentSet[entry] {
			fixedEntries = append(fixedEntries, entry)
		}
	}
	return newEntries, fixedEntries, nil
}
