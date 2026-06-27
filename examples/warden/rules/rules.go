//go:build darwin

// Package rules implements Warden's rule engine: an in-memory, disk-backed store
// of firewall rules keyed by process identity, with the lookup that the network
// extension consults for every new flow. Mirrors Warden's Extension/Rules.
package rules

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/deploymenttheory/go-bindings-macosplatform/examples/warden/shared"
)

// Engine is a concurrency-safe rule store keyed by process identity (signing id
// or path). The network extension reads it on the flow path; the XPC daemon
// mutates it in response to app requests and user alert decisions.
type Engine struct {
	mu    sync.RWMutex
	path  string                    // JSON persistence path ("" = memory only)
	rules map[string][]*shared.Rule // process key → rules
}

// New returns an empty engine persisting to path (pass "" for memory-only).
func New(path string) *Engine {
	return &Engine{path: path, rules: map[string][]*shared.Rule{}}
}

// Load reads the rule set from disk. A missing file is not an error.
func (e *Engine) Load() error {
	if e.path == "" {
		return nil
	}
	data, err := os.ReadFile(e.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var stored map[string][]*shared.Rule
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}
	e.mu.Lock()
	e.rules = stored
	e.mu.Unlock()
	return nil
}

// Save writes the rule set to disk (no-op for a memory-only engine).
func (e *Engine) Save() error {
	if e.path == "" {
		return nil
	}
	e.mu.RLock()
	data, err := json.MarshalIndent(e.rules, "", "  ")
	e.mu.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(e.path, data, 0o600)
}

// Find returns the verdict for a flow from process key to remoteAddr:remotePort:
// RuleStateAllow, RuleStateBlock, or RuleStateNotFound when no rule matches
// (the caller then prompts the user). Endpoint-specific rules win over
// process-wide ones because they are matched first.
func (e *Engine) Find(key, remoteAddr, remotePort string) shared.RuleState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	candidates := e.rules[key]
	// Endpoint-specific rules first.
	for _, r := range candidates {
		if r.EndpointAddr != "" && r.EndpointAddr != "*" && r.Matches(remoteAddr, remotePort) {
			return r.Action
		}
	}
	// Process-wide rules.
	for _, r := range candidates {
		if (r.EndpointAddr == "" || r.EndpointAddr == "*") && r.Matches(remoteAddr, remotePort) {
			return r.Action
		}
	}
	return shared.RuleStateNotFound
}

// Add inserts a rule under its Key.
func (e *Engine) Add(r *shared.Rule) {
	e.mu.Lock()
	e.rules[r.Key] = append(e.rules[r.Key], r)
	e.mu.Unlock()
}

// Delete removes the rule with uuid under key, returning whether it was found.
func (e *Engine) Delete(key, uuid string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	list := e.rules[key]
	for i, r := range list {
		if r.UUID == uuid {
			e.rules[key] = append(list[:i], list[i+1:]...)
			if len(e.rules[key]) == 0 {
				delete(e.rules, key)
			}
			return true
		}
	}
	return false
}

// Toggle enables/disables the rule with uuid under key.
func (e *Engine) Toggle(key, uuid string, disabled bool) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, r := range e.rules[key] {
		if r.UUID == uuid {
			r.IsDisabled = disabled
			return true
		}
	}
	return false
}

// All returns a flat snapshot copy of every rule.
func (e *Engine) All() []*shared.Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var out []*shared.Rule
	for _, list := range e.rules {
		out = append(out, list...)
	}
	return out
}

// MarshalJSON serializes the whole store (used by the daemon's getRules reply).
func (e *Engine) MarshalJSON() ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return json.Marshal(e.rules)
}
