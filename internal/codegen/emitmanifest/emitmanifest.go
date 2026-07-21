// Package emitmanifest records, per generated construct, which metadata symbol
// a code emitter turned into Go source and under what Go name.
//
// It exists to prove emittance parity between the two generator styles. The raw
// emitter (purego frameworks, CGo libraries) and the idiomatic emitter cover the
// same macOS SDK surface, but under different Go spellings — raw NSRange versus
// idiomatic Range, raw kSecClass versus idiomatic SecClass, raw's <pkg>_externs.go
// versus idiomatic's <pkg>_constants_generated.go. A plain diff of the emitted Go
// trees cannot tell a rename apart from a dropped symbol.
//
// The manifest sidesteps that by keying every entry on the ObjC/C metadata name
// (MetaKey), never the Go name. Two runs — one per style — each produce a manifest;
// the parity check (see cmd/parity) joins them on (Kind, MetaKey) and reports every
// raw entry that has no idiomatic counterpart. A differing GoSymbol on a matched key
// is a rename, reported for information only, not a parity failure.
//
// A Recorder is threaded through the generation config exactly like the existing
// type-degradation DiagnosticsSink: nil-safe, so a nil Recorder records nothing and
// generation is unaffected. Record one entry at the point each emitter has already
// decided to emit a construct.
package emitmanifest

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

// Style identifies which of the two generator styles produced an entry.
const (
	StyleRaw       = "raw"
	StyleIdiomatic = "idiomatic"
)

// Kind classifies a recorded construct. The parity check groups findings by
// Kind so that, for example, a missing enum member reads differently from a
// missing whole enum. These strings are the stable vocabulary shared by every
// record site across both emitters — add to them rather than inventing ad-hoc
// values, so the two styles always agree on how to name the same construct.
const (
	KindPackage        = "package"
	KindEnum           = "enum"
	KindEnumMember     = "enum-member"
	KindStruct         = "struct"
	KindStructField    = "struct-field"
	KindTypedefAlias   = "typedef-alias"
	KindExtern         = "extern"
	KindFunction       = "function"
	KindProtocol       = "protocol"
	KindProtocolMethod = "protocol-method"
	KindClass          = "class"
	KindClassMethod    = "class-method"
)

// Entry is one recorded emission: the metadata symbol a construct came from
// (MetaKey), plus the Go name it was emitted as (GoSymbol in package GoPkg).
type Entry struct {
	Style     string `json:"style"`
	Kind      string `json:"kind"`
	Framework string `json:"framework"`
	// MetaKey is the join key: the fully-qualified ObjC/C name of the construct,
	// canonically "<Framework>:<Kind>:<name>" (a struct field appends
	// ".<field>", a method appends ".<selector>"). It must be identical across
	// the two styles for the same construct — it is derived from metadata, never
	// from the emitted Go name.
	MetaKey string `json:"meta_key"`
	// GoPkg is the emitted Go package name (e.g. "foundation").
	GoPkg string `json:"go_pkg"`
	// GoSymbol is the Go identifier actually emitted (e.g. "NSRange" or "Range").
	GoSymbol string `json:"go_symbol"`
	// Derived marks helper constructs that are mechanically derived rather than
	// direct SDK surface (proxy types, duck-typed interfaces). They are recorded
	// for visibility but excluded from the parity denominator.
	Derived bool `json:"derived,omitempty"`
}

// joinKey is the identity the parity check matches on across the two styles.
// Framework is folded in via MetaKey's canonical prefix, so Kind+MetaKey is
// sufficient and avoids a spurious mismatch if the same name recurs under two
// kinds.
func (e Entry) joinKey() string { return e.Kind + "\x00" + e.MetaKey }

// Recorder accumulates entries during a generation run. The zero value is not
// usable; obtain one from New. A nil *Recorder is valid and records nothing, so
// call sites need no guard.
type Recorder struct {
	mu      sync.Mutex
	entries []Entry
}

// New returns an empty Recorder ready to accumulate entries.
func New() *Recorder { return &Recorder{} }

// Record appends one entry. It is safe on a nil receiver (no-op) and safe for
// concurrent callers, though generation is currently single-threaded.
func (r *Recorder) Record(e Entry) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.entries = append(r.entries, e)
	r.mu.Unlock()
}

