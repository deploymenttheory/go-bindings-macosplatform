//go:build darwin

// macosversion reports the current macOS version so the acceptance workflow can
// decide whether the committed bindings (generated against one SDK major) are
// exercisable on this runner.
//
// Deliberately does NOT use the generated bindings: this check is what gates
// their use, so it must work on a runner whose OS predates the bindings' SDK
// (where loading them can fail). sw_vers is the narrowest OS-independent probe.
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
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	out, err := exec.Command("/usr/bin/sw_vers", "-productVersion").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sw_vers -productVersion: %v\n", err)
		os.Exit(1)
	}
	raw := strings.TrimSpace(string(out))

	// sw_vers prints e.g. "26.5" or "26.5.1" — extract major and minor.
	re := regexp.MustCompile(`^(\d+)\.(\d+)`)
	m := re.FindStringSubmatch(raw)
	if m == nil {
		fmt.Fprintf(os.Stderr, "could not parse version from sw_vers output: %q\n", raw)
		os.Exit(1)
	}
	major, minor := m[1], m[2]
	version := major + "." + minor

	emit("macos_version", version)
	emit("macos_major", major)
	emit("macos_minor", minor)

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
