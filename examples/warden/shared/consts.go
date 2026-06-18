//go:build darwin

// Package shared holds the models, constants, and XPC protocol descriptors used
// by both the Warden network-extension daemon and the controlling app — mirroring
// Warden's Shared/ directory (Rule, consts, XPCDaemonProto, XPCUserProto).
package shared

// Rule states: the verdict a rule encodes for a flow.
const (
	RuleStateNotFound = -1 // no matching rule
	RuleStateBlock    = 0  // deny the flow
	RuleStateAllow    = 1  // permit the flow
)

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
