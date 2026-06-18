//go:build darwin

// Command lulu is the LuLu controlling app (CLI form): it activates/deactivates
// the network system extension and manages firewall rules over XPC. Mirrors the
// control surface of LuLu's App/ target.
//
// Usage:
//
//	lulu activate                 # submit the system-extension activation request
//	lulu deactivate               # submit the deactivation request
//	lulu list                     # list rules from the daemon
//	lulu allow <path> [host]      # add an allow rule for a process (+ optional endpoint)
//	lulu block <path> [host]      # add a block rule
//	lulu delete <key> <uuid>      # delete a rule
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/deploymenttheory/go-bindings-macosplatform/examples/lulu/app"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/lulu/shared"
)

// extensionBundleID is the network extension's bundle identifier. A real build
// uses the team-prefixed identifier embedded in the signed app.
const extensionBundleID = "com.example.lulu.extension"

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "activate":
		app.ActivateExtension(extensionBundleID)
		fmt.Println("submitted system-extension activation request for", extensionBundleID)
	case "deactivate":
		app.DeactivateExtension(extensionBundleID)
		fmt.Println("submitted system-extension deactivation request")
	case "list":
		listRules()
	case "allow":
		addRule(shared.RuleStateAllow)
	case "block":
		addRule(shared.RuleStateBlock)
	case "delete":
		if len(os.Args) < 4 {
			usage()
		}
		c := app.Connect()
		defer c.Close()
		c.DeleteRule(os.Args[2], os.Args[3])
		fmt.Println("delete requested")
	default:
		usage()
	}
}

func listRules() {
	c := app.Connect()
	defer c.Close()
	rules, err := c.GetRules()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if len(rules) == 0 {
		fmt.Println("(no rules)")
		return
	}
	for key, list := range rules {
		fmt.Printf("● %s\n", key)
		for _, r := range list {
			verdict := "allow"
			if r.Action == shared.RuleStateBlock {
				verdict = "block"
			}
			ep := r.EndpointAddr
			if ep == "" {
				ep = "*"
			}
			fmt.Printf("    %s  %s → %s  (%s)\n", verdict, ep, r.EndpointPort, r.UUID)
		}
	}
}

func addRule(action int) {
	if len(os.Args) < 3 {
		usage()
	}
	path := os.Args[2]
	host := ""
	if len(os.Args) >= 4 {
		host = os.Args[3]
	}
	r := &shared.Rule{
		UUID:         newUUID(),
		Key:          path,
		Path:         path,
		EndpointAddr: host,
		Action:       action,
		Creation:     time.Now(),
	}
	c := app.Connect()
	defer c.Close()
	if err := c.AddRule(r); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("added rule %s for %s\n", r.UUID, path)
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: lulu <command>

  activate                 submit the system-extension activation request
  deactivate               submit the deactivation request
  list                     list rules from the daemon
  allow <path> [host]      add an allow rule
  block <path> [host]      add a block rule
  delete <key> <uuid>      delete a rule
`)
	os.Exit(2)
}
