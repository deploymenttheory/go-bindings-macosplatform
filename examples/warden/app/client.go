//go:build darwin

package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	rt "github.com/deploymenttheory/go-bindings-macosplatform/bindings/runtime/purego"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/warden/shared"

	_ "github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
)

// Client is a connection to the Warden daemon's XPC service.
type Client struct {
	conn  rt.XPCConn
	proxy rt.XPCProxy
}

// Connect dials the daemon mach service and configures the remote interface,
// returning a ready Client the caller must Close.
func Connect() (Client, error) {
	conn := rt.DialMachService(shared.DaemonMachServiceName, false)
	if err := conn.SetRemoteInterface(shared.DaemonProtocol()); err != nil {
		conn.Invalidate()
		return Client{}, fmt.Errorf("warden: configure remote interface: %w", err)
	}
	conn.Resume()
	proxy := conn.RemoteProxy(func(error) {})
	return Client{conn: conn, proxy: proxy}, nil
}

// Close tears down the connection.
func (c Client) Close() { c.conn.Invalidate() }

// GetRules requests the daemon's rule set (JSON), blocking up to 5s for the reply.
func (c Client) GetRules() (map[string][]*shared.Rule, error) {
	done := make(chan []byte, 1)
	c.proxy.CallWithReply("getRulesWithReply:", func(_ rt.Block, data rt.ID) {
		done <- shared.NSDataBytes(data)
	})
	select {
	case raw := <-done:
		var out map[string][]*shared.Rule
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
		return out, nil
	case <-time.After(5 * time.Second):
		return nil, errors.New("warden: timed out waiting for daemon reply")
	}
}

// AddRule sends a new rule to the daemon. It returns an error only if r cannot be
// encoded; the XPC send itself is one-way (see DeleteRule for that contract).
func (c Client) AddRule(r *shared.Rule) error {
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("warden: encode rule: %w", err)
	}
	c.proxy.CallOneWay("addRule:", shared.BytesToNSData(data))
	return nil
}

// DeleteRule asks the daemon to remove the rule (key, uuid). The XPC call is
// one-way, so this returns nothing: success means "request delivered", not
// "rule removed" — confirm the effect with a subsequent GetRules if needed.
func (c Client) DeleteRule(key, uuid string) {
	c.proxy.CallOneWay("deleteRuleForKey:rule:", rt.NSString(key), rt.NSString(uuid))
}

// ToggleRule enables/disables the rule (key, uuid) on the daemon. Like DeleteRule
// this is a one-way send with no acknowledgement.
func (c Client) ToggleRule(key, uuid string, disabled bool) {
	n := int64(0)
	if disabled {
		n = 1
	}
	num := rt.Send[rt.ID](shared.ClassID("NSNumber"), rt.RegisterName("numberWithBool:"), n)
	c.proxy.CallOneWay("toggleRuleForKey:rule:state:", rt.NSString(key), rt.NSString(uuid), num)
}
