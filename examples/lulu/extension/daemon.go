//go:build darwin

package extension

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/lulu/rules"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/lulu/shared"
)

var daemonExportCounter atomic.Uint64

// StartDaemon vends the daemon XPC service over the mach service the app
// connects to, dispatching each request to the rule engine. Mirrors LuLu's
// XPCListener + XPCDaemon. Call Resume on the returned listener-backed loop via
// RunExtension's run loop.
func StartDaemon(eng *rules.Engine) error {
	listener := rt.NewMachServiceListener(shared.DaemonMachServiceName)
	err := listener.HandleNewConnections(func(c rt.XPCConn) bool {
		className := fmt.Sprintf("LuLuDaemonExport_%d", daemonExportCounter.Add(1))
		obj, err := rt.NewExportedObject(className, rt.XPCExport{
			Protocol: shared.DaemonProtocol(),
			Handlers: daemonHandlers(eng),
		})
		if err != nil {
			return false
		}
		if err := c.SetExported(shared.DaemonProtocol(), obj); err != nil {
			return false
		}
		c.Resume()
		return true
	})
	if err != nil {
		return err
	}
	listener.Resume()
	return nil
}

// daemonHandlers maps the daemon protocol selectors to Go closures over eng.
func daemonHandlers(eng *rules.Engine) map[string]any {
	return map[string]any{
		// getRulesWithReply: replies with the rule set as JSON-in-NSData.
		"getRulesWithReply:": func(_ rt.ID, _ rt.SEL, reply rt.Block) {
			data, err := eng.MarshalJSON()
			if err != nil {
				data = []byte("{}")
			}
			_, _ = rt.InvokeBlock[uintptr](reply, bytesToNSData(data))
		},
		// addRule: takes a rule as JSON-in-NSData.
		"addRule:": func(_ rt.ID, _ rt.SEL, ruleData rt.ID) {
			var r shared.Rule
			if json.Unmarshal(nsDataBytes(ruleData), &r) == nil {
				eng.Add(&r)
				_ = eng.Save()
			}
		},
		// deleteRuleForKey:rule: removes the rule (key, uuid).
		"deleteRuleForKey:rule:": func(_ rt.ID, _ rt.SEL, key, uuid rt.ID) {
			if eng.Delete(rt.GoString(key), rt.GoString(uuid)) {
				_ = eng.Save()
			}
		},
		// toggleRuleForKey:rule:state: enables/disables a rule (state: 1 = disabled).
		"toggleRuleForKey:rule:state:": func(_ rt.ID, _ rt.SEL, key, uuid, state rt.ID) {
			disabled := rt.Send[int64](state, rt.RegisterName("integerValue")) != 0
			if eng.Toggle(rt.GoString(key), rt.GoString(uuid), disabled) {
				_ = eng.Save()
			}
		},
		// uninstallWithReply: acknowledges (no-op teardown for the port).
		"uninstallWithReply:": func(_ rt.ID, _ rt.SEL, reply rt.Block) {
			_, _ = rt.InvokeBlock[uintptr](reply, boolNumber(true))
		},
	}
}

// boolNumber wraps a Go bool as an NSNumber for reply blocks.
func boolNumber(v bool) rt.ID {
	n := int64(0)
	if v {
		n = 1
	}
	return rt.Send[rt.ID](classID("NSNumber"), rt.RegisterName("numberWithBool:"), n)
}
