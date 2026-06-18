//go:build darwin

// Command luludaemon is the LuLu network-extension daemon: it registers the
// NEFilterDataProvider subclass, vends the XPC control service, and runs the
// system-extension run loop. macOS only loads this as a system extension from a
// signed .systemextension bundle (see examples/lulu/README.md).
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/deploymenttheory/go-bindings-macosplatform/examples/lulu/extension"
)

func main() {
	rulesPath := filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "LuLu", "rules.json")
	if v := os.Getenv("LULU_RULES"); v != "" {
		rulesPath = v
	}
	if err := os.MkdirAll(filepath.Dir(rulesPath), 0o755); err != nil {
		log.Fatalf("luludaemon: %v", err)
	}
	// LULU_CONFIG, if set, names a declarative JSON/YAML config the daemon
	// reconciles its rule set against at startup.
	configPath := os.Getenv("LULU_CONFIG")
	log.Printf("luludaemon: starting (rules=%s config=%q)", rulesPath, configPath)
	if err := extension.Run(rulesPath, configPath); err != nil {
		log.Fatalf("luludaemon: %v", err)
	}
}
