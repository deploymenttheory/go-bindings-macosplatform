//go:build darwin

package shared

import rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"

// DaemonProtocol describes the daemon-side XPC interface (Warden's XPCDaemonProto):
// the app sends these to the extension's remote-object proxy. Reply-bearing
// methods carry their result in a reply block; fire-and-forget mutators do not.
func DaemonProtocol() rt.XPCProtocol {
	o, b := rt.XPCObject, rt.XPCReplyBlock
	return rt.XPCProtocol{
		Name: DaemonProtocolName,
		Methods: []rt.XPCMethod{
			{Selector: "getRulesWithReply:", Args: []rt.XPCArgKind{b}, ReplyArgs: []rt.XPCArgKind{o}},
			{Selector: "getPreferencesWithReply:", Args: []rt.XPCArgKind{b}, ReplyArgs: []rt.XPCArgKind{o}},
			{Selector: "updatePreferences:reply:", Args: []rt.XPCArgKind{o, b}, ReplyArgs: []rt.XPCArgKind{o}},
			{Selector: "addRule:", Args: []rt.XPCArgKind{o}},
			{Selector: "deleteRuleForKey:rule:", Args: []rt.XPCArgKind{o, o}},
			{Selector: "toggleRuleForKey:rule:state:", Args: []rt.XPCArgKind{o, o, o}},
			{Selector: "importRules:reply:", Args: []rt.XPCArgKind{o, b}, ReplyArgs: []rt.XPCArgKind{o}},
			{Selector: "uninstallWithReply:", Args: []rt.XPCArgKind{b}, ReplyArgs: []rt.XPCArgKind{o}},
		},
	}
}

// UserProtocol describes the app-side XPC interface (Warden's XPCUserProto): the
// daemon sends these to the app's exported object — a rules-changed notification
// and an alert that expects the user's decision in its reply.
func UserProtocol() rt.XPCProtocol {
	o, b := rt.XPCObject, rt.XPCReplyBlock
	return rt.XPCProtocol{
		Name: UserProtocolName,
		Methods: []rt.XPCMethod{
			{Selector: "rulesChanged"},
			{Selector: "showAlert:reply:", Args: []rt.XPCArgKind{o, b}, ReplyArgs: []rt.XPCArgKind{o}},
		},
	}
}