// Entries returns a copy of the recorded entries, sorted for determinism.
func (r *Recorder) Entries() []Entry {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	out := make([]Entry, len(r.entries))
	copy(out, r.entries)
	r.mu.Unlock()
	sortEntries(out)
	return out
}

func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Style != b.Style {
			return a.Style < b.Style
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.MetaKey != b.MetaKey {
			return a.MetaKey < b.MetaKey
		}
		return a.GoSymbol < b.GoSymbol
	})
}

// Write serialises the recorder's entries to path as indented JSON.
func (r *Recorder) Write(path string) error {
	data, err := json.MarshalIndent(r.Entries(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling manifest: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing manifest %s: %w", path, err)
	}
	return nil
}

// Read loads a manifest written by Write.
func Read(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
	}
	return entries, nil
}

// Missing is a raw construct that has no idiomatic counterpart.
type Missing struct {
	Kind      string
	Framework string
	MetaKey   string
	GoSymbol  string // the raw Go name, to orient a reader
}

// Rename is a construct present in both styles but emitted under different Go
// names. Reported for information only — never a parity failure.
type Rename struct {
	Kind       string
	Framework  string
	MetaKey    string
	RawSymbol  string
	IdioSymbol string
}

// Report is the outcome of comparing a raw manifest against an idiomatic one.
type Report struct {
	Missing []Missing
	Renames []Rename
	// RawTotal / IdiomaticTotal count non-derived entries considered.
	RawTotal       int
	IdiomaticTotal int
}

// MissingByGroup buckets Missing entries by "<Kind>/<Framework>" for reporting.
func (rep Report) MissingByGroup() map[string][]Missing {
	out := make(map[string][]Missing)
	for _, m := range rep.Missing {
		key := m.Kind + "/" + m.Framework
		out[key] = append(out[key], m)
	}
	return out
}

// Compare joins raw and idiomatic entries on (Kind, MetaKey). Derived entries on
// either side are ignored. A raw key with no idiomatic entry is Missing; a key
// present on both with differing GoSymbol is a Rename.
func Compare(raw, idiomatic []Entry) Report {
	idioByKey := make(map[string]Entry, len(idiomatic))
	idioCount := 0
	for _, e := range idiomatic {
		if e.Derived {
			continue
		}
		idioCount++
		// First writer wins; a later duplicate key (same construct emitted
		// twice) does not overwrite the recorded Go name.
		if _, ok := idioByKey[e.joinKey()]; !ok {
			idioByKey[e.joinKey()] = e
		}
	}

	rep := Report{IdiomaticTotal: idioCount}
	seenRaw := make(map[string]bool)
	for _, e := range raw {
		if e.Derived {
			continue
		}
		if seenRaw[e.joinKey()] {
			continue
		}
		seenRaw[e.joinKey()] = true
		rep.RawTotal++

		idio, ok := idioByKey[e.joinKey()]
		if !ok {
			rep.Missing = append(rep.Missing, Missing{
				Kind:      e.Kind,
				Framework: e.Framework,
				MetaKey:   e.MetaKey,
				GoSymbol:  e.GoSymbol,
			})
			continue
		}
		if idio.GoSymbol != e.GoSymbol {
			rep.Renames = append(rep.Renames, Rename{
				Kind:       e.Kind,
				Framework:  e.Framework,
				MetaKey:    e.MetaKey,
				RawSymbol:  e.GoSymbol,
				IdioSymbol: idio.GoSymbol,
			})
		}
	}

	sort.Slice(rep.Missing, func(i, j int) bool {
		a, b := rep.Missing[i], rep.Missing[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Framework != b.Framework {
			return a.Framework < b.Framework
		}
		return a.MetaKey < b.MetaKey
	})
	sort.Slice(rep.Renames, func(i, j int) bool {
		a, b := rep.Renames[i], rep.Renames[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.MetaKey < b.MetaKey
	})
	return rep
}

// MetaKey builds the canonical join key "<framework>:<kind>:<name>". Suffix is
// appended (as ".<suffix>") for members and fields; pass "" when there is none.
func MetaKey(framework, kind, name, suffix string) string {
	key := framework + ":" + kind + ":" + name
	if suffix != "" {
		key += "." + suffix
	}
	return key
}

// FrameworkOf extracts the framework prefix from a MetaKey, or "" if malformed.
func FrameworkOf(metaKey string) string {
	if i := strings.IndexByte(metaKey, ':'); i >= 0 {
		return metaKey[:i]
	}
	return ""
}
