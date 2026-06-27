//go:build darwin

package extension

import (
	"unsafe"

	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/warden/rules"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/warden/shared"

	// Side-effect import: loads NetworkExtension.framework so its classes
	// (NEFilterDataProvider, NEFilterNewFlowVerdict, NEProvider, …) resolve.
	_ "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/networkextension"
)

// ProviderClassName is the ObjC class the network extension's Info.plist names as
// its NSExtensionPrincipalClass — the system instantiates it to filter flows.
const ProviderClassName = "WardenFilterDataProvider"

// defaultAllowPolicy is the verdict applied to a flow that matches no rule. Warden
// is policy-driven (it does not prompt), so the unmatched default is fixed here; a
// declarative config can still block such flows by setting defaultAction: block.
const defaultAllowPolicy = true

// RegisterProvider registers the NEFilterDataProvider subclass whose flow handler
// consults eng. defaultAllow is the verdict for a flow with no matching rule —
// where LuLu prompts the user, this policy-driven port applies a configured default.
// The system extension runtime later instantiates ProviderClassName and drives
// startFilter / handleNewFlow on it.
//
// The handlers are closures over eng/defaultAllow rather than package globals (the
// same shape daemon.go uses), so the provider carries no mutable package state.
func RegisterProvider(eng *rules.Engine, defaultAllow bool) error {
	// startFilter signals the system the filter is ready. No up-front filtering
	// rules are installed, so every flow reaches handleNewFlow.
	startFilter := func(_ rt.ID, _ rt.SEL, completion rt.Block) {
		_, _ = rt.InvokeBlock[uintptr](completion, rt.ID(0)) // nil NSError → success
	}
	// stopFilter acknowledges teardown.
	stopFilter := func(_ rt.ID, _ rt.SEL, _ int, completion rt.Block) {
		_, _ = rt.InvokeBlock[uintptr](completion)
	}
	// handleNewFlow is the core firewall decision, called for every new flow:
	// attribute the flow to a process, look up the rule verdict, and return an
	// allow/drop NEFilterNewFlowVerdict. Equivalent to Warden's -handleNewFlow:.
	handleNewFlow := func(_ rt.ID, _ rt.SEL, flow rt.ID) rt.ID {
		addr, port := remoteEndpoint(flow)
		key := ProcessPath(flowPID(flow))
		if key == "" {
			key = "unknown"
		}
		switch eng.Find(key, addr, port) {
		case shared.RuleStateAllow:
			return allowVerdict()
		case shared.RuleStateBlock:
			return dropVerdict()
		default:
			if defaultAllow {
				return allowVerdict()
			}
			return dropVerdict()
		}
	}

	_, err := rt.NewDelegate(
		ProviderClassName,
		rt.GetClass("NEFilterDataProvider"),
		nil,
		rt.DelegateHandler{Selector: "startFilterWithCompletionHandler:", Fn: startFilter},
		rt.DelegateHandler{Selector: "stopFilterWithReason:completionHandler:", Fn: stopFilter},
		rt.DelegateHandler{Selector: "handleNewFlow:", Fn: handleNewFlow},
	)
	return err
}

// remoteEndpoint reads the destination host and port from a socket flow's
// remoteEndpoint (an NWHostEndpoint).
func remoteEndpoint(flow rt.ID) (addr, port string) {
	ep := rt.Send[rt.ID](flow, rt.RegisterName("remoteEndpoint"))
	if ep == 0 {
		return "", ""
	}
	addr = rt.GoString(rt.Send[rt.ID](ep, rt.RegisterName("hostname")))
	port = rt.GoString(rt.Send[rt.ID](ep, rt.RegisterName("port")))
	return addr, port
}

// flowPID extracts the source process id from the flow's sourceAppAuditToken.
// An audit_token_t is eight uint32 words; the pid is word 5.
func flowPID(flow rt.ID) int32 {
	data := rt.Send[rt.ID](flow, rt.RegisterName("sourceAppAuditToken"))
	if data == 0 {
		return -1
	}
	if rt.Send[uint64](data, rt.RegisterName("length")) < 32 {
		return -1
	}
	p := rt.Send[unsafe.Pointer](data, rt.RegisterName("bytes"))
	if p == nil {
		return -1
	}
	tok := (*[8]uint32)(p)
	return int32(tok[5])
}

// allowVerdict / dropVerdict build the corresponding NEFilterNewFlowVerdict.
func allowVerdict() rt.ID {
	return rt.Send[rt.ID](shared.ClassID("NEFilterNewFlowVerdict"), rt.RegisterName("allowVerdict"))
}

func dropVerdict() rt.ID {
	return rt.Send[rt.ID](shared.ClassID("NEFilterNewFlowVerdict"), rt.RegisterName("dropVerdict"))
}
