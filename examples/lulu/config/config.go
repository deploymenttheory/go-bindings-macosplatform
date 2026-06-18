//go:build darwin

// Package config defines a declarative firewall configuration document (JSON or
// YAML) and reconciles it against a rule store, kubectl-apply style: rules in
// the document are ensured present and managed rules absent from it are pruned.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/deploymenttheory/go-bindings-macosplatform/examples/lulu/shared"
	"gopkg.in/yaml.v3"
)

// Config is a declarative firewall configuration. It can be authored in JSON or
// YAML; the same struct tags drive both.
type Config struct {
	Version       string     `json:"version" yaml:"version"`
	DefaultAction string     `json:"defaultAction" yaml:"defaultAction"` // "allow" | "block"
	Rules         []RuleSpec `json:"rules" yaml:"rules"`
}

// RuleSpec is one declarative rule: a verdict for a process, optionally scoped to
// specific endpoints. With no endpoints the rule is process-wide.
type RuleSpec struct {
	Name      string         `json:"name,omitempty" yaml:"name,omitempty"`
	Path      string         `json:"path" yaml:"path"`
	Action    string         `json:"action" yaml:"action"` // "allow" | "block"
	Endpoints []EndpointSpec `json:"endpoints,omitempty" yaml:"endpoints,omitempty"`
}

// EndpointSpec restricts a rule to a remote host and/or port. Empty fields mean
// "any"; "*" is also accepted as a wildcard.
type EndpointSpec struct {
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
	Port string `json:"port,omitempty" yaml:"port,omitempty"`
}

