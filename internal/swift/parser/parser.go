//go:build darwin

package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/swift/meta"
)

// Parse locates the arm64e .swiftinterface file for the named framework under
// sdkPath, reads it, and returns populated framework metadata.
// Returns an error if no .swiftinterface file is found.
func Parse(framework, sdkPath string) (*meta.FrameworkMeta, error) {
	path, err := findSwiftInterface(framework, sdkPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("swift parser: read %s: %w", path, err)
	}
	return ParseContent(framework, string(data))
}

// findSwiftInterface returns the path to the best available .swiftinterface file
// for the given framework. Prefers arm64e, falls back to x86_64.
func findSwiftInterface(framework, sdkPath string) (string, error) {
	base := filepath.Join(sdkPath, "System", "Library", "Frameworks",
		framework+".framework", "Modules", framework+".swiftmodule")

	for _, arch := range []string{"arm64e-apple-macos", "arm64-apple-macos", "x86_64-apple-macos"} {
		p := filepath.Join(base, arch+".swiftinterface")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Glob for any .swiftinterface in case naming differs.
	matches, _ := filepath.Glob(filepath.Join(base, "*.swiftinterface"))
	for _, m := range matches {
		if filepath.Ext(m) == ".swiftinterface" {
			return m, nil
		}
	}

	return "", fmt.Errorf("swift parser: no .swiftinterface found for %s under %s", framework, sdkPath)
}
