//go:build darwin

package shared

import "time"

// Rule is a firewall rule, mirroring Warden's Shared/Rule. It is JSON-serializable
// so the daemon can persist the rule set to disk and ship it to the app over XPC
// (Warden uses NSData / NSKeyedArchiver; a Go port uses JSON for the same purpose).
type Rule struct {
	UUID string `json:"uuid"`
	// Key is the process identity the rule is filed under — the signing identifier
	// when code-signed, otherwise the binary path.
	Key string `json:"key"`

	// Process identity.
	Path        string `json:"path"`
	Name        string `json:"name,omitempty"`
	IsGlobal    bool   `json:"isGlobal,omitempty"`
	IsDirectory bool   `json:"isDirectory,omitempty"`

	// Endpoint match. Empty EndpointAddr means "any endpoint".
	EndpointAddr string       `json:"endpointAddr,omitempty"`
	EndpointPort string       `json:"endpointPort,omitempty"`
	EndpointType EndpointType `json:"endpointType,omitempty"`

	// Verdict + lifecycle.
	Action     RuleState  `json:"action"` // RuleStateAllow / RuleStateBlock
	Type       int        `json:"type,omitempty"`
	Protocol   int        `json:"protocol,omitempty"`
	IsDisabled bool       `json:"isDisabled,omitempty"`
	Creation   time.Time  `json:"creation"`
	Expiration *time.Time `json:"expiration,omitempty"`

	// Managed marks rules that originate from the declarative config (provenance,
	// e.g. for display). Reconciliation is authoritative and prunes any rule not
	// in the config regardless of this flag, so it is not a safety gate.
	Managed bool `json:"managed,omitempty"`
}

// Matches reports whether the rule applies to a flow to remoteAddr:remotePort.
// An empty EndpointAddr matches any endpoint (a process-wide rule). Disabled or
// expired rules never match.
func (r *Rule) Matches(remoteAddr, remotePort string) bool {
	if r.IsDisabled {
		return false
	}
	if r.Expiration != nil && time.Now().After(*r.Expiration) {
		return false
	}
	if r.EndpointAddr != "" && r.EndpointAddr != "*" && r.EndpointAddr != remoteAddr {
		return false
	}
	if r.EndpointPort != "" && r.EndpointPort != "*" && r.EndpointPort != remotePort {
		return false
	}
	return true
}