// Load reads and validates a config document. The format is chosen by file
// extension (.json → JSON, .yaml/.yml → YAML); for any other extension the
// content is sniffed (a leading '{' is JSON, otherwise YAML).
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := Parse(data, formatFor(path, data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// Format identifies a serialization.
type Format int

const (
	FormatJSON Format = iota
	FormatYAML
)

func formatFor(path string, data []byte) Format {
	switch strings.ToLower(path[strings.LastIndex(path, ".")+1:]) {
	case "json":
		return FormatJSON
	case "yaml", "yml":
		return FormatYAML
	}
	if t := strings.TrimSpace(string(data)); strings.HasPrefix(t, "{") {
		return FormatJSON
	}
	return FormatYAML
}

// Parse decodes a config document in the given format and validates it.
func Parse(data []byte, format Format) (*Config, error) {
	var cfg Config
	switch format {
	case FormatJSON:
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	default:
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Marshal serializes the config in the given format.
func (c *Config) Marshal(format Format) ([]byte, error) {
	if format == FormatJSON {
		return json.MarshalIndent(c, "", "  ")
	}
	return yaml.Marshal(c)
}

// Validate checks the document for structural correctness.
func (c *Config) Validate() error {
	if c.DefaultAction != "" {
		if _, err := actionState(c.DefaultAction); err != nil {
			return fmt.Errorf("defaultAction: %w", err)
		}
	}
	for i, r := range c.Rules {
		if r.Path == "" {
			return fmt.Errorf("rules[%d]: path is required", i)
		}
		if _, err := actionState(r.Action); err != nil {
			return fmt.Errorf("rules[%d] (%s): %w", i, r.Path, err)
		}
	}
	return nil
}

// actionState maps "allow"/"block" to the corresponding rule state.
func actionState(s string) (int, error) {
	switch strings.ToLower(s) {
	case "allow":
		return shared.RuleStateAllow, nil
	case "block":
		return shared.RuleStateBlock, nil
	default:
		return 0, fmt.Errorf("invalid action %q (want \"allow\" or \"block\")", s)
	}
}

// Desired expands the config into the concrete set of managed rules it implies.
// Each rule's UUID is content-addressed (a hash of path+action+host+port) so
// re-applying an unchanged document is a no-op and an edited endpoint cleanly
// replaces the old managed rule.
func (c *Config) Desired() []*shared.Rule {
	now := time.Now()
	var out []*shared.Rule
	for _, spec := range c.Rules {
		action, _ := actionState(spec.Action) // validated already
		eps := spec.Endpoints
		if len(eps) == 0 {
			eps = []EndpointSpec{{}} // process-wide
		}
		for _, ep := range eps {
			out = append(out, &shared.Rule{
				UUID:         managedUUID(spec.Path, spec.Action, ep.Host, ep.Port),
				Key:          spec.Path,
				Path:         spec.Path,
				Name:         spec.Name,
				EndpointAddr: ep.Host,
				EndpointPort: ep.Port,
				Action:       action,
				Creation:     now,
				Managed:      true,
			})
		}
	}
	return out
}

func managedUUID(path, action, host, port string) string {
	h := sha256.Sum256([]byte(path + "\x00" + action + "\x00" + host + "\x00" + port))
	return hex.EncodeToString(h[:16])
}

// RuleStore is the subset of the rule engine the reconciler needs.
// rules.Engine satisfies it directly.
type RuleStore interface {
	All() []*shared.Rule
	Add(r *shared.Rule)
	Delete(key, uuid string) bool
}

// Diff computes the authoritative reconciliation between the desired rule set and
// the current rules: the document is the complete firewall state. It returns the
// rules to add (in desired, not current) and the rules to delete (current, not in
// desired — every one of them, regardless of origin). Pruning is unconditional:
// a rule added out-of-band (interactively or by any XPC client) that the document
// does not sanction is removed, so the policy cannot be circumvented by adding
// local rules.
func Diff(desired, current []*shared.Rule) (toAdd, toDelete []*shared.Rule) {
	desiredByUUID := make(map[string]*shared.Rule, len(desired))
	for _, r := range desired {
		desiredByUUID[r.UUID] = r
	}
	currentByUUID := make(map[string]*shared.Rule, len(current))
	for _, r := range current {
		currentByUUID[r.UUID] = r
	}
	for uuid, r := range desiredByUUID {
		if _, ok := currentByUUID[uuid]; !ok {
			toAdd = append(toAdd, r)
		}
	}
	for uuid, r := range currentByUUID {
		if _, ok := desiredByUUID[uuid]; !ok {
			toDelete = append(toDelete, r)
		}
	}
	return toAdd, toDelete
}

// Apply authoritatively reconciles store to match cfg, returning how many rules
// were added and deleted. The config is the single source of truth: any rule not
// in the document is removed. It is idempotent: applying the same config twice
// changes nothing the second time.
func Apply(cfg *Config, store RuleStore) (added, deleted int) {
	toAdd, toDelete := Diff(cfg.Desired(), store.All())
	for _, r := range toAdd {
		store.Add(r)
		added++
	}
	for _, r := range toDelete {
		if store.Delete(r.Key, r.UUID) {
			deleted++
		}
	}
	return added, deleted
}

// FromRules builds a Config document from an existing rule set (the inverse of
// Desired), so a running firewall's managed rules can be exported and re-edited.
func FromRules(rules []*shared.Rule) *Config {
	byPath := map[string]*RuleSpec{}
	var order []string
	for _, r := range rules {
		action := "allow"
		if r.Action == shared.RuleStateBlock {
			action = "block"
		}
		spec, ok := byPath[r.Path+"\x00"+action]
		if !ok {
			order = append(order, r.Path+"\x00"+action)
			spec = &RuleSpec{Name: r.Name, Path: r.Path, Action: action}
			byPath[r.Path+"\x00"+action] = spec
		}
		if r.EndpointAddr != "" || r.EndpointPort != "" {
			spec.Endpoints = append(spec.Endpoints, EndpointSpec{Host: r.EndpointAddr, Port: r.EndpointPort})
		}
	}
	cfg := &Config{Version: "1"}
	for _, k := range order {
		cfg.Rules = append(cfg.Rules, *byPath[k])
	}
	return cfg
}
