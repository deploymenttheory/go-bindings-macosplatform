//go:build darwin

package app

import (
	"encoding/json"
	"errors"
	"time"

	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/lulu/shared"

	_ "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
)

// Client is a connection to the LuLu daemon's XPC service.
type Client struct {
	conn  rt.XPCConn
	proxy rt.XPCProxy
}

// Connect dials the daemon mach service and configures the remote interface.
func Connect() Client {
	conn := rt.DialMachService(shared.DaemonMachServiceName, false)
	_ = conn.SetRemoteInterface(shared.DaemonProtocol())
	conn.Resume()
	proxy := conn.RemoteProxy(func(error) {})
	return Client{conn: conn, proxy: proxy}
}

// Close tears down the connection.
func (c Client) Close() { c.conn.Invalidate() }

// GetRules requests the daemon's rule set (JSON), blocking up to 5s for the reply.
func (c Client) GetRules() (map[string][]*shared.Rule, error) {
	done := make(chan []byte, 1)
	c.proxy.CallWithReply("getRulesWithReply:", func(_ rt.Block, data rt.ID) {
		done <- nsDataBytes(data)
	})
	select {
	case raw := <-done:
		var out map[string][]*shared.Rule
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
		return out, nil
	case <-time.After(5 * time.Second):
		return nil, errors.New("lulu: timed out waiting for daemon reply")
	}
}

// AddRule sends a new rule to the daemon (fire-and-forget).
func (c Client) AddRule(r *shared.Rule) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	c.proxy.CallOneWay("addRule:", bytesToNSData(data))
	return nil
}

// DeleteRule removes the rule (key, uuid) on the daemon.
func (c Client) DeleteRule(key, uuid string) {
	c.proxy.CallOneWay("deleteRuleForKey:rule:", rt.NSString(key), rt.NSString(uuid))
}

// ToggleRule enables/disables the rule (key, uuid) on the daemon.
func (c Client) ToggleRule(key, uuid string, disabled bool) {
	n := int64(0)
	if disabled {
		n = 1
	}
	num := rt.Send[rt.ID](classID("NSNumber"), rt.RegisterName("numberWithBool:"), n)
	c.proxy.CallOneWay("toggleRuleForKey:rule:state:", rt.NSString(key), rt.NSString(uuid), num)
}
