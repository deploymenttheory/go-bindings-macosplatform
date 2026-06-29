//go:build darwin

package extension

import (
	"unsafe"

	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/warden/rules"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/warden/shared"
	// The idiomatic NetworkExtension package supplies the typed verdict factories
	// and, by being imported, loads NetworkExtension.framework so its classes
	// (NEFilterDataProvider, NEFilterNewFlowVerdict, NEProvider, …) resolve for the
	// low-level flow inspection done via the runtime below.
	ne "github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/framework/networkextension"
	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/obj"
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

	// ADOPTION: this is the case the idiomatic layer cannot serve. The system
	// instantiates *our* class and calls its methods, so we must define a new ObjC
	// subclass of NEFilterDataProvider whose methods are backed by the Go closures
	// above — that's rt.NewDelegate, a pure-runtime operation. The idiomatic layer
	// wraps existing classes; it has no way to register a subclass. Inside the
	// handlers it's the reverse: the verdicts (allowVerdict/dropVerdict, below) are
	// built with the idiomatic NetworkExtension wrappers, and only the low-level flow
	// inspection (audit-token bytes, endpoint fields) stays on rt.Send. So one
	// function spans both layers: runtime for the subclass shell, idiomatic for the
	// framework objects it produces. See examples/README.md.
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

// allowVerdict / dropVerdict build the corresponding NEFilterNewFlowVerdict via
// the idiomatic factories, returning its raw object id for the -handleNewFlow:
// implementation to hand back to the system.
func allowVerdict() rt.ID {
	return obj.ID(ne.NEFilterNewFlowVerdictAllowVerdict())
}

func dropVerdict() rt.ID {
	return obj.ID(ne.NEFilterNewFlowVerdictDropVerdict())
}
