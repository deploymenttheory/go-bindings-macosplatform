//go:build darwin

// Command warden is the Warden controlling app (CLI form): it activates/deactivates
// the network system extension and manages firewall rules over XPC. Mirrors the
// control surface of Warden's App/ target.
//
// Usage:
//
//	warden activate                 # submit the system-extension activation request
//	warden deactivate               # submit the deactivation request
//	warden list                     # list rules from the daemon
//	warden allow <path> [host]      # add an allow rule for a process (+ optional endpoint)
//	warden block <path> [host]      # add a block rule
//	warden delete <key> <uuid>      # delete a rule
//	warden apply <file.yaml|json>   # declaratively reconcile the firewall to a config
//	warden export <file.yaml|json>  # write the current rules as a config document
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/deploymenttheory/go-bindings-macosplatform/examples/warden/app"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/warden/config"
	"github.com/deploymenttheory/go-bindings-macosplatform/examples/warden/shared"
)

// extensionBundleID is the network extension's bundle identifier. A real build
// uses the team-prefixed identifier embedded in the signed app.
const extensionBundleID = "com.example.warden.extension"

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
	case "apply":
		if len(os.Args) < 3 {
			usage()
		}
		applyConfig(os.Args[2])
	case "export":
		if len(os.Args) < 3 {
			usage()
		}
		exportConfig(os.Args[2])
	default:
		usage()
	}
}

// daemonStore adapts the XPC client to config.RuleStore so the declarative
// reconciler drives the live daemon. All() returns a one-shot snapshot fetched
// before reconciliation; Add/Delete are sent to the daemon over XPC.
type daemonStore struct {
	c        app.Client
	snapshot []*shared.Rule
}

func (d *daemonStore) All() []*shared.Rule          { return d.snapshot }
func (d *daemonStore) Add(r *shared.Rule)           { _ = d.c.AddRule(r) }
func (d *daemonStore) Delete(key, uuid string) bool { d.c.DeleteRule(key, uuid); return true }

func flatten(m map[string][]*shared.Rule) []*shared.Rule {
	var out []*shared.Rule
	for _, list := range m {
		out = append(out, list...)
	}
	return out
}

func applyConfig(file string) {
	cfg, err := config.Load(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	c := app.Connect()
	defer c.Close()
	current, err := c.GetRules()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	store := &daemonStore{c: c, snapshot: flatten(current)}
	added, deleted := config.Apply(cfg, store)
	fmt.Printf("applied %s: +%d -%d rules\n", file, added, deleted)
}

func exportConfig(file string) {
	c := app.Connect()
	defer c.Close()
	current, err := c.GetRules()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	cfg := config.FromRules(flatten(current))
	format := config.FormatYAML
	if strings.HasSuffix(strings.ToLower(file), ".json") {
		format = config.FormatJSON
	}
	data, err := cfg.Marshal(format)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(file, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("exported %d rules to %s\n", len(flatten(current)), file)
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
	fmt.Fprint(os.Stderr, `usage: warden <command>

  activate                 submit the system-extension activation request
  deactivate               submit the deactivation request
  list                     list rules from the daemon
  allow <path> [host]      add an allow rule
  block <path> [host]      add a block rule
  delete <key> <uuid>      delete a rule
  apply <file>             reconcile the firewall to a JSON/YAML config (idempotent)
  export <file>            write current rules as a JSON/YAML config (.json or .yaml)
`)
	os.Exit(2)
}
