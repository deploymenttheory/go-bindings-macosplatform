//go:build darwin

// macosversion reports the current macOS version by querying NSProcessInfo from
// the generated Foundation framework bindings.
//
// When GITHUB_OUTPUT is set the results are written as GitHub Actions step outputs:
//
//	macos_version  – full macOS version string, e.g. "26.5"
//	macos_major    – macOS major version number, e.g. "26"
//	macos_minor    – macOS minor version number, e.g. "5"
//
// When running locally the same key=value pairs are printed to stdout.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/deploymenttheory/go-bindings-macosplatform/bindings/frameworks/foundation"
	objc "github.com/deploymenttheory/go-bindings-macosplatform/internal/objc"
	"github.com/deploymenttheory/go-bindings-macosplatform/internal/pureobjc"
)

func main() {
	info := foundation.NSProcessInfoProcessInfo()
	if info == nil {
		fmt.Fprintln(os.Stderr, "NSProcessInfo.processInfo returned nil")
		os.Exit(1)
	}
	nsStr := info.OperatingSystemVersionString()
	if nsStr == nil {
		fmt.Fprintln(os.Stderr, "operatingSystemVersionString returned nil")
		os.Exit(1)
	}
	raw := pureobjc.GoString(nsStr.Ptr())
	objc.KeepAlive(nsStr)

	// NSProcessInfo returns e.g. "Version 26.5 (Build 26A5252d)" — extract "26.5".
	re := regexp.MustCompile(`Version\s+(\d+)\.(\d+)`)
	m := re.FindStringSubmatch(raw)
	if m == nil {
		fmt.Fprintf(os.Stderr, "could not parse version from NSProcessInfo string: %q\n", raw)
		os.Exit(1)
	}
	major, minor := m[1], m[2]
	version := major + "." + minor

	parts := strings.SplitN(version, ".", 2)
	emit("macos_version", version)
	emit("macos_major", parts[0])
	if len(parts) > 1 {
		emit("macos_minor", parts[1])
	}

	fmt.Printf("macOS version : %s (major %s, minor %s)\n", version, major, minor)
}

// emit writes a key=value pair to GITHUB_OUTPUT (if set) and always to stdout.
func emit(key, value string) {
	if path := os.Getenv("GITHUB_OUTPUT"); path != "" {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintf(f, "%s=%s\n", key, value)
			f.Close()
		}
	}
}
