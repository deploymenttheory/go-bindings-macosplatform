// Package mainactor defines the shared schema for the per-framework
// "main actor isolation" sidecar (mainactor.json) committed next to each
// .gometa.json metadata file.
//
// The data answers one question for the code generator: which Objective-C
// classes, protocols, and selectors must be called on the main thread because
// Apple isolates them to Swift's @MainActor (the global main-thread actor).
//
// This fact is observable from Apple's Swift symbol graph but NOT from the
// Clang JSON AST the scanner uses (the JSON dump drops swift_attr argument
// strings). The harvest tool (scripts/tools/mainactorisolation) extracts it
// with swift-symbolgraph-extract and writes these sidecars; the codegen loaders
// read them at load time and propagate the isolation down the class hierarchy.
// This mirrors the appledocs sidecar convention exactly.
package mainactor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FileName is the sidecar's fixed name, written next to the .gometa.json.
const FileName = "mainactor.json"

// CurrentSchemaVersion is bumped when the Isolation layout changes incompatibly.
const CurrentSchemaVersion = 1

// Isolation is the on-disk sidecar payload for one framework.
//
// Selector strings use Objective-C method notation so they match a scanned
// Method directly: an instance selector is written bare ("addSubview:") and a
// class (factory) selector is written with a leading "+" ("+redColor"). This is
// the same instance/class disambiguation the scanner records as
// Method.IsClassMethod.
type Isolation struct {
	SchemaVersion int    `json:"schema_version"`
	Framework     string `json:"framework"`

	// MainActorClasses are classes whose instances are @MainActor-isolated:
	// every instance method and property accessor must run on the main thread.
	MainActorClasses []string `json:"main_actor_classes,omitempty"`

	// MainActorProtocols are protocols declared @MainActor. A class is treated
	// as main-thread when it conforms to one of these (callbacks the consumer
	// implements are expected on the main thread).
	MainActorProtocols []string `json:"main_actor_protocols,omitempty"`

	// MainActorSelectors lists individually-isolated selectors on classes that
	// are NOT wholly @MainActor (e.g. a single method marked NS_SWIFT_UI_ACTOR
	// on an otherwise thread-agnostic class). Keyed by class name.
	MainActorSelectors map[string][]string `json:"main_actor_selectors,omitempty"`

	// NonisolatedSelectors lists selectors on a @MainActor class that are
	// explicitly `nonisolated` — they opt OUT of main-thread affinity and must
	// not be wrapped. Keyed by class name.
	NonisolatedSelectors map[string][]string `json:"nonisolated_selectors,omitempty"`
}

// LoadAdjacent reads the mainactor.json sidecar sitting next to metaPath.
// A missing sidecar is not an error: it returns (nil, false, nil).
func LoadAdjacent(metaPath string) (*Isolation, bool, error) {
	path := filepath.Join(filepath.Dir(metaPath), FileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("reading %s: %w", path, err)
	}
	var iso Isolation
	if err := json.Unmarshal(data, &iso); err != nil {
		return nil, false, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &iso, true, nil
}
