//go:build darwin

// Package shared holds the models, constants, and XPC protocol descriptors used
// by both the Warden network-extension daemon and the controlling app — mirroring
// Warden's Shared/ directory (Rule, consts, XPCDaemonProto, XPCUserProto).
package shared

// RuleState is the verdict a rule encodes for a flow. It is a named type rather
// than a bare int so the zero value (RuleStateBlock) is explicit at every call
// site and the compiler rejects accidental mixing with unrelated integers.
type RuleState int

const (
	RuleStateNotFound RuleState = -1 // no matching rule
	RuleStateBlock    RuleState = 0  // deny the flow
	RuleStateAllow    RuleState = 1  // permit the flow
)

// String renders a verdict as "allow"/"block" (and "not-found" for the sentinel),
// so callers can log a rule's action without re-deriving the mapping.
func (s RuleState) String() string {
	switch s {
	case RuleStateAllow:
		return "allow"
	case RuleStateBlock:
		return "block"
	default:
		return "not-found"
	}
}

// EndpointType classifies how a rule's endpoint address is matched.
type EndpointType int

const (
	EndpointTypeExact EndpointType = 0
	EndpointTypeRegex EndpointType = 1
	EndpointTypeCIDR  EndpointType = 2
)

// Rule durations (how long a rule persists).
const (
	RuleDurationAlways  = 101
	RuleDurationOnce    = 102
	RuleDurationProcess = 103
	RuleDurationCustom  = 104
)

// DaemonMachServiceName is the registered mach service the daemon vends and the
// app connects to. A real deployment uses a team-prefixed name matching the
// extension's NEMachServiceName entitlement.
const DaemonMachServiceName = "com.example.warden.daemon"

// Protocol names for the runtime-built ObjC protocols used over NSXPCConnection.
const (
	DaemonProtocolName = "WardenDaemonProtocol"
	UserProtocolName   = "WardenUserProtocol"
)
