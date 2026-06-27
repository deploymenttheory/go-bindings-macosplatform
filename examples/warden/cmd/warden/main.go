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
	"errors"
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

// errUsage signals that arguments were malformed; main prints usage and exits 2.
var errUsage = errors.New("usage")

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, errUsage) {
			fmt.Fprint(os.Stderr, usageText)
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "warden:", err)
		os.Exit(1)
	}
}

// run dispatches one CLI command. Each branch returns an error rather than exiting,
// so main is the single place that decides exit codes.
func run(args []string) error {
	if len(args) < 1 {
		return errUsage
	}
	switch args[0] {
	case "activate":
		if err := app.ActivateExtension(extensionBundleID); err != nil {
			return err
		}
		fmt.Println("submitted system-extension activation request for", extensionBundleID)
		return nil
	case "deactivate":
		if err := app.DeactivateExtension(extensionBundleID); err != nil {
			return err
		}
		fmt.Println("submitted system-extension deactivation request")
		return nil
	case "list":
		return listRules()
	case "allow":
		return addRule(shared.RuleStateAllow, args[1:])
	case "block":
		return addRule(shared.RuleStateBlock, args[1:])
	case "delete":
		if len(args) < 3 {
			return errUsage
		}
		return deleteRule(args[1], args[2])
	case "apply":
		if len(args) < 2 {
			return errUsage
		}
		return applyConfig(args[1])
	case "export":
		if len(args) < 2 {
			return errUsage
		}
		return exportConfig(args[1])
	default:
		return errUsage
	}
}

// daemonStore adapts the XPC client to config.RuleStore so the declarative
// reconciler drives the live daemon. All() returns a one-shot snapshot fetched
// before reconciliation; Add/Delete are sent to the daemon over XPC. config.RuleStore
// has no error return, so the first Add failure is captured in err for the caller
// to surface after Apply.
type daemonStore struct {
	c        app.Client
	snapshot []*shared.Rule
	err      error
}

func (d *daemonStore) All() []*shared.Rule { return d.snapshot }

func (d *daemonStore) Add(r *shared.Rule) {
	if err := d.c.AddRule(r); err != nil && d.err == nil {
		d.err = err
	}
}

// Delete reports true to mean "delete request sent": the XPC call is one-way, so
// the reconciler counts it as actioned (see Client.DeleteRule).
func (d *daemonStore) Delete(key, uuid string) bool { d.c.DeleteRule(key, uuid); return true }

func flatten(m map[string][]*shared.Rule) []*shared.Rule {
	var out []*shared.Rule
	for _, list := range m {
		out = append(out, list...)
	}
	return out
}

func deleteRule(key, uuid string) error {
	c, err := app.Connect()
	if err != nil {
		return err
	}
	defer c.Close()
	c.DeleteRule(key, uuid)
	fmt.Println("delete requested")
	return nil
}

func applyConfig(file string) error {
	cfg, err := config.Load(file)
	if err != nil {
		return err
	}
	c, err := app.Connect()
	if err != nil {
		return err
	}
	defer c.Close()
	current, err := c.GetRules()
	if err != nil {
		return err
	}
	store := &daemonStore{c: c, snapshot: flatten(current)}
	added, deleted := config.Apply(cfg, store)
	if store.err != nil {
		return store.err
	}
	fmt.Printf("applied %s: +%d -%d rules\n", file, added, deleted)
	return nil
}

func exportConfig(file string) error {
	c, err := app.Connect()
	if err != nil {
		return err
	}
	defer c.Close()
	current, err := c.GetRules()
	if err != nil {
		return err
	}
	rules := flatten(current)
	cfg := config.FromRules(rules)
	format := config.FormatYAML
	if strings.HasSuffix(strings.ToLower(file), ".json") {
		format = config.FormatJSON
	}
	data, err := cfg.Marshal(format)
	if err != nil {
		return err
	}
	if err := os.WriteFile(file, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", file, err)
	}
	fmt.Printf("exported %d rules to %s\n", len(rules), file)
	return nil
}

func listRules() error {
	c, err := app.Connect()
	if err != nil {
		return err
	}
	defer c.Close()
	rules, err := c.GetRules()
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		fmt.Println("(no rules)")
		return nil
	}
	for key, list := range rules {
		fmt.Printf("● %s\n", key)
		for _, r := range list {
			ep := r.EndpointAddr
			if ep == "" {
				ep = "*"
			}
			fmt.Printf("    %s  %s → %s  (%s)\n", r.Action, ep, r.EndpointPort, r.UUID)
		}
	}
	return nil
}

// addRule adds an allow/block rule for args[0] (process path), optionally scoped to
// args[1] (endpoint host).
func addRule(action shared.RuleState, args []string) error {
	if len(args) < 1 {
		return errUsage
	}
	path := args[0]
	host := ""
	if len(args) >= 2 {
		host = args[1]
	}
	uuid, err := newUUID()
	if err != nil {
		return err
	}
	r := &shared.Rule{
		UUID:         uuid,
		Key:          path,
		Path:         path,
		EndpointAddr: host,
		Action:       action,
		Creation:     time.Now(),
	}
	c, err := app.Connect()
	if err != nil {
		return err
	}
	defer c.Close()
	if err := c.AddRule(r); err != nil {
		return err
	}
	fmt.Printf("added rule %s for %s\n", r.UUID, path)
	return nil
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// usageText is printed to stderr when a command is missing or malformed.
const usageText = `usage: warden <command>

  activate                 submit the system-extension activation request
  deactivate               submit the deactivation request
  list                     list rules from the daemon
  allow <path> [host]      add an allow rule
  block <path> [host]      add a block rule
  delete <key> <uuid>      delete a rule
  apply <file>             reconcile the firewall to a JSON/YAML config (idempotent)
  export <file>            write current rules as a JSON/YAML config (.json or .yaml)
`
