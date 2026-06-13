package ergonomic

// NameTracker tracks package-level function names claimed by any ergonomic
// emitter for the current framework. All emitters share one instance per
// framework so cross-emitter collisions are caught before code is emitted.
//
// Previously each emitter maintained its own per-file `seen` map. When two
// emitters produced a function with the same name (e.g. constructors and
// properties both emitting "ScrollDirection") the second file failed to
// compile. This type serialises all claims into a single registry.
type NameTracker struct {
	claimed map[string]string // funcName → first emitter that claimed it
}

// NewNameTracker returns a NameTracker ready for use.
func NewNameTracker() *NameTracker {
	return &NameTracker{claimed: make(map[string]string)}
}

// Claim reserves name for the named emitter. Returns true when the claim
// succeeds (name was unclaimed). Returns false when already claimed by any
// emitter, including the caller itself.
func (nt *NameTracker) Claim(name, emitter string) bool {
	if _, exists := nt.claimed[name]; exists {
		return false
	}
	nt.claimed[name] = emitter
	return true
}
